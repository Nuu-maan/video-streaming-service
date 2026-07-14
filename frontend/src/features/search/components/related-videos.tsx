import Link from "next/link";

import { Skeleton } from "@/components/ui/skeleton";
import { routes } from "@/config/routes";
import { getRelated } from "@/features/search/api";
import { ResultThumbnail } from "@/features/search/components/result-thumbnail";
import { formatCount, formatRelativeTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { VideoSearchItem } from "@/types/common";

interface RelatedVideosProps {
  /** The video being watched; excluded from its own rail. */
  videoId: string;
  className?: string;
}

/**
 * The "up next" rail for the watch page: compact rows, fetched from
 * `GET /videos/:id/related`. It is an async Server Component — render it
 * inside a `<Suspense fallback={<RelatedVideosSkeleton />}>` so the player
 * never waits on it. A rail must never take the watch page down, so a failed
 * fetch renders nothing.
 */
export async function RelatedVideos({ videoId, className }: RelatedVideosProps) {
  let items: VideoSearchItem[];
  try {
    items = await getRelated(videoId, 12);
  } catch {
    return null;
  }

  // "Topped up from trending" can hand the current video back — drop it.
  const related = items.filter((item) => item.video_id !== videoId);
  if (related.length === 0) return null;

  return (
    <aside aria-label="Related videos" className={cn("min-w-0", className)}>
      <h2 className="text-heading">Related</h2>
      <ul className="mt-3">
        {related.map((item) => (
          <li key={item.video_id}>
            <Link
              href={routes.video(item.video_id)}
              className="group -mx-2 flex gap-2.5 rounded-lg p-2 outline-none transition-colors duration-(--motion-fast) hover:bg-muted/50 focus-visible:ring-3 focus-visible:ring-ring/50"
            >
              <ResultThumbnail src={item.thumbnail_url} duration={item.duration} className="w-36" />
              <div className="min-w-0 flex-1 py-0.5">
                <h3 className="line-clamp-2 text-sm font-medium">{item.title}</h3>
                <p className="mt-1 truncate text-xs text-muted-foreground">{item.username}</p>
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
    </aside>
  );
}

/** Suspense fallback shaped like the rail it replaces. */
export function RelatedVideosSkeleton({ rows = 6, className }: { rows?: number; className?: string }) {
  return (
    <div className={cn("min-w-0", className)}>
      <Skeleton className="h-6 w-24 rounded-md" />
      <div className="mt-3 flex flex-col gap-2">
        {Array.from({ length: rows }, (_, index) => (
          <div key={index} className="flex gap-2.5 p-2 pl-0">
            <Skeleton className="aspect-video w-36 shrink-0 rounded-lg" />
            <div className="flex min-w-0 flex-1 flex-col gap-2 py-0.5">
              <Skeleton className="h-4 w-full rounded-md" />
              <Skeleton className="h-3 w-2/3 rounded-md" />
              <Skeleton className="h-3 w-1/2 rounded-md" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
