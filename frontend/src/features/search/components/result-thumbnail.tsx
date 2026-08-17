import { Film } from "lucide-react";

import { mediaUrl } from "@/lib/api-client";
import { formatDuration } from "@/lib/format";
import { cn } from "@/lib/utils";

interface ResultThumbnailProps {
  /** The `thumbnail_url` path the API hands back; resolved via `mediaUrl`. */
  src: string | null | undefined;
  /** Seconds; rendered as the corner duration badge. */
  duration: number;
  /** Sizing lives on the wrapper — pass a width, the 16:9 box does the rest. */
  className?: string;
}

/**
 * The 16:9 thumbnail every discovery surface shares: fixed aspect box (no
 * CLS), lazy image, a film-strip placeholder when there is no thumbnail yet,
 * and the duration stamped in the corner in tabular figures.
 *
 * Server-only by virtue of `mediaUrl` — render it from Server Components.
 */
export function ResultThumbnail({ src, duration, className }: ResultThumbnailProps) {
  const resolved = mediaUrl(src);

  return (
    <div className={cn("relative aspect-video shrink-0 overflow-hidden rounded-lg bg-muted", className)}>
      {resolved ? (
        // eslint-disable-next-line @next/next/no-img-element -- API-origin media; no remotePatterns configured for next/image.
        <img
          src={resolved}
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
      {duration > 0 ? (
        <span className="absolute right-1.5 bottom-1.5 rounded-md bg-black/75 px-1.5 py-0.5 text-[11px] leading-4 font-medium text-white tabular-nums">
          {formatDuration(duration)}
          <span className="sr-only"> long</span>
        </span>
      ) : null}
    </div>
  );
}
