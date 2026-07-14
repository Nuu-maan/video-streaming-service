import { Skeleton } from "@/components/ui/skeleton";
import { VideoGridSkeleton } from "@/features/videos/components/video-card-skeleton";

/**
 * Shaped like the page it precedes: a title, a subtitle, the three-tab window
 * switcher on the right, and a grid. Nothing moves when the real content lands.
 */
export default function TrendingLoading() {
  return (
    <div className="mx-auto flex w-full max-w-7xl flex-1 flex-col gap-6 px-4 py-8 sm:px-6">
      <div className="flex flex-wrap items-end justify-between gap-x-6 gap-y-3">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-9 w-40 rounded-lg" />
          <Skeleton className="h-4 w-64 rounded-md" />
        </div>
        <Skeleton className="h-10 w-64 rounded-full" />
      </div>
      <VideoGridSkeleton count={12} />
    </div>
  );
}
