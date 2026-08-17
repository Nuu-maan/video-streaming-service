import { Film, FileQuestion } from "lucide-react";
import Link from "next/link";

import { routes } from "@/config/routes";
import { toVideoCard } from "@/features/videos/card-data";
import { getVideo } from "@/features/videos/api";
import { isApiError } from "@/lib/api-error";
import { formatCount, formatDuration } from "@/lib/format";

/**
 * The reported video, inline in the queue.
 *
 * A moderator should not have to open a second tab to find out what they are
 * being asked to judge. The catch is that the thing being judged may already be
 * gone — deleted by its owner, or actioned by a colleague thirty seconds ago —
 * and the API answers that with a 404. That is not an error worth a red box: it
 * is information ("there is nothing left to delete here"), so it renders as its
 * own quiet state and the report stays reviewable, because a report on deleted
 * content still needs dismissing.
 *
 * Note the thumbnail goes through `toVideoCard`, which routes a private or
 * unlisted video's image through this origin's `/api/media` proxy. A reported
 * video is very often a private one, and the browser has no bearer token to
 * fetch it with — hitting the API origin directly would render a broken image.
 */
export async function ReportedVideo({ videoId }: { videoId: string }) {
  let video;
  try {
    video = await getVideo(videoId);
  } catch (error) {
    if (isApiError(error) && error.isNotFound) {
      return (
        <div className="flex items-center gap-3 rounded-lg bg-muted/40 p-3 text-sm text-muted-foreground">
          <FileQuestion aria-hidden className="size-4 shrink-0" />
          <span>This video is already gone — deleted, or actioned by someone else.</span>
        </div>
      );
    }
    throw error;
  }

  const card = toVideoCard(video);

  return (
    <div className="flex items-start gap-3 rounded-lg bg-muted/40 p-3">
      <div className="relative aspect-video w-28 shrink-0 overflow-hidden rounded-md bg-muted">
        {card.thumbnailUrl ? (
          // Plain <img>, as everywhere else: these come off the API origin (or
          // the media proxy), not the image optimizer's remotePatterns. The
          // aspect box owns layout, so there is no CLS when it lands.
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={card.thumbnailUrl}
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
        {video.duration > 0 ? (
          <span className="absolute right-1 bottom-1 rounded bg-black/80 px-1 text-[11px] font-medium tabular-nums text-white">
            {formatDuration(video.duration)}
          </span>
        ) : null}
      </div>

      <div className="min-w-0 flex-1">
        <Link
          href={routes.video(video.id)}
          className="line-clamp-2 text-sm font-medium outline-none hover:underline focus-visible:ring-3 focus-visible:ring-ring/50 rounded-sm"
        >
          {video.title}
        </Link>
        <p className="mt-1 text-xs tabular-nums text-muted-foreground">
          {formatCount(video.view_count, "view")} · {formatCount(video.like_count, "like")} ·{" "}
          {video.visibility}
        </p>
        {video.description ? (
          <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">{video.description}</p>
        ) : null}
      </div>
    </div>
  );
}
