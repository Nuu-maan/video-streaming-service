import { videoGridClassName } from "@/features/videos/components/video-grid";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

/**
 * Loading placeholder in the exact shape of VideoCard: 16:9 media, two title
 * lines, one meta line — so the swap to real content moves nothing.
 */
export function VideoCardSkeleton({ className }: { className?: string }) {
  return (
    <div className={cn("flex flex-col gap-2.5", className)}>
      <Skeleton className="aspect-video w-full rounded-xl" />
      <div className="flex flex-col gap-1.5 px-0.5">
        <Skeleton className="h-4 w-full rounded-md" />
        <Skeleton className="h-4 w-3/5 rounded-md" />
        <Skeleton className="mt-0.5 h-3 w-2/5 rounded-md" />
      </div>
    </div>
  );
}

export function VideoGridSkeleton({ count = 12, className }: { count?: number; className?: string }) {
  return (
    <div className={cn(videoGridClassName, className)}>
      {Array.from({ length: count }, (_, index) => (
        <VideoCardSkeleton key={index} />
      ))}
    </div>
  );
}
