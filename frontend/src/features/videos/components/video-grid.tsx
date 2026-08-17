import { VideoCard } from "@/features/videos/components/video-card";
import type { VideoCardData } from "@/features/videos/types";
import { cn } from "@/lib/utils";

/**
 * Shared by the real grid and its skeleton so the loading state occupies the
 * exact same tracks as the content that replaces it. auto-fill instead of
 * breakpoint counts: cards stay between 16rem and a track's share at every
 * viewport, without a jump at each breakpoint.
 */
export const videoGridClassName =
  "grid grid-cols-[repeat(auto-fill,minmax(16rem,1fr))] gap-x-4 gap-y-8";

interface VideoGridProps {
  videos: VideoCardData[];
  className?: string;
}

export function VideoGrid({ videos, className }: VideoGridProps) {
  return (
    <div className={cn(videoGridClassName, className)}>
      {videos.map((video) => (
        <VideoCard key={video.id} video={video} />
      ))}
    </div>
  );
}
