# Generates web/static/docs.html from docs/openapi.yaml.
# Self-contained output: all CSS/JS inline, no CDN assets, so the page renders
# behind a strict CSP or with no egress.
#
# Regenerate after editing the spec (from the repo root):
#   python docs/gen_docs_html.py
#
# Requires: pyyaml (pip install pyyaml)

import html
import re
import sys

import yaml

root = sys.argv[1] if len(sys.argv) > 1 else "."

with open(f"{root}/docs/openapi.yaml", encoding="utf-8") as f:
    spec = yaml.safe_load(f)


def esc(s):
    return html.escape(str(s), quote=True)


def md_inline(s):
    """Escape HTML, then apply the few inline markdown forms the spec uses."""
    s = esc(s)
    s = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", s)
    s = re.sub(r"(?<!\*)\*([^*\s][^*]*)\*(?!\*)", r"<em>\1</em>", s)
    s = re.sub(r"`([^`]+)`", r"<code>\1</code>", s)
    s = re.sub(r"\[([^\]]+)\]\(([^)]+)\)", r'<a href="\2">\1</a>', s)
    return s


def md_block(text):
    """Small markdown renderer for info/operation descriptions."""
    out, para, in_ul, in_ol = [], [], False, False

    def flush_para():
        nonlocal para
        if para:
            out.append("<p>" + md_inline(" ".join(para)) + "</p>")
            para = []

    def close_lists():
        nonlocal in_ul, in_ol
        if in_ul:
            out.append("</ul>")
            in_ul = False
        if in_ol:
            out.append("</ol>")
            in_ol = False

    for raw in (text or "").splitlines():
        line = raw.rstrip()
        stripped = line.strip()
        if not stripped:
            # A blank line ends a paragraph but not a list: the spec separates
            # list items with blank lines, and closing here would restart the
            # numbering of every <ol> at 1.
            flush_para()
            continue
        m = re.match(r"^(#{1,4})\s+(.*)$", stripped)
        if m:
            flush_para()
            close_lists()
            lvl = min(len(m.group(1)) + 1, 5)
            out.append(f"<h{lvl}>{md_inline(m.group(2))}</h{lvl}>")
            continue
        if re.match(r"^\d+\.\s+", stripped):
            flush_para()
            if in_ul:
                out.append("</ul>")
                in_ul = False
            if not in_ol:
                out.append("<ol>")
                in_ol = True
            out.append("<li>" + md_inline(re.sub(r"^\d+\.\s+", "", stripped)) + "</li>")
            continue
        if stripped.startswith("- "):
            flush_para()
            if in_ol:
                out.append("</ol>")
                in_ol = False
            if not in_ul:
                out.append("<ul>")
                in_ul = True
            out.append("<li>" + md_inline(stripped[2:]) + "</li>")
            continue
        # Continuation line: append to the open list item or paragraph.
        if (in_ul or in_ol) and raw.startswith(("  ", "\t")) and out and out[-1].endswith("</li>"):
            out[-1] = out[-1][: -len("</li>")] + " " + md_inline(stripped) + "</li>"
            continue
        close_lists()
        para.append(stripped)
    flush_para()
    close_lists()
    return "\n".join(out)


def ref_name(ref):
    return ref.rsplit("/", 1)[-1]


