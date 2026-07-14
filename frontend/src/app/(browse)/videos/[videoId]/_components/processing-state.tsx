import { CircleAlert, Film } from "lucide-react";

import { ProcessingPoller } from "@/features/videos/components/processing-poller";
import type { Video } from "@/types/common";

interface ProcessingStateProps {
  video: Video;
  /** Only the owner is told what to do about a failure — nobody else can act on it. */
  isOwner: boolean;
}

/**
 * What stands in for the player while the video is not watchable yet.
 *
 * The alternative — rendering the player anyway — gives the viewer a black
 * rectangle with a play button that does nothing, and no explanation. This box
 * occupies the identical 16:9 footprint, so when transcoding finishes and
 * `ProcessingPoller` refreshes the route, the player takes its place without the
 * page moving a pixel.
 */
export function ProcessingState({ video, isOwner }: ProcessingStateProps) {
  const failed = video.status === "failed";

  return (
    <div className="flex aspect-video w-full flex-col items-center justify-center gap-4 rounded-xl bg-muted/60 px-6 text-center ring-1 ring-border ring-inset">
      <div
        className={
          failed
            ? "flex size-14 items-center justify-center rounded-2xl bg-destructive/10 text-destructive ring-1 ring-destructive/20 ring-inset"
            : "flex size-14 items-center justify-center rounded-2xl bg-background text-muted-foreground ring-1 ring-border ring-inset"
        }
      >
        {failed ? <CircleAlert aria-hidden className="size-6" /> : <Film aria-hidden className="size-6" />}
      </div>

      <div className="space-y-1.5">
        <h2 className="text-heading text-balance">
          {failed ? "This video couldn't be processed" : "This video is still being prepared"}
        </h2>
        <p className="mx-auto max-w-sm text-sm text-pretty text-muted-foreground">
          {failed
            ? isOwner
              ? "Transcoding failed. Try uploading the file again — if it keeps failing, the source file may be damaged."
              : "Something went wrong while transcoding it. It may never become watchable."
            : "We're transcoding it into every quality. This page updates itself the moment it's ready."}
        </p>
      </div>

      {!failed ? (
        <ProcessingPoller videoId={video.id} status={video.status} progress={video.transcoding_progress} />
      ) : null}
    </div>
  );
}
