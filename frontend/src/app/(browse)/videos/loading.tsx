import { Skeleton } from "@/components/ui/skeleton";
import { VideoGridSkeleton } from "@/features/videos/components/video-card-skeleton";

/**
 * Not a spinner. A spinner says "something is happening"; this says "a page
 * header and a grid of twenty-four videos are about to be here", in the exact
 * positions they will occupy — so the swap to real content moves nothing.
 */
export default function VideosLoading() {
  return (
    <div className="mx-auto flex w-full max-w-[1600px] flex-1 flex-col gap-6 px-4 py-6 sm:px-6">
      <div className="flex flex-col gap-2">
        <Skeleton className="h-9 w-40 rounded-lg" />
        <Skeleton className="h-4 w-72 rounded-md" />
      </div>
      <VideoGridSkeleton count={12} />
    </div>
  );
}
