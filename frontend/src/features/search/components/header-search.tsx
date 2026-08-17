"use client";

import { useSearchParams } from "next/navigation";
import { Suspense } from "react";

import { SearchInput } from "@/features/search/components/search-input";

/**
 * The app-wide search box, as mounted in the header by every shell layout.
 *
 * It exists as a wrapper for two reasons, both of which the bare `SearchInput`
 * cannot solve from inside a layout:
 *
 * 1. SEEDING. A layout is not handed `searchParams` — only pages are. So the
 *    header cannot be told the current query from the server; it has to read it.
 *    Reading it matters: landing on `/search?q=otters` with an empty header box
 *    means the field next to the results does not contain the search that
 *    produced them, and refining the query means retyping it.
 *
 * 2. RESYNCING. `SearchInput` takes `initialQuery` as an *initial* value, not a
 *    controlled one. The header never unmounts as you navigate, so a plain prop
 *    would go stale the moment `?q=` changed. `key={q}` is React's own answer —
 *    a new key is a new component instance with fresh state — and it is why the
 *    query is read here rather than inside the input with an effect that writes
 *    state on every change.
 *
 * This is also the one field that claims ⌘K (`shortcut`): it is on every page,
 * so it is the only one a global shortcut can honestly promise to reach.
 */
function HeaderSearchField() {
  const q = useSearchParams().get("q") ?? "";
  return <SearchInput key={q} initialQuery={q} shortcut />;
}

/**
 * `useSearchParams()` opts its subtree into client rendering, so it needs a
 * Suspense boundary above it. The fallback is the field's own footprint — an
 * inert pill of exactly the same height — so the header never reflows around it.
 */
export function HeaderSearch() {
  return (
    <Suspense
      fallback={
        <div
          aria-hidden
          className="h-11 w-full rounded-full border border-input bg-muted/40"
        />
      }
    >
      <HeaderSearchField />
    </Suspense>
  );
}
