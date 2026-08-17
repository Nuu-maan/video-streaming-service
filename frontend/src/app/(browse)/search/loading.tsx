import { SearchResultsSkeleton } from "@/features/search/components/search-results";
import { Skeleton } from "@/components/ui/skeleton";

/**
 * Shown while `/search` streams in. It mirrors the real page's chrome — search
 * field, filter row, result count, then result-shaped rows — so navigating to a
 * new query redraws in place instead of collapsing to a spinner and back.
 *
 * The filter pills are sized like the controls they stand in for; skeletons
 * that lie about the shape of what is coming are worse than none.
 */
export default function SearchLoading() {
  return (
    <div className="mx-auto w-full max-w-5xl flex-1 px-4 py-6 sm:px-6">
      <div className="mx-auto max-w-2xl">
        <Skeleton className="h-11 w-full rounded-full" />
      </div>

      <div className="mt-5 flex flex-wrap items-center gap-2">
        <Skeleton className="h-8 w-36 rounded-full" />
        <Skeleton className="h-8 w-32 rounded-full" />
        <Skeleton className="h-8 w-36 rounded-full" />
      </div>

      <Skeleton className="mt-5 h-3 w-44 rounded-md" />

      <SearchResultsSkeleton className="mt-2" />
    </div>
  );
}