def schema_inline(s, depth=0):
    """Compact one-line description of a schema, with links to named schemas."""
    if s is None:
        return "any"
    if "$ref" in s:
        n = ref_name(s["$ref"])
        return f'<a class="sref" href="#schema-{n}">{esc(n)}</a>'
    if "allOf" in s:
        parts = []
        for sub in s["allOf"]:
            if "$ref" in sub:
                parts.append(schema_inline(sub, depth + 1))
            elif sub.get("type") == "object" and "data" in sub.get("properties", {}):
                parts.append("data: " + schema_inline(sub["properties"]["data"], depth + 1))
            else:
                parts.append(schema_inline(sub, depth + 1))
        return " &middot; ".join(parts)
    t = s.get("type")
    if isinstance(t, list):
        t = " | ".join(str(x) for x in t)
    if t == "array":
        return "array of " + schema_inline(s.get("items"), depth + 1)
    if "enum" in s:
        return f"{t or 'string'}: " + " | ".join(f"<code>{esc(v)}</code>" for v in s["enum"])
    if t == "object" or "properties" in s:
        props = s.get("properties", {})
        if not props:
            return "object"
        if depth >= 2:
            return "object"
        inner = ", ".join(
            f"<code>{esc(k)}</code>: {schema_inline(v, depth + 2)}" for k, v in props.items()
        )
        return "object { " + inner + " }"
    if t:
        fmt = s.get("format")
        r = esc(t)
        if fmt:
            r += f" ({esc(fmt)})"
        return r
    if "const" in s:
        return f"const <code>{esc(s['const'])}</code>"
    return "any"


def object_prop_rows(s):
    """Rows for an object schema's property table; None if not a plain object."""
    props = s.get("properties")
    if not props:
        return None
    required = set(s.get("required", []))
    rows = []
    for name, sub in props.items():
        desc = sub.get("description", "")
        constraints = []
        for key, label in (
            ("minLength", "min length"), ("maxLength", "max length"),
            ("minimum", "min"), ("maximum", "max"), ("pattern", "pattern"),
            ("default", "default"), ("const", "always"),
        ):
            if key in sub:
                constraints.append(f"{label} <code>{esc(sub[key])}</code>")
        c = ("<br><span class='muted'>" + ", ".join(constraints) + "</span>") if constraints else ""
        rows.append(
            "<tr><td><code>{}</code>{}</td><td>{}</td><td>{}{}</td></tr>".format(
                esc(name),
                ' <span class="req">required</span>' if name in required else "",
                schema_inline(sub, 1),
                md_inline(desc) if desc else "",
                c,
            )
        )
    return rows


METHOD_ORDER = ["get", "post", "put", "patch", "delete"]

# Collect operations grouped by tag, preserving the tag order declared in the spec.
tag_meta = {t["name"]: t.get("description", "") for t in spec.get("tags", [])}
tag_order = [t["name"] for t in spec.get("tags", [])]
by_tag = {t: [] for t in tag_order}

for path, item in spec["paths"].items():
    shared_params = item.get("parameters", [])
    path_servers = item.get("servers")
    for method in METHOD_ORDER:
        if method not in item:
            continue
        op = item[method]
        tag = op.get("tags", ["Other"])[0]
        by_tag.setdefault(tag, []).append((path, method, op, shared_params, path_servers))


def resolve_param(p):
    if "$ref" in p:
        return spec["components"]["parameters"][ref_name(p["$ref"])]
    return p


def resolve_response(r):
    if "$ref" in r:
        return spec["components"]["responses"][ref_name(r["$ref"])]
    return r


def op_anchor(method, path):
    return "op-" + method + "-" + re.sub(r"[^a-zA-Z0-9]+", "-", path).strip("-")


def render_params(params):
    if not params:
        return ""
    rows = []
    for p in map(resolve_param, params):
        rows.append(
            "<tr><td><code>{}</code>{}</td><td>{}</td><td>{}</td><td>{}</td></tr>".format(
                esc(p["name"]),
                ' <span class="req">required</span>' if p.get("required") else "",
                esc(p["in"]),
                schema_inline(p.get("schema"), 1),
                md_inline(p.get("description", "")),
            )
        )
    return (
        "<h4>Parameters</h4><table><thead><tr><th>Name</th><th>In</th>"
        "<th>Type</th><th>Description</th></tr></thead><tbody>"
        + "".join(rows) + "</tbody></table>"
    )


def render_body(body):
    if not body:
        return ""
    out = ["<h4>Request body{}</h4>".format(
        "" if body.get("required") else ' <span class="muted">(optional)</span>')]
    for ctype, media in body.get("content", {}).items():
        out.append(f'<div class="ctype"><code>{esc(ctype)}</code></div>')
        schema = media.get("schema", {})
        rows = object_prop_rows(schema)
        if rows:
            out.append(
                "<table><thead><tr><th>Field</th><th>Type</th><th>Description</th>"
                "</tr></thead><tbody>" + "".join(rows) + "</tbody></table>"
            )
        else:
            out.append(f"<p>{schema_inline(schema)}</p>")
    return "".join(out)


