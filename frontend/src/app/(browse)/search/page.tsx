import { SearchX, Telescope } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { Pagination } from "@/components/common/pagination";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { getCategories, search } from "@/features/search/api";
import { CategoryChips } from "@/features/search/components/category-chips";
import { SearchFilters } from "@/features/search/components/search-filters";
import { SearchInput } from "@/features/search/components/search-input";
import { SearchResults } from "@/features/search/components/search-results";
import { TrendingRail, TrendingRailSkeleton } from "@/features/search/components/trending-rail";
import { isSearchSort, type SearchQuery } from "@/features/search/types";
import { isApiError } from "@/lib/api-error";
import { formatCompact } from "@/lib/format";
import type { Page, VideoSearchItem } from "@/types/common";

/** searchParams values arrive as string | string[] — a repeated param takes its first value. */
function first(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

/** Non-negative integer or nothing — never forward garbage the API will 400 on. */
function toCount(value: string | undefined, minimum = 0): number | undefined {
  if (value === undefined) return undefined;
  const parsed = Number.parseInt(value, 10);
  return Number.isInteger(parsed) && parsed >= minimum ? parsed : undefined;
}

interface ParsedParams {
  q: string;
  query: SearchQuery;
  hasFilters: boolean;
}

function parse(sp: Record<string, string | string[] | undefined>): ParsedParams {
  const q = first(sp.q)?.trim() ?? "";
  const rawSort = first(sp.sort);
  const sort = rawSort && isSearchSort(rawSort) ? rawSort : undefined;
  const category = first(sp.category)?.trim() || undefined;
  const language = first(sp.language)?.trim() || undefined;
  const tags = first(sp.tags)?.trim() || undefined;
  const minDuration = toCount(first(sp.min_duration));
  const maxDuration = toCount(first(sp.max_duration));

  return {
    q,
    query: {
      q,
      sort,
      category,
      language,
      tags,
      minDuration,
      maxDuration,
      page: toCount(first(sp.page), 1),
      limit: 20,
    },
    hasFilters: Boolean(sort || category || language || tags || minDuration !== undefined || maxDuration !== undefined),
  };
}

export async function generateMetadata(props: PageProps<"/search">): Promise<Metadata> {
  const { q } = parse(await props.searchParams);
  return { title: q ? `${q} — Search` : "Search" };
}

export default async function SearchPage(props: PageProps<"/search">) {
  const sp = await props.searchParams;
  const { q, query, hasFilters } = parse(sp);

  // The filter bar and chips survive an empty result set or a failed search,
  // so the categories fetch must never take the page down with it.
  const categoriesPromise = getCategories().catch(() => []);

  /* ------------------------------------------------------------------ */
  /* No query yet: a designed starting point, not a blank results page.  */
  /* ------------------------------------------------------------------ */
  if (!q) {
    const categories = await categoriesPromise;
    return (
      // A <div>, not a <main>: the browse layout already owns the page's <main>
      // landmark, and nesting a second one leaves a screen reader with two.
      <div className="mx-auto w-full max-w-5xl flex-1 px-4 py-8 sm:px-6">
        <div className="mx-auto max-w-2xl pt-6 text-center sm:pt-12">
          <div className="mx-auto mb-5 flex size-14 items-center justify-center rounded-2xl bg-muted text-muted-foreground ring-1 ring-border/60 ring-inset">
            <Telescope aria-hidden className="size-6" />
          </div>
          <h1 className="text-title text-balance">Find something to watch</h1>
          <p className="mt-2 text-sm text-pretty text-muted-foreground">
            Search across every public video — by title, description or tag.
          </p>
          <SearchInput autoFocus className="mt-6 text-left" />
        </div>

        <CategoryChips categories={categories} className="mt-12" />

        {/* The rail is a secondary read — streaming it keeps the search field
            interactive while trending is still in flight. */}
        <Suspense fallback={<TrendingRailSkeleton className="mt-8" />}>
          <TrendingRail window="7d" title="Trending this week" className="mt-8" />
        </Suspense>
      </div>
    );
  }

  /* ------------------------------------------------------------------ */
  /* A live search.                                                      */
  /* ------------------------------------------------------------------ */
  let results: Page<VideoSearchItem> | null = null;
  let failure: { title: string; description: string } | null = null;

  try {
    results = await search(query);
  } catch (error) {
    if (isApiError(error) && error.isRateLimited) {
      failure = {
        title: "Slow down a moment",
        description: "You have been searching very quickly. Wait a few seconds and try again.",
      };
    } else if (isApiError(error) && error.isValidation) {
      failure = { title: "That search did not work", description: error.message };
    } else {
      failure = {
        title: "Search is having trouble",
        description: "Something went wrong on our side. Try the search again in a moment.",
      };
    }
  }

  const categories = await categoriesPromise;

  return (
    <div className="mx-auto w-full max-w-5xl flex-1 px-4 py-6 sm:px-6">
      {/*
       * Below `sm` the header hides its search box (a full autocomplete does not
       * fit a phone header, so it degrades to an icon), which would otherwise
       * leave the results page with no way to refine the query that produced
       * them. This field covers exactly that gap and disappears at `sm`, where
       * the header's takes over — so there is one search box on screen, never
       * two, at every viewport.
       *
       * key={q}: back/forward changes ?q= without remounting the tree, and a
       * reset key is how React resyncs state to a prop — not an effect.
       */}
      <div className="mx-auto max-w-2xl sm:hidden">
        <SearchInput key={q} initialQuery={q} />
      </div>

      <SearchFilters categories={categories} className="mt-5 max-sm:mt-4" />

      {failure ? (
        <ErrorState title={failure.title} description={failure.description} className="mt-6" />
      ) : results && results.items.length === 0 ? (
        <EmptyState
          icon={SearchX}
          title={`No results for “${q}”`}
          description={
            hasFilters
              ? "Nothing matches with these filters on. Loosen them, check the spelling, or try fewer keywords."
              : "Check the spelling, try fewer or more general keywords, or search for a tag instead."
          }
          action={
            hasFilters ? (
              <Button asChild variant="outline">
                <Link href={`${routes.search}?q=${encodeURIComponent(q)}`}>Clear all filters</Link>
              </Button>
            ) : undefined
          }
          className="mt-6"
        />
      ) : results ? (
        <>
          <p aria-live="polite" className="mt-5 text-xs text-muted-foreground">
            <span className="tabular-nums">{formatCompact(results.pagination.total)}</span>{" "}
            {results.pagination.total === 1 ? "result" : "results"} for{" "}
            <span className="font-medium text-foreground">&ldquo;{q}&rdquo;</span>
          </p>
          <SearchResults items={results.items} className="mt-2" />
          <Pagination pagination={results.pagination} className="mt-8" />
        </>
      ) : null}
    </div>
  );
}
