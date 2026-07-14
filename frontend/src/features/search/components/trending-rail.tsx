import Link from "next/link";

import { Skeleton } from "@/components/ui/skeleton";
import { routes } from "@/config/routes";
import { getTrending } from "@/features/search/api";
import { ResultThumbnail } from "@/features/search/components/result-thumbnail";
import type { TrendingWindow } from "@/features/search/types";
import { formatCount, formatRelativeTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { VideoSearchItem } from "@/types/common";

interface TrendingRailProps {
  window?: TrendingWindow;
  limit?: number;
  title?: string;
  className?: string;
}

/**
 * A horizontal scroller of trending videos — a real scroll surface with snap
 * points, not a carousel: trackpads, touch and Shift+wheel all just work, the
 * scroller itself is focusable so arrow keys pan it, and every card is a
 * plain link the Tab key reaches (the browser scrolls focused cards into
 * view for free).
 *
 * Async Server Component; wrap in Suspense with `<TrendingRailSkeleton />`.
 * A rail is furniture — if the fetch fails it renders nothing rather than
 * breaking its page.
 */
export async function TrendingRail({ window = "24h", limit = 12, title = "Trending", className }: TrendingRailProps) {
  let items: VideoSearchItem[];
  try {
    items = await getTrending(window, limit);
  } catch {
    return null;
  }
  if (items.length === 0) return null;

  return (
    <section aria-label={title} className={cn("min-w-0", className)}>
      <h2 className="text-heading">{title}</h2>
      <ul
        tabIndex={0}
        aria-label={`${title} videos, horizontally scrollable`}
        className={cn(
          "mt-3 flex snap-x snap-mandatory gap-3 overflow-x-auto rounded-xl pb-3",
          "outline-none focus-visible:ring-3 focus-visible:ring-ring/50",
          // Snap to the padding edge, not the viewport edge.
          "-mx-1 scroll-px-1 px-1 pt-1",
        )}
      >
        {items.map((item) => (
          <li key={item.video_id} className="w-56 shrink-0 snap-start sm:w-64">
            <Link
              href={routes.video(item.video_id)}
              className="group flex flex-col gap-2 rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
            >
              <ResultThumbnail src={item.thumbnail_url} duration={item.duration} className="w-full" />
              <div className="min-w-0 px-0.5">
                <h3 className="line-clamp-2 text-sm font-medium">{item.title}</h3>
                <p className="mt-0.5 truncate text-xs text-muted-foreground">{item.username}</p>
                <p className="text-xs text-muted-foreground">
                  <span className="tabular-nums">{formatCount(item.views, "view")}</span>
                  <span aria-hidden className="px-1">
                    ·
                  </span>
                  <time dateTime={item.created_at}>{formatRelativeTime(item.created_at)}</time>
                </p>
              </div>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}

/** Suspense fallback shaped like the rail it replaces. */
export function TrendingRailSkeleton({ cards = 5, className }: { cards?: number; className?: string }) {
  return (
    <div className={cn("min-w-0", className)}>
      <Skeleton className="h-6 w-28 rounded-md" />
      <div className="mt-3 flex gap-3 overflow-hidden pt-1 pb-3">
        {Array.from({ length: cards }, (_, index) => (
          <div key={index} className="flex w-56 shrink-0 flex-col gap-2 sm:w-64">
            <Skeleton className="aspect-video w-full rounded-lg" />
            <Skeleton className="h-4 w-full rounded-md" />
            <Skeleton className="h-3 w-1/2 rounded-md" />
          </div>
        ))}
      </div>
    </div>
  );
}
