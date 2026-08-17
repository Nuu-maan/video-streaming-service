import { cn } from "@/lib/utils";
import type { VideoStatus } from "@/types/common";

/**
 * The ladder the transcoder builds, in the order it builds it. Rungs a source
 * file is too small for are simply never produced, which is why a finished
 * video can legitimately show three of four filled.
 */
const LADDER = ["360p", "480p", "720p", "1080p"] as const;

interface QualityLadderProps {
  status: VideoStatus;
  /** The renditions the API reports as done. */
  available: string[];
  /** 0–100, meaningful only while processing. */
  progress: number;
  className?: string;
}

/**
 * Four segments, one per rendition, filling as the worker finishes each.
 *
 * Most video tools show an indeterminate spinner here and hide the only genuinely
 * interesting thing the backend does. This API reports `available_qualities` the
 * moment each rendition lands, so a creator can watch 360p go solid while 1080p
 * is still queued — which answers the question a spinner never does, namely
 * "is this actually moving, and how much is left".
 *
 * A Server Component: it renders whatever the last poll reported, and the
 * ProcessingPoller beside it is what makes that number current.
 */
export function QualityLadder({ status, available, progress, className }: QualityLadderProps) {
  const done = new Set(available.map((quality) => quality.toLowerCase()));
  const failed = status === "failed";

  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      <div className="flex items-center gap-1" aria-hidden>
        {LADDER.map((rung) => {
          const complete = done.has(rung);
          return (
            <span
              key={rung}
              title={rung}
              className={cn(
                "h-1.5 flex-1 rounded-full transition-colors duration-(--motion-medium) ease-out-quart",
                failed
                  ? "bg-destructive/30"
                  : complete
                    ? "bg-brand-500"
                    : status === "processing"
                      ? "bg-muted-foreground/25"
                      : "bg-muted",
              )}
            />
          );
        })}
      </div>

      {/*
       * The text, which is what a screen reader gets — the bars above are
       * decorative and marked so. Not a live region: the poller beside this
       * already announces progress, and two announcements per tick is one too
       * many.
       */}
      <p className="text-[0.6875rem] leading-none text-muted-foreground tabular-nums">
        {failed ? (
          "Transcode failed"
        ) : done.size > 0 ? (
          <>
            {[...LADDER].filter((rung) => done.has(rung)).join(" · ")}
            {status === "processing" ? <span className="ml-1 opacity-70">+{progress}%</span> : null}
          </>
        ) : status === "processing" ? (
          `Encoding — ${progress}%`
        ) : (
          "Queued"
        )}
      </p>
    </div>
  );
}
