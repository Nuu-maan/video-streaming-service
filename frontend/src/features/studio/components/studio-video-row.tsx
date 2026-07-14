import { Film } from "lucide-react";
import Link from "next/link";

import { Badge } from "@/components/ui/badge";
import { TableCell, TableRow } from "@/components/ui/table";
import { routes } from "@/config/routes";
import { StudioVideoActions } from "@/features/studio/components/studio-video-actions";
import { VisibilityBadge } from "@/features/studio/components/visibility-badge";
import type { StudioVideoRow as Row } from "@/features/studio/types";
import { ProcessingPoller } from "@/features/videos/components/processing-poller";
import { formatCompact, formatDate, formatDuration } from "@/lib/format";

interface StudioVideoRowProps {
  video: Row;
}

/**
 * One video in the creator's table. A Server Component: only the actions menu
 * and the live status poller are client islands, so a hundred rows cost two
 * small bundles, not a hundred hydrated rows.
 */
export function StudioVideoRow({ video }: StudioVideoRowProps) {
  const watchable = video.status === "ready";

  return (
    <TableRow className="group">
      <TableCell className="py-3">
        <div className="flex items-center gap-3">
          {/* A fixed 16:9 box, always. The thumbnail arrives late (or never,
              for a video still transcoding) and a box that grows when it lands
              would shove the whole table down. */}
          <div className="relative w-28 shrink-0 overflow-hidden rounded-lg bg-muted ring-1 ring-border/60 ring-inset">
            <div className="aspect-video">
              {video.thumbnailUrl ? (
                /* Plain <img>: thumbnails are served from the API origin (or
                   proxied through /api/media for private videos), neither of
                   which is configured in next/image's remotePatterns. */
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={video.thumbnailUrl}
                  alt=""
                  loading="lazy"
                  decoding="async"
                  className="size-full object-cover"
                />
              ) : (
                <div className="flex size-full items-center justify-center text-muted-foreground">
                  <Film aria-hidden className="size-5" />
                </div>
              )}
            </div>
            {video.duration > 0 ? (
              <span className="absolute right-1 bottom-1 rounded bg-black/75 px-1 py-px text-[0.6875rem] leading-4 font-medium text-white tabular-nums">
                {formatDuration(video.duration)}
              </span>
            ) : null}
          </div>

          <div className="min-w-0">
            {watchable ? (
              <Link
                href={routes.video(video.id)}
                className="line-clamp-2 rounded-sm text-sm font-medium text-pretty outline-none group-hover:underline underline-offset-4 focus-visible:ring-3 focus-visible:ring-ring/50"
              >
                {video.title}
              </Link>
            ) : (
              // Nothing to watch yet — a link to a page that cannot play is a
              // dead end, so it stays plain text until the transcode lands.
              <p className="line-clamp-2 text-sm font-medium text-pretty">{video.title}</p>
            )}
          </div>
        </div>
      </TableCell>

      <TableCell>
        {watchable ? (
          <Badge variant="secondary">Ready</Badge>
        ) : (
          /* Live: polls while the worker is still transcoding, and refreshes
             the page once it finishes, so the row becomes watchable without
             anyone reaching for F5. */
          <ProcessingPoller
            videoId={video.id}
            status={video.status}
            progress={video.transcodingProgress}
            /* Named: several rows can be transcoding at once, and five anonymous
               "processing, 50 percent" announcements tell a listener nothing. */
            title={video.title}
          />
        )}
      </TableCell>

      <TableCell>
        <VisibilityBadge visibility={video.visibility} />
      </TableCell>

      <TableCell className="text-right text-sm tabular-nums">{formatCompact(video.viewCount)}</TableCell>
      <TableCell className="text-right text-sm tabular-nums">{formatCompact(video.likeCount)}</TableCell>
      <TableCell className="text-right text-sm tabular-nums">
        {formatCompact(video.commentCount)}
      </TableCell>

      <TableCell className="text-sm whitespace-nowrap text-muted-foreground tabular-nums">
        <time dateTime={video.createdAt}>{formatDate(video.createdAt)}</time>
      </TableCell>

      <TableCell className="text-right">
        <StudioVideoActions video={video} />
      </TableCell>
    </TableRow>
  );
}
