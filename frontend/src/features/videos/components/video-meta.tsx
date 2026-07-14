import type { Video } from "@/types/common";
import { formatCount, formatDate, formatRelativeTime } from "@/lib/format";
import { cn } from "@/lib/utils";

interface VideoMetaProps {
  video: Video;
  /**
   * The uploader, when the caller has it — `Video` itself only carries a
   * user_id, so whoever composes the watch page resolves the channel and
   * passes it in.
   */
  channel?: { id: string; username: string };
  className?: string;
}

/**
 * Title-and-facts block for a single video: the h1, the channel, and the
 * "12K views · 3 days ago" line. Server-compatible.
 *
 * The channel name is plain text, not a link: `/users/{id}` is not a route this
 * app can have, because the API exposes no `GET /users/{id}` and no way to list
 * another creator's videos. It used to link there and it 404'd every time.
 */
export function VideoMeta({ video, channel, className }: VideoMetaProps) {
  return (
    <div className={cn("min-w-0", className)}>
      <h1 className="text-heading text-balance">{video.title}</h1>
      <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-muted-foreground">
        {channel ? (
          <>
            <span className="max-w-56 truncate font-medium text-foreground">
              {channel.username}
            </span>
            <span aria-hidden>·</span>
          </>
        ) : null}
        <span className="tabular-nums">{formatCount(video.view_count, "view")}</span>
        <span aria-hidden>·</span>
        <time dateTime={video.created_at} title={formatDate(video.created_at)} suppressHydrationWarning>
          {formatRelativeTime(video.created_at)}
        </time>
        {video.category ? (
          <>
            <span aria-hidden>·</span>
            <span className="truncate">{video.category}</span>
          </>
        ) : null}
      </div>
    </div>
  );
}
