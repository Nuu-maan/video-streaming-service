import { Film } from "lucide-react";
import Link from "next/link";

import { routes } from "@/config/routes";
import { RemoveFromHistoryButton } from "@/features/history/components/remove-from-history-button";
import type { HistoryRow as HistoryRowData } from "@/features/history/types";
import { formatCount, formatDate, formatDuration, formatRelativeTime } from "@/lib/format";

/**
 * One watched video.
 *
 * A row rather than a card: history is a list you scan by title and date, and
 * the resume bar — the thing that actually distinguishes this surface from every
 * other grid of videos — reads far better under a wide thumbnail than a small one.
 *
 * Server Component. The only client island is the remove button.
 *
 * `video.thumbnailUrl` is already resolved by `toVideoCard`, which routes a
 * private or unlisted video's thumbnail through this origin's `/api/media` proxy
 * (the API 404s it without a bearer token, and the browser has none). Your own
 * private videos can legitimately appear in your own history, so this matters.
 */
export function HistoryRow({ row }: { row: HistoryRowData }) {
  const { video, progressPercent, completed, watchedAt } = row;
  const href = routes.video(video.id);

  return (
    <li className="group/row flex items-start gap-3 rounded-xl p-2 transition-colors duration-(--motion-fast) hover:bg-muted/40 sm:gap-4">
      <Link
        href={href}
        tabIndex={-1}
        aria-hidden
        className="relative aspect-video w-36 shrink-0 overflow-hidden rounded-lg bg-muted sm:w-44"
      >
        {video.thumbnailUrl ? (
          // eslint-disable-next-line @next/next/no-img-element -- API-origin media, and private thumbnails go through /api/media, which must not be re-cached by the image optimiser.
          <img
            src={video.thumbnailUrl}
            alt=""
            loading="lazy"
            decoding="async"
            className="absolute inset-0 size-full object-cover"
          />
        ) : (
          <div className="absolute inset-0 flex items-center justify-center text-muted-foreground/60">
            <Film aria-hidden className="size-6" />
          </div>
        )}

        {video.duration > 0 ? (
          <span className="absolute right-1.5 bottom-1.5 rounded-md bg-black/75 px-1.5 py-0.5 text-[11px] leading-4 font-medium text-white tabular-nums">
            {formatDuration(video.duration)}
          </span>
        ) : null}

        {/* The resume bar. Sits on the thumbnail's bottom edge, where a player's
            scrubber would be, so it reads as "you got this far" without a label.
            scaleX, not width: a transform skips layout entirely. */}
        {progressPercent > 0 ? (
          <span className="absolute inset-x-0 bottom-0 h-1 bg-white/25">
            <span
              className="block h-full origin-left bg-brand-500"
              style={{ transform: `scaleX(${progressPercent / 100})` }}
            />
          </span>
        ) : null}
      </Link>

      <div className="flex min-w-0 flex-1 flex-col gap-1 py-0.5">
        <h3 className="min-w-0">
          <Link
            href={href}
            className="line-clamp-2 rounded-sm text-sm font-medium text-pretty outline-none transition-colors duration-(--motion-fast) hover:text-brand-700 focus-visible:ring-3 focus-visible:ring-ring/50 dark:hover:text-brand-400"
          >
            {video.title}
          </Link>
        </h3>

        <p className="flex flex-wrap items-center gap-x-1.5 text-xs text-muted-foreground">
          {video.channelName ? (
            <>
              <span className="truncate">{video.channelName}</span>
              <span aria-hidden>·</span>
            </>
          ) : null}
          <span className="tabular-nums">{formatCount(video.viewCount, "view")}</span>
        </p>

        <p className="text-xs text-muted-foreground">
          <time dateTime={watchedAt} title={formatDate(watchedAt)}>
            Watched {formatRelativeTime(watchedAt)}
          </time>
          <span aria-hidden> · </span>
          {/* Persistent meta text, not a hover state — brand-400 alone was 2.5:1
              on the light theme. Scoped: 6.6:1 light, 6.9:1 dark. */}
          <span
            className={
              completed
                ? "text-muted-foreground"
                : "text-brand-700 tabular-nums dark:text-brand-400"
            }
          >
            {completed ? "Finished" : `${progressPercent}% watched`}
          </span>
        </p>
      </div>

      <RemoveFromHistoryButton videoId={video.id} videoTitle={video.title} />
    </li>
  );
}
