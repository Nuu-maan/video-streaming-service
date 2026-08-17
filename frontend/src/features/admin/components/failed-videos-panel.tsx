import { CircleCheck } from "lucide-react";

import { EmptyState } from "@/components/common/empty-state";
import { Separator } from "@/components/ui/separator";
import { Panel } from "@/features/admin/components/panel";
import { RetryVideoButton } from "@/features/admin/components/retry-video-button";
import { RetryVideoForm } from "@/features/admin/components/retry-video-form";
import { formatRelativeTime } from "@/lib/format";
import type { Video } from "@/types/common";

/**
 * Failed transcodes, each with a one-click retry.
 *
 * The list is best-effort: it comes from `GET /videos?status=failed`, which
 * only surfaces public videos, so a private video that failed is not in it. The
 * retry-by-ID form underneath is the escape hatch for exactly that case, and the
 * copy says so — an admin who trusts an incomplete list is worse off than one
 * who knows it is incomplete.
 */
export function FailedVideosPanel({ videos }: { videos: Video[] }) {
  return (
    <Panel
      title="Failed transcodes"
      description="Re-queue a video whose transcode failed. Only a video in status failed can be retried."
    >
      {videos.length === 0 ? (
        <EmptyState
          icon={CircleCheck}
          title="Nothing has failed"
          description="No public video is sitting in a failed state. A private one still could be — use the ID field below if you're chasing a specific video."
          className="min-h-40 border-0"
        />
      ) : (
        <ul className="flex flex-col">
          {videos.map((video) => (
            <li
              key={video.id}
              className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 border-b border-border py-3 first:pt-0 last:border-0 last:pb-0"
            >
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium" title={video.title}>
                  {video.title}
                </p>
                <p className="mt-0.5 truncate font-mono text-xs text-muted-foreground">
                  {video.id} ·{" "}
                  <time dateTime={video.created_at}>{formatRelativeTime(video.created_at)}</time>
                </p>
              </div>
              <RetryVideoButton videoId={video.id} title={video.title} />
            </li>
          ))}
        </ul>
      )}

      <Separator className="my-5" />

      <div className="space-y-2">
        <p className="text-sm text-pretty text-muted-foreground">
          Chasing a video that isn&rsquo;t listed? A failed <em>private</em> video never appears
          above — the listing endpoint only returns public ones. Retry it by ID.
        </p>
        <RetryVideoForm />
      </div>
    </Panel>
  );
}
