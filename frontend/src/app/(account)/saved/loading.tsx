import { Skeleton } from "@/components/ui/skeleton";
import { VideoGridSkeleton } from "@/features/videos/components/video-card-skeleton";

/** The shape of the page that is coming, not a spinner in the middle of nowhere. */
export default function WatchLaterLoading() {
  return (
    <>
      <div className="flex flex-col gap-2">
        <Skeleton className="h-8 w-48 rounded-md" />
        <Skeleton className="h-4 w-80 max-w-full rounded-md" />
      </div>
      <VideoGridSkeleton count={8} />
    </>
  );
}
