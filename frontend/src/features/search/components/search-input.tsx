"use client";

import { Command as CommandPrimitive } from "cmdk";
import { Search, X } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, useSyncExternalStore } from "react";

import { Command, CommandGroup, CommandItem, CommandList } from "@/components/ui/command";
import { routes } from "@/config/routes";
import { getSuggestions } from "@/features/search/actions";
import { useDebounce } from "@/hooks/use-debounce";
import { cn } from "@/lib/utils";

interface SearchInputProps {
  /**
   * Seeds the field, e.g. with the current `?q=` on the search page. This is an
   * *initial* value, not a controlled one: pass `key={q}` alongside it if the
   * field must reset when the URL's query changes (the search page does). That
   * is React's own answer to "resync state from a prop" — cheaper and less
   * bug-prone than an effect that writes state on every prop change.
   */
  initialQuery?: string;
  autoFocus?: boolean;
  /**
   * Claim ⌘K / Ctrl+K for this field. Off by default, and it MUST stay that way:
   * the shortcut is a single document-level listener, so if two mounted
   * SearchInputs both claimed it they would both focus on the same keypress and
   * whichever mounted last would silently win.
   *
   * Exactly one input claims it — the one in the header (`HeaderSearch`), which
   * is mounted on every page and is therefore the only field a global shortcut
   * can honestly promise to reach. The search page's own fields do not: the hero
   * already autofocuses, and the mobile field is only visible where there is no
   * keyboard to press ⌘K with.
   */
  shortcut?: boolean;
  className?: string;
}

/** `navigator` cannot be read on the server, and never changes on the client. */
const subscribeToNothing = () => () => {};
const readShortcutHint = () => (/mac|iphone|ipad/i.test(navigator.platform) ? "⌘K" : "Ctrl K");

/**
 * The search box: a combobox over `GET /search/suggest`. Typing is debounced
 * 200ms before a request fires; ↑/↓ move through suggestions (cmdk), Enter
 * picks — the first row is always "search for what I typed", so Enter never
 * hijacks a literal query — and Esc closes. ⌘K / Ctrl+K focuses it from
 * anywhere on the page.
 *
 * The suggestion panel deliberately does not animate: it appears on every
 * keystroke, and per-keystroke motion reads as lag, not polish.
 */