def render_responses(responses):
    out = ["<h4>Responses</h4><table><thead><tr><th>Status</th><th>Description</th>"
           "<th>Body</th></tr></thead><tbody>"]
    for status, r in responses.items():
        r = resolve_response(r)
        bodies = []
        for ctype, media in (r.get("content") or {}).items():
            bodies.append(f"<code>{esc(ctype)}</code> &mdash; {schema_inline(media.get('schema'))}")
        out.append(
            "<tr><td><span class='status s{}'>{}</span></td><td>{}</td><td>{}</td></tr>".format(
                str(status)[0], esc(status),
                md_inline(r.get("description", "")),
                "<br>".join(bodies) or "<span class='muted'>empty</span>",
            )
        )
    out.append("</tbody></table>")
    return "".join(out)


sections = []
nav = []
for tag in tag_order:
    ops = by_tag.get(tag, [])
    if not ops:
        continue
    nav.append(f'<div class="nav-tag">{esc(tag)}</div>')
    body = [f'<section class="tag" id="tag-{esc(tag)}"><h2>{esc(tag)}</h2>']
    if tag_meta.get(tag):
        body.append(f"<p class='tagdesc'>{md_inline(tag_meta[tag])}</p>")
    for path, method, op, shared_params, path_servers in ops:
        anchor = op_anchor(method, path)
        nav.append(
            '<a class="nav-op" href="#{a}" data-text="{m} {p} {s}">'
            '<span class="m m-{m}">{mu}</span><span class="np">{p}</span></a>'.format(
                a=anchor, m=method, mu=method.upper(), p=esc(path),
                s=esc(op.get("summary", "").lower()),
            )
        )
        sec = op.get("security", spec.get("security"))
        needs_auth = bool(sec) and any(s for s in sec)
        auth_badge = (
            '<span class="badge auth">requires bearer token</span>' if needs_auth
            else '<span class="badge open">no auth required</span>'
        )
        root_note = (
            '<span class="badge root">server root (not /api/v1)</span>' if path_servers else ""
        )
        body.append(f'<article class="op" id="{anchor}">')
        body.append(
            f'<h3><span class="m m-{method}">{method.upper()}</span> '
            f'<code class="path">{esc(path)}</code></h3>'
        )
        body.append(f'<p class="summary">{md_inline(op.get("summary", ""))} {auth_badge} {root_note}</p>')
        if op.get("description"):
            body.append(f'<div class="desc">{md_block(op["description"])}</div>')
        body.append(render_params(shared_params + op.get("parameters", [])))
        body.append(render_body(op.get("requestBody")))
        body.append(render_responses(op.get("responses", {})))
        body.append("</article>")
    body.append("</section>")
    sections.append("".join(body))

# Schemas section
schema_html = ['<section class="tag" id="schemas"><h2>Schemas</h2>']
nav.append('<div class="nav-tag">Schemas</div>')
for name, s in spec["components"]["schemas"].items():
    nav.append(
        f'<a class="nav-op" href="#schema-{name}" data-text="{name.lower()}">'
        f'<span class="np">{esc(name)}</span></a>'
    )
    schema_html.append(f'<article class="op" id="schema-{name}"><h3><code>{esc(name)}</code></h3>')
    if s.get("description"):
        schema_html.append(f"<p class='summary'>{md_inline(s['description'])}</p>")
    target = s
    prefix = ""
    if "allOf" in s:
        parts = [schema_inline(sub) for sub in s["allOf"] if "$ref" in sub]
        if parts:
            prefix = "Extends " + ", ".join(parts)
        for sub in s["allOf"]:
            if sub.get("properties"):
                target = sub
                break
    if prefix:
        schema_html.append(f"<p class='muted'>{prefix}</p>")
    rows = object_prop_rows(target)
    if rows:
        schema_html.append(
            "<table><thead><tr><th>Field</th><th>Type</th><th>Description</th></tr></thead>"
            "<tbody>" + "".join(rows) + "</tbody></table>"
        )
    elif "enum" in s:
        schema_html.append("<p>" + schema_inline(s) + "</p>")
    schema_html.append("</article>")
