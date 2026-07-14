import { Film } from "lucide-react";
import Link from "next/link";

import { VideoStatusBadge } from "@/features/videos/components/video-status-badge";
import type { VideoCardData } from "@/features/videos/types";
import { routes } from "@/config/routes";
import { formatCount, formatDate, formatDuration, formatRelativeTime } from "@/lib/format";
import { cn } from "@/lib/utils";

interface VideoCardProps {
  video: VideoCardData;
  className?: string;
}

/**
 * The card everything browses through. One link covers the whole card — the
 * uploader is plain text, not a nested link, so keyboard users get exactly one
 * tab stop per video and the focus ring wraps the full card.
 *
 * The 16:9 box is fixed before the image loads (no CLS), the duration sits in
 * tabular figures so a 9:59 → 10:00 tick never shifts the badge, and hover
 * feedback is transform-only: the thumbnail scales inside its clipped box.
 */
export function VideoCard({ video, className }: VideoCardProps) {
  const status = video.status ?? "ready";
  const ready = status === "ready";

  return (
    <article className={cn("group", className)}>
      <Link
        href={routes.video(video.id)}
        className={cn(
          "flex flex-col gap-2.5 rounded-xl outline-none",
          // Transform only — the lift and the press never touch layout, so the
          // grid around the card cannot reflow while it is being hovered.
          "transition-transform duration-(--motion-fast) ease-out-quart",
          // `hover-ok:` — on a touch device :hover fires on tap and STICKS, so an
          // ungated lift would leave the card hanging mid-air on the way to a
          // navigation the user already committed to. The press scale is fine
          // ungated: :active ends when the finger does.
          "hover-ok:group-hover:-translate-y-0.5 active:scale-[0.985] active:duration-75",
          "focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
        )}
      >
        {/* The 16:9 box is sized before a single byte of the image arrives, so
            the thumbnail loading in shifts nothing (no CLS). rounded-xl outer,
            and the overflow clip is what the thumbnail scales inside. */}
        <div className="relative aspect-video w-full overflow-hidden rounded-xl bg-muted">
          {video.thumbnailUrl ? (
            /* Plain <img>: thumbnails come off the API origin, which is not in
               the image optimizer's remotePatterns — and they are already
               encoder-sized. The aspect box above owns layout, so no CLS. */
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={video.thumbnailUrl}
              alt=""
              loading="lazy"
              decoding="async"
              className="size-full object-cover transition-transform duration-(--motion-medium) ease-out-quart hover-ok:group-hover:scale-[1.04]"
            />
          ) : (
            <div className="flex size-full items-center justify-center text-muted-foreground/60">
              <Film aria-hidden className="size-7" />
            </div>
          )}

          {/* No image hairline here on purpose: globals.css already puts a
              1px inset black/white outline on every <img>. A second one would
              double up along the same edge. */}
          {ready && video.duration > 0 ? (
            /* Tabular figures: 9:59 → 10:00 must not nudge the badge's width. */
            <span className="absolute right-1.5 bottom-1.5 rounded-md bg-black/75 px-1.5 py-0.5 text-xs font-medium text-white tabular-nums">
              {formatDuration(video.duration)}
            </span>
          ) : null}

          {!ready ? (
            <div className="absolute inset-0 flex items-center justify-center bg-background/55">
              <VideoStatusBadge status={status} progress={video.transcodingProgress} />
            </div>
          ) : null}
        </div>

        <div className="min-w-0 px-0.5">
          {/* brand-400 is a DARK-THEME text colour: on the light background it is
              2.56:1, which fails AA and even the 3:1 UI floor. The scoped pair is
              6.9:1 in light and 7.4:1 in dark. See the ramp note in globals.css. */}
          <h3 className="line-clamp-2 text-sm leading-snug font-medium text-pretty transition-colors duration-(--motion-fast) group-hover:text-brand-700 dark:group-hover:text-brand-400">
            {video.title}
          </h3>
          <div className="mt-1 text-xs text-muted-foreground">
            {video.channelName ? <p className="truncate">{video.channelName}</p> : null}
            <p className="flex items-center gap-1">
              <span className="tabular-nums">{formatCount(video.viewCount, "view")}</span>
              <span aria-hidden>·</span>
              {/* Relative time drifts between render and hydration when this
                  ends up inside a client boundary — the drift is cosmetic. */}
              <time dateTime={video.createdAt} title={formatDate(video.createdAt)} suppressHydrationWarning>
                {formatRelativeTime(video.createdAt)}
              </time>
            </p>
          </div>
        </div>
      </Link>
    </article>
  );
}