export function SearchInput({ initialQuery, autoFocus, shortcut = false, className }: SearchInputProps) {
  const router = useRouter();
  const inputRef = useRef<HTMLInputElement>(null);

  const [value, setValue] = useState(initialQuery ?? "");
  const [open, setOpen] = useState(false);
  /**
   * Suggestions are stored together with the query they were fetched for.
   * Keeping the pair in one piece of state is what lets the visible list be
   * *derived* rather than cleared by an effect — a synchronous setState inside
   * an effect costs a second render pass on every keystroke.
   */
  const [fetched, setFetched] = useState<{ query: string; items: string[] }>({ query: "", items: [] });

  // Platform-specific, so it cannot be known during SSR: the server snapshot is
  // null (render no hint) and the real value swaps in at hydration.
  const shortcutHint = useSyncExternalStore(subscribeToNothing, readShortcutHint, () => null);

  // ⌘K / Ctrl+K focuses the field from anywhere. No animation: a keyboard
  // shortcut is used constantly, and motion would only make it feel slow.
  // Only the field that claimed the shortcut listens — see the `shortcut` prop.
  useEffect(() => {
    if (!shortcut) return;

    function onKeyDown(event: KeyboardEvent) {
      if (event.key.toLowerCase() === "k" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        inputRef.current?.focus();
        inputRef.current?.select();
      }
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [shortcut]);

  // One request per settled query, never one per keystroke. Nothing is set
  // synchronously here — the only setState runs in the resolved callback.
  const debounced = useDebounce(value, 200);
  useEffect(() => {
    const query = debounced.trim();
    if (query.length < 2) return;

    let stale = false;
    void getSuggestions(query).then((items) => {
      if (!stale) setFetched({ query, items });
    });
    return () => {
      stale = true;
    };
  }, [debounced]);

  const trimmed = value.trim();
  const showList = open && trimmed.length > 0;

  /**
   * cmdk hardcodes `aria-expanded={true}` on its input — permanently, whatever
   * the listbox is actually doing — and it spreads consumer props BEFORE its own
   * attributes, so the prop cannot be overridden from the outside. Left alone,
   * this is a combobox that tells every screen reader it is open at all times,
   * including on a blank search box with no panel on screen.
   *
   * So it is set on the element instead. This is stable, not a race: cmdk's own
   * value never changes, so React writes `aria-expanded` exactly once at mount
   * and never reconciles it again — which means the value written here survives
   * every subsequent render.
   */
  useEffect(() => {
    inputRef.current?.setAttribute("aria-expanded", String(showList));
  }, [showList]);

  /**
   * Derived, not stored. Below two characters there is nothing to suggest, so
   * the list empties without an effect ever touching state. Above it, the last
   * fetched list stays up while the next one is in flight — an autocomplete
   * that blanks between keystrokes flickers.
   */
  const suggestions =
    trimmed.length >= 2
      ? // The literal query is already row one; repeating it below is noise.
        [...new Set(fetched.items)].filter((suggestion) => suggestion !== trimmed)
      : [];

  /**
   * What the live region says. Keyed off `debounced`, not `value`, so it speaks
   * once the typing settles rather than once per character — a live region wired
   * to a per-keystroke value talks over the person using it.
   */
  const settled = debounced.trim();
  const announcement =
    showList && settled.length >= 2 && settled === fetched.query
      ? suggestions.length === 0
        ? "No suggestions."
        : `${suggestions.length} ${suggestions.length === 1 ? "suggestion" : "suggestions"} available.`
      : "";

  function submit(term: string) {
    const query = term.trim();
    if (!query) return;
    setValue(query);
    setOpen(false);
    inputRef.current?.blur();
    router.push(`${routes.search}?q=${encodeURIComponent(query)}`);
  }

  return (
    <Command
      shouldFilter={false}
      className={cn("relative overflow-visible rounded-none! bg-transparent p-0", className)}
      onKeyDown={(event) => {
        if (event.key === "Escape" && open) {
          event.preventDefault();
          setOpen(false);
          return;
        }
        // When the list is closed, Enter searches the literal text — cmdk only
        // owns Enter while suggestions are showing.
        if (event.key === "Enter" && !showList && !event.nativeEvent.isComposing) {
          event.preventDefault();
          submit(value);
        }
      }}
    >
      <div
        className={cn(
          "flex h-11 items-center gap-2.5 rounded-full border border-input bg-muted/40 pr-2 pl-4",
          "transition-colors duration-(--motion-fast)",
          "focus-within:border-ring/60 focus-within:ring-3 focus-within:ring-ring/40",
        )}
      >
        <Search aria-hidden className="size-4 shrink-0 text-muted-foreground" />
        <CommandPrimitive.Input
          ref={inputRef}
          value={value}
          onValueChange={(next) => {
            setValue(next);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          onBlur={() => setOpen(false)}
          placeholder="Search videos"
          aria-label="Search videos"
          enterKeyHint="search"
          autoFocus={autoFocus}
          className="h-full min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
        />
        {value ? (
          <button
            type="button"
            aria-label="Clear search"
            // Keep focus in the input: preventDefault stops the blur that would
            // otherwise close the list before the click lands.
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => {
              setValue("");
              inputRef.current?.focus();
            }}
            className="flex size-8 shrink-0 items-center justify-center rounded-full text-muted-foreground outline-none transition-colors duration-(--motion-fast) hover:bg-muted hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50"
          >
            <X aria-hidden className="size-4" />
          </button>
        ) : shortcut && shortcutHint ? (
          // Only the field that actually answers ⌘K advertises it. A hint on a
          // field the shortcut does not focus is a lie the user finds out about.
          <kbd
            aria-hidden
            className="mr-1.5 hidden shrink-0 rounded-md border border-border/70 bg-muted/60 px-1.5 py-0.5 font-sans text-[11px] leading-4 text-muted-foreground select-none sm:block"
          >
            {shortcutHint}
          </kbd>
        ) : null}
      </div>

      {/*
        Hidden, never unmounted. cmdk's input advertises `aria-controls` pointing
        at this list's id; unmounting the list left that attribute dangling at an
        element that does not exist, which is a broken combobox as far as a
        screen reader is concerned. `hidden` keeps the id resolvable and costs
        nothing — the panel is three rows of text.
      */}
      <div
        // Mousedown on the panel must not blur the input mid-click.
        onMouseDown={(event) => event.preventDefault()}
        className={cn(
          "absolute inset-x-0 top-full z-50 mt-2 overflow-hidden rounded-xl bg-popover text-popover-foreground shadow-border-overlay",
          !showList && "hidden",
        )}
      >
        <CommandList>
          <CommandItem value="__literal-query__" onSelect={() => submit(value)} className="rounded-lg">
            <Search aria-hidden className="size-4 text-muted-foreground" />
            <span className="truncate">
              Search for <span className="font-medium">&ldquo;{trimmed}&rdquo;</span>
            </span>
          </CommandItem>
          {suggestions.length > 0 ? (
            <CommandGroup heading="Suggestions">
              {suggestions.map((suggestion) => (
                <CommandItem
                  key={suggestion}
                  value={suggestion}
                  onSelect={() => submit(suggestion)}
                  className="rounded-lg"
                >
                  <Search aria-hidden className="size-4 text-muted-foreground/70" />
                  <span className="truncate">{suggestion}</span>
                </CommandItem>
              ))}
            </CommandGroup>
          ) : null}
        </CommandList>
      </div>

      {/*
        The panel opening is silent otherwise — a sighted user sees rows appear
        under the caret and a screen-reader user gets nothing. Announced off the
        DEBOUNCED query, not the raw one, so it reports settled results instead
        of re-interrupting the typist on every keystroke.
      */}
      <span role="status" aria-live="polite" className="sr-only">
        {announcement}
      </span>
    </Command>
  );
}
