import { Skeleton } from "@/components/ui/skeleton";
import { StudioVideoTableSkeleton } from "@/features/studio/components/studio-video-table-skeleton";

/**
 * Streamed instantly while the library is fetched. It mirrors the real
 * layout — header block, then table rows — so the page settles rather than
 * jumps when the data arrives.
 */
export default function StudioLoading() {
  return (
    <>
      <div className="flex flex-wrap items-end justify-between gap-x-6 gap-y-3">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-8 w-44 rounded-lg" />
          <Skeleton className="h-4 w-72 rounded-md" />
        </div>
        <Skeleton className="h-8 w-28 rounded-lg" />
      </div>
      <StudioVideoTableSkeleton />
    </>
  );
}