schema_html.append("</section>")
sections.append("".join(schema_html))

info = spec["info"]
servers_rows = "".join(
    f"<tr><td><code>{esc(s['url'])}</code></td><td>{md_inline(s.get('description', ''))}</td></tr>"
    for s in spec.get("servers", [])
)

page = f"""<!DOCTYPE html>
<!-- GENERATED from docs/openapi.yaml — do not edit by hand.
     Regenerate from the repo root with: python docs/gen_docs_html.py
     This page is deliberately self-contained: no CDN scripts or styles. -->
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark">
<title>{esc(info["title"])} — API Reference</title>
<style>
:root {{
  --bg: #ffffff; --fg: #1a1d21; --muted: #667085; --line: #e4e7ec;
  --panel: #f8fafc; --code-bg: #eef1f5; --accent: #2563eb;
  --get: #2563eb; --post: #16a34a; --put: #d97706; --patch: #9333ea; --delete: #dc2626;
}}
@media (prefers-color-scheme: dark) {{
  :root {{
    --bg: #0f1216; --fg: #e6e9ee; --muted: #98a2b3; --line: #262c35;
    --panel: #161b22; --code-bg: #1e252e; --accent: #60a5fa;
    --get: #60a5fa; --post: #4ade80; --put: #fbbf24; --patch: #c084fc; --delete: #f87171;
  }}
}}
* {{ box-sizing: border-box; }}
body {{ margin: 0; font: 15px/1.55 -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  background: var(--bg); color: var(--fg); }}
code {{ font-family: ui-monospace, "Cascadia Code", Consolas, monospace; font-size: .92em;
  background: var(--code-bg); padding: .1em .35em; border-radius: 4px; }}
a {{ color: var(--accent); text-decoration: none; }}
a:hover {{ text-decoration: underline; }}
.layout {{ display: flex; min-height: 100vh; }}
nav {{ width: 300px; flex: none; border-right: 1px solid var(--line); padding: 1rem .75rem;
  position: sticky; top: 0; height: 100vh; overflow-y: auto; background: var(--panel); }}
nav .brand {{ font-weight: 700; margin: 0 .25rem .75rem; }}
nav input {{ width: 100%; padding: .45rem .6rem; margin-bottom: .75rem; border: 1px solid var(--line);
  border-radius: 6px; background: var(--bg); color: var(--fg); }}
.nav-tag {{ font-size: .72rem; font-weight: 700; text-transform: uppercase; letter-spacing: .06em;
  color: var(--muted); margin: .9rem .25rem .25rem; }}
.nav-op {{ display: flex; align-items: baseline; gap: .45rem; padding: .18rem .35rem;
  border-radius: 5px; color: var(--fg); font-size: .82rem; overflow: hidden; }}
.nav-op:hover {{ background: var(--code-bg); text-decoration: none; }}
.nav-op .np {{ white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  font-family: ui-monospace, Consolas, monospace; }}
main {{ flex: 1; min-width: 0; padding: 2rem clamp(1rem, 4vw, 3.5rem) 5rem; max-width: 70rem; }}
h1 {{ margin-top: 0; }}
h2 {{ border-bottom: 1px solid var(--line); padding-bottom: .35rem; margin-top: 3rem; }}
h4 {{ margin: 1.1rem 0 .4rem; color: var(--muted); font-size: .8rem;
  text-transform: uppercase; letter-spacing: .05em; }}
.m {{ font-family: ui-monospace, Consolas, monospace; font-weight: 700; font-size: .72rem;
  padding: .15rem .45rem; border-radius: 4px; color: #fff; }}
.m-get {{ background: var(--get); }} .m-post {{ background: var(--post); }}
.m-put {{ background: var(--put); }} .m-patch {{ background: var(--patch); }}
.m-delete {{ background: var(--delete); }}
nav .m {{ font-size: .6rem; padding: .05rem .3rem; min-width: 2.6rem; text-align: center; flex: none; }}
.op {{ border: 1px solid var(--line); border-radius: 10px; padding: 1rem 1.25rem;
  margin: 1.25rem 0; background: var(--panel); overflow-x: auto; }}
.op h3 {{ margin: 0 0 .3rem; display: flex; align-items: center; gap: .6rem; flex-wrap: wrap; }}
.path {{ background: transparent; font-size: 1.02rem; padding: 0; }}
.summary {{ margin: .2rem 0 .6rem; }}
.badge {{ font-size: .7rem; padding: .12rem .5rem; border-radius: 99px; border: 1px solid var(--line);
  vertical-align: middle; white-space: nowrap; }}
.badge.auth {{ color: var(--put); border-color: var(--put); }}
.badge.open {{ color: var(--muted); }}
.badge.root {{ color: var(--patch); border-color: var(--patch); }}
.req {{ color: var(--delete); font-size: .7rem; font-weight: 600; }}
.muted {{ color: var(--muted); }}
table {{ border-collapse: collapse; width: 100%; margin: .3rem 0 .8rem; font-size: .86rem; }}
th, td {{ text-align: left; padding: .4rem .6rem; border-bottom: 1px solid var(--line);
  vertical-align: top; }}
th {{ color: var(--muted); font-weight: 600; }}
.status {{ font-family: ui-monospace, Consolas, monospace; font-weight: 700; }}
.s2 {{ color: var(--post); }} .s3 {{ color: var(--accent); }}
.s4 {{ color: var(--put); }} .s5 {{ color: var(--delete); }}
.ctype {{ margin: .2rem 0; }}
.intro {{ max-width: 52rem; }}
.intro ol li, .intro ul li {{ margin: .4rem 0; }}
.desc p {{ margin: .4rem 0; }}
@media (max-width: 860px) {{
  .layout {{ flex-direction: column; }}
  nav {{ position: static; width: 100%; height: auto; max-height: 40vh; }}
}}
</style>
</head>
<body>
<div class="layout">
<nav>
  <div class="brand">{esc(info["title"])}</div>
  <input id="filter" type="search" placeholder="Filter endpoints..." aria-label="Filter endpoints">
  {"".join(nav)}
</nav>
<main>
  <h1>{esc(info["title"])}</h1>
  <p class="muted">Version {esc(info["version"])} &middot; OpenAPI {esc(spec["openapi"])} &middot;
    raw spec: <a href="/openapi.yaml"><code>/openapi.yaml</code></a></p>
  <div class="intro">{md_block(info.get("description", ""))}</div>
  <h2>Servers</h2>
  <table><thead><tr><th>URL</th><th>Description</th></tr></thead><tbody>{servers_rows}</tbody></table>
  {"".join(sections)}
</main>
</div>
<script>
(function () {{
  var input = document.getElementById('filter');
  var ops = Array.prototype.slice.call(document.querySelectorAll('.nav-op'));
  var tags = Array.prototype.slice.call(document.querySelectorAll('.nav-tag'));
  input.addEventListener('input', function () {{
    var q = input.value.toLowerCase();
    ops.forEach(function (a) {{
      var t = (a.getAttribute('data-text') || '') + ' ' + a.textContent.toLowerCase();
      a.style.display = t.indexOf(q) !== -1 ? '' : 'none';
    }});
    tags.forEach(function (h) {{
      var el = h.nextElementSibling, any = false;
      while (el && !el.classList.contains('nav-tag')) {{
        if (el.style.display !== 'none') any = true;
        el = el.nextElementSibling;
      }}
      h.style.display = any ? '' : 'none';
    }});
  }});
}})();
</script>
</body>
</html>
"""

out_path = f"{root}/web/static/docs.html"
with open(out_path, "w", encoding="utf-8") as f:
    f.write(page)
print("wrote", out_path, len(page), "bytes")
