import { WatchLaterButton } from "@/features/watch-later/components/watch-later-button";
import { VideoCard } from "@/features/videos/components/video-card";
import { videoGridClassName } from "@/features/videos/components/video-grid";
import type { VideoCardData } from "@/features/videos/types";
import { cn } from "@/lib/utils";

interface WatchLaterGridProps {
  videos: VideoCardData[];
  className?: string;
}

/**
 * The standard video grid, plus a remove affordance per card. The button sits
 * over the thumbnail and only materialises on hover or keyboard focus — it is
 * an edit control on a browsing surface, so it stays out of the way until the
 * pointer says otherwise, and it never hides from a keyboard.
 */
export function WatchLaterGrid({ videos, className }: WatchLaterGridProps) {
  return (
    <div className={cn(videoGridClassName, className)}>
      {videos.map((video) => (
        <div key={video.id} className="group/item relative">
          <VideoCard video={video} />
          <div className="absolute top-2 right-2 opacity-0 transition-opacity duration-(--motion-fast) group-hover/item:opacity-100 focus-within:opacity-100 max-md:opacity-100">
            <WatchLaterButton videoId={video.id} initialSaved isAuthenticated variant="icon" />
          </div>
        </div>
      ))}
    </div>
  );
}
