"use client";

import { useEffect, useRef } from "react";

import { recordView } from "@/features/player/actions";

interface UseViewTrackingOptions {
  videoId: string;
  videoRef: React.RefObject<HTMLVideoElement | null>;
  /** Latest rendition label ("720p", "auto") without re-subscribing listeners. */
  qualityRef: React.RefObject<string | undefined>;
}

/** Seconds of genuine playback before a view is worth counting. */
const VIEW_THRESHOLD_SECONDS = 3;

const SESSION_KEY = "reel.view-session";

/**
 * Stable per-tab session id for anonymous viewers — the API dedupes repeat
 * views on it. sessionStorage on purpose: it survives navigation within the
 * tab (so bouncing between videos does not mint fresh identities) and dies
 * with the tab (so it is not a tracking cookie).
 */
function getViewSessionId(): string | undefined {
  try {
    let id = sessionStorage.getItem(SESSION_KEY);
    if (!id) {
      id = crypto.randomUUID();
      sessionStorage.setItem(SESSION_KEY, id);
    }
    return id;
  } catch {
    // Storage can be denied (private mode policies); the view is then simply
    // dedup'd less well, which is not worth surfacing.
    return undefined;
  }
}

/**
 * Fires `POST /videos/:id/view` once per mount, and only after ~3 seconds of
 * actual playback — counting on page load would inflate the number with every
 * bounced visit. Playback time is accumulated from `timeupdate` deltas, so
 * seeking, pausing, and buffering do not count toward the threshold.
 */
export function useViewTracking({ videoId, videoRef, qualityRef }: UseViewTrackingOptions): void {
  const firedRef = useRef(false);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    firedRef.current = false;
    let watched = 0;
    let lastTime: number | null = null;

    const onTimeUpdate = () => {
      if (firedRef.current || video.paused || video.seeking) return;

      const now = video.currentTime;
      if (lastTime !== null) {
        const delta = now - lastTime;
        // timeupdate fires ~4×/s; anything larger is a seek, not watching.
        if (delta > 0 && delta < 1) watched += delta;
      }
      lastTime = now;

      if (watched >= VIEW_THRESHOLD_SECONDS) {
        firedRef.current = true;
        void recordView(videoId, {
          quality: qualityRef.current,
          sessionId: getViewSessionId(),
        });
      }
    };

    const resetBaseline = () => {
      lastTime = null;
    };

    video.addEventListener("timeupdate", onTimeUpdate);
    video.addEventListener("seeking", resetBaseline);
    video.addEventListener("pause", resetBaseline);

    return () => {
      video.removeEventListener("timeupdate", onTimeUpdate);
      video.removeEventListener("seeking", resetBaseline);
      video.removeEventListener("pause", resetBaseline);
    };
  }, [videoId, videoRef, qualityRef]);
}
