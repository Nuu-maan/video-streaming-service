"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

import { pollVideoStatus } from "@/features/videos/actions";
import { VideoStatusBadge } from "@/features/videos/components/video-status-badge";
import type { VideoStatus } from "@/types/common";
import { cn } from "@/lib/utils";

const POLL_INTERVAL_MS = 3_000;
/** Transcoding an average upload takes a minute or two; past five, stop asking. */
const GIVE_UP_AFTER_MS = 5 * 60 * 1_000;

interface ProcessingPollerProps {
  videoId: string;
  /** The status the server rendered with; polling starts only if it is non-terminal. */
  status: VideoStatus;
  /** 0–100 as of the server render. */
  progress?: number;
  /**
   * Names the video in the spoken announcement. Worth passing wherever several
   * of these can be on screen at once — in the studio table, five rows all
   * saying "Processing, 50 percent" are five rows saying nothing.
   */
  title?: string;
  className?: string;
}

/**
 * Live status for a video that is still uploading/processing. Polls the
 * status endpoint (via a Server Action — the token is httpOnly) every three
 * seconds, re-renders the badge with fresh progress, and `router.refresh()`es
 * once the video reaches a terminal state so the server tree catches up.
 * Stops on unmount, on terminal states, and after five minutes — a poller
 * that never gives up is a tab quietly hammering the API forever.
 *
 * WHAT IS SPOKEN IS NOT WHAT IS SHOWN. The badge ticks every three seconds and
 * is NOT in a live region: wrapping a ticking percentage in `aria-live` meant
 * the screen reader was interrupted with a new number every three seconds — and
 * because the studio renders one poller per unfinished row, a creator with five
 * uploads in flight had five live regions doing it at once. That is not a
 * progress indicator, it is a denial of service.
 *
 * The live region below announces MILESTONES only — the status transitions, and
 * every 25% — so the string it holds changes four or five times across an entire
 * transcode instead of a hundred. A live region only speaks when its content
 * actually changes, so a re-render carrying the same milestone is silent.
 */
export function ProcessingPoller({ videoId, status, progress, title, className }: ProcessingPollerProps) {
  const router = useRouter();
  const [live, setLive] = useState({ status, progress });
  const [timedOut, setTimedOut] = useState(false);
  const inFlight = useRef(false);

  const active = live.status === "uploading" || live.status === "processing";

  useEffect(() => {
    if (!active || timedOut) return;

    const startedAt = Date.now();
    const interval = setInterval(async () => {
      if (Date.now() - startedAt > GIVE_UP_AFTER_MS) {
        clearInterval(interval);
        setTimedOut(true);
        return;
      }
      if (inFlight.current) return; // a slow response must not stack requests
      inFlight.current = true;
      try {
        const report = await pollVideoStatus(videoId);
        if (!report?.status) return; // transient failure — try again next tick
        setLive({ status: report.status, progress: report.progress });
        if (report.status === "ready" || report.status === "failed") {
          clearInterval(interval);
          router.refresh();
        }
      } finally {
        inFlight.current = false;
      }
    }, POLL_INTERVAL_MS);

    return () => clearInterval(interval);
  }, [active, timedOut, videoId, router]);

  if (timedOut) {
    return (
      <p role="status" className={cn("text-sm text-muted-foreground", className)}>
        Still processing — this one is taking longer than usual. Check back in a bit.
      </p>
    );
  }

  return (
    <>
      {/* aria-live is deliberately absent here: this is the ticking one. */}
      <div className={className}>
        <VideoStatusBadge status={live.status} progress={live.progress} />
      </div>

      <span role="status" aria-live="polite" className="sr-only">
        {announce(live.status, live.progress, title)}
      </span>
    </>
  );
}

/**
 * The spoken string. Quantised to 25% steps, so it changes — and therefore is
 * spoken — at most a handful of times over the life of a transcode.
 */
function announce(status: VideoStatus, progress: number | undefined, title?: string): string {
  const subject = title ? `“${title}”` : "Your video";

  switch (status) {
    case "uploading":
      return `${subject}: uploading.`;
    case "processing": {
      if (typeof progress !== "number") return `${subject}: processing.`;
      const step = Math.floor(Math.min(100, Math.max(0, progress)) / 25) * 25;
      return step === 0 ? `${subject}: processing.` : `${subject}: processing, ${step} percent.`;
    }
    case "ready":
      return `${subject} is ready to watch.`;
    case "failed":
      return `${subject} failed to process.`;
    default:
      return "";
  }
}
