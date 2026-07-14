import { Film } from "lucide-react";
import Link from "next/link";

import { RemoveFromPlaylistButton } from "@/features/playlists/components/remove-from-playlist-button";
import type { PlaylistRow } from "@/features/playlists/types";
import { routes } from "@/config/routes";
import { formatCount, formatDuration } from "@/lib/format";
import { cn } from "@/lib/utils";

interface PlaylistVideoListProps {
  playlistId: string;
  rows: PlaylistRow[];
  /** Owner-only affordances. Everyone else gets a read-only list. */
  canEdit: boolean;
  /** Videos already shown on earlier pages, so the ordinal keeps counting. */
  offset?: number;
  className?: string;
}

/**
 * A playlist reads as an ordered list, so it is rendered as one: rows, not a
 * grid, with a running ordinal down the left.
 *
 * That ordinal is the reader's count — `offset + index + 1` — and deliberately
 * *not* `item.position`. Positions have gaps: removing a video leaves its slot
 * empty and the API does not renumber, so a list of three videos can hold
 * positions 1, 4 and 5. Showing those numbers would be showing the reader our
 * database.
 */
export function PlaylistVideoList({
  playlistId,
  rows,
  canEdit,
  offset = 0,
  className,
}: PlaylistVideoListProps) {
  return (
    <ol className={cn("flex flex-col", className)}>
      {rows.map((row, index) => (
        <li
          key={row.video.id}
          className="group/row flex items-center gap-3 rounded-xl p-2 transition-colors duration-(--motion-fast) hover:bg-muted/50 sm:gap-4"
        >
          <span
            aria-hidden
            className="w-6 shrink-0 text-right text-sm text-muted-foreground tabular-nums"
          >
            {offset + index + 1}
          </span>

          <Link
            href={routes.video(row.video.id)}
            className="flex min-w-0 flex-1 items-center gap-3 rounded-lg outline-none focus-visible:ring-3 focus-visible:ring-ring/50 sm:gap-4"
          >
            <div className="relative aspect-video w-32 shrink-0 overflow-hidden rounded-lg bg-muted sm:w-40">
              {row.video.thumbnailUrl ? (
                /* Plain <img>: these come off the API origin, outside the image
                   optimizer's remotePatterns, and the aspect box owns layout. */
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={row.video.thumbnailUrl}
                  alt=""
                  loading="lazy"
                  decoding="async"
                  className="size-full object-cover"
                />
              ) : (
                <div className="flex size-full items-center justify-center text-muted-foreground/60">
                  <Film aria-hidden className="size-5" />
                </div>
              )}

              {row.video.duration > 0 ? (
                <span className="absolute right-1 bottom-1 rounded bg-black/75 px-1 py-0.5 text-[0.6875rem] font-medium text-white tabular-nums">
                  {formatDuration(row.video.duration)}
                </span>
              ) : null}
            </div>

            <div className="min-w-0 flex-1">
              <h3 className="line-clamp-2 text-sm leading-snug font-medium text-pretty">
                {row.video.title}
              </h3>
              <p className="mt-1 text-xs text-muted-foreground">
                {row.video.channelName ? (
                  <>
                    <span className="truncate">{row.video.channelName}</span>
                    <span aria-hidden> · </span>
                  </>
                ) : null}
                <span className="tabular-nums">{formatCount(row.video.viewCount, "view")}</span>
              </p>
            </div>
          </Link>

          {canEdit ? (
            <RemoveFromPlaylistButton
              playlistId={playlistId}
              videoId={row.video.id}
              videoTitle={row.video.title}
            />
          ) : null}
        </li>
      ))}
    </ol>
  );
}
