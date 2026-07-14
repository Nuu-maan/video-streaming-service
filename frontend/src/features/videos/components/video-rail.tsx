import { VideoCard } from "@/features/videos/components/video-card";
import { VideoCardSkeleton } from "@/features/videos/components/video-card-skeleton";
import type { VideoCardData } from "@/features/videos/types";
import { cn } from "@/lib/utils";

interface VideoRailProps {
  videos: VideoCardData[];
  className?: string;
}

/**
 * A horizontal shelf of VideoCards — the same card the grid uses, so trending
 * and "latest" never look like two different products.
 *
 * A real scroll surface, not a carousel: trackpads, touch and shift+wheel all
 * work for free, snap points stop it landing mid-card, and every card stays a
 * plain link the Tab key reaches (the browser scrolls a focused card into view
 * on its own). Nothing here is a custom scroll implementation, which is exactly
 * why it behaves correctly on every input device.
 */
export function VideoRail({ videos, className }: VideoRailProps) {
  if (videos.length === 0) return null;

  return (
    <ul
      className={cn(
        "no-scrollbar flex snap-x snap-mandatory gap-4 overflow-x-auto",
        // The rail bleeds into the page gutter so cards scroll out under the
        // margin rather than stopping short of it; scroll-padding keeps the
        // snap target aligned to the content edge, not the viewport edge.
        "-mx-4 scroll-px-4 px-4 sm:-mx-6 sm:scroll-px-6 sm:px-6",
        // Room for the card's hover lift and focus ring to render un-clipped.
        "pt-1 pb-3",
        className,
      )}
    >
      {videos.map((video) => (
        <li key={video.id} className="w-64 shrink-0 snap-start sm:w-72">
          <VideoCard video={video} />
        </li>
      ))}
    </ul>
  );
}

/** Loading placeholder in the rail's exact shape. */
export function VideoRailSkeleton({ count = 6, className }: { count?: number; className?: string }) {
  return (
    <div
      className={cn(
        "no-scrollbar flex gap-4 overflow-hidden",
        "-mx-4 px-4 pt-1 pb-3 sm:-mx-6 sm:px-6",
        className,
      )}
    >
      {Array.from({ length: count }, (_, index) => (
        <VideoCardSkeleton key={index} className="w-64 shrink-0 sm:w-72" />
      ))}
    </div>
  );
}
