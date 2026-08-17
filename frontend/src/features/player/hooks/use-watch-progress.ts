"use client";

import { useEffect, useRef } from "react";

import { saveWatchProgress } from "@/features/player/actions";

interface UseWatchProgressOptions {
  videoId: string;
  videoRef: React.RefObject<HTMLVideoElement | null>;
  /**
   * Progress is a per-user record, so it needs a session. Anonymous viewers skip
   * this entirely — every beat would be a guaranteed 401.
   */
  enabled: boolean;
  /** Seconds to resume from, from `GET /me/history`. Null starts at the beginning. */
  resumeAt?: number | null;
  /** Fired once, after a resume actually happened, with the position seeked to. */
  onResume?: (seconds: number) => void;
}

/** Ten seconds: frequent enough to lose little, rare enough not to be chatty. */
const BEAT_MS = 10_000;

/** Watched this far and it counts as watched. Credits are not the point of a video. */
const COMPLETION_RATIO = 0.95;

/** A resume this close to the end is not a resume, it is a replay. */
const RESUME_TAIL_GUARD_SECONDS = 15;

/**
 * Persists where the viewer is, and puts them back there next time.
 *
 * Writes on a ten-second beat while playing, on pause, on seek-settle, on end,
 * on unmount — and, crucially, on the way out of the tab.
 *
 * That last one used to be a lie. The effect cleanup called `send()` and the
 * docblock claimed this covered "a viewer who closes the tab mid-video", but
 * React does not run effect cleanups when a tab is closed or the document is
 * discarded — only on client-side unmount. So the one case the comment named as
 * its reason for existing was exactly the case that lost the last nine seconds.
 *
 * There are now two exits, because there are two ways to leave:
 *
 *   • Client-side navigation → the cleanup runs → Server Action, as before.
 *   • Tab close / bfcache / backgrounding → `pagehide` and `visibilitychange`
 *     fire, and the save goes out through `navigator.sendBeacon` to
 *     `/api/progress`. A fetch (and therefore a Server Action) started during
 *     unload may be killed with the document; a beacon is handed to the browser
 *     and sent independently of it. It is the only thing the platform actually
 *     guarantees here.
 *
 * `visibilitychange` as well as `pagehide` because iOS Safari frequently
 * terminates a backgrounded tab without ever firing `pagehide`, and because
 * backgrounding is itself a good moment to checkpoint. Both routes share the
 * `lastSent` guard, so the pair firing together writes once, not twice.
 *
 * The position is clamped to the duration before it is sent. The API rejects
 * `position > duration` with a 400, and a media element will happily report a
 * currentTime a hair past its own duration at the moment it ends.
 */
export function useWatchProgress({
  videoId,
  videoRef,
  enabled,
  resumeAt,
  onResume,
}: UseWatchProgressOptions): void {
  // The callback is read through a ref so that a caller passing an inline arrow
  // does not re-subscribe the media listeners on every render. Synced in an
  // effect: a ref written during render is a value React may discard.
  const onResumeRef = useRef(onResume);
  useEffect(() => {
    onResumeRef.current = onResume;
  }, [onResume]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !enabled) return;

    let lastSent = -1;
    let resumed = false;

    /** Where we are, clamped, or null when the element cannot yet say. */
    const readPosition = (): { position: number; duration: number } | null => {
      const duration = video.duration;
      if (!Number.isFinite(duration) || duration <= 0) return null;
      return { position: Math.min(Math.max(video.currentTime, 0), duration), duration };
    };

    const send = (completed: boolean) => {
      const at = readPosition();
      if (!at) return;

      // Nothing moved since the last beat (paused, then paused again) — say nothing.
      if (!completed && Math.abs(at.position - lastSent) < 1) return;
      lastSent = at.position;

      void saveWatchProgress(videoId, { ...at, completed });
    };

    /**
     * The terminal save. A Server Action is a fetch, and a fetch begun while the
     * document is being torn down is not guaranteed to survive it — so this one
     * goes out as a beacon, which is.
     */
    const flush = () => {
      const at = readPosition();
      if (!at) return;
      if (Math.abs(at.position - lastSent) < 1) return;
      lastSent = at.position;

      if (typeof navigator.sendBeacon !== "function") {
        void saveWatchProgress(videoId, { ...at, completed: false });
        return;
      }

      const body = JSON.stringify({ videoId, ...at, completed: false });
      navigator.sendBeacon("/api/progress", new Blob([body], { type: "application/json" }));
    };

    const onPageHide = () => flush();
    const onVisibilityChange = () => {
      if (document.visibilityState === "hidden") flush();
    };

    const tryResume = () => {
      if (resumed) return;
      resumed = true;

      const duration = video.duration;
      if (!resumeAt || !Number.isFinite(duration) || duration <= 0) return;
      if (resumeAt >= duration - RESUME_TAIL_GUARD_SECONDS) return;

      video.currentTime = resumeAt;
      lastSent = resumeAt;
      onResumeRef.current?.(resumeAt);
    };

    const onLoadedMetadata = () => tryResume();
    const onPause = () => send(false);
    const onSeeked = () => send(false);
    const onEnded = () => send(true);

    // Metadata may already be in hand — hls.js can parse the manifest before
    // React gets around to running this effect.
    if (video.readyState >= HTMLMediaElement.HAVE_METADATA) tryResume();

    video.addEventListener("loadedmetadata", onLoadedMetadata);
    video.addEventListener("pause", onPause);
    video.addEventListener("seeked", onSeeked);
    video.addEventListener("ended", onEnded);

    // Leaving the document entirely. Neither of these is a React lifecycle, and
    // that is the whole point — React will not be running by the time they matter.
    window.addEventListener("pagehide", onPageHide);
    document.addEventListener("visibilitychange", onVisibilityChange);

    const beat = setInterval(() => {
      if (video.paused || video.seeking) return;
      const duration = video.duration;
      const completed = duration > 0 && video.currentTime >= duration * COMPLETION_RATIO;
      send(completed);
    }, BEAT_MS);

    return () => {
      clearInterval(beat);
      video.removeEventListener("loadedmetadata", onLoadedMetadata);
      video.removeEventListener("pause", onPause);
      video.removeEventListener("seeked", onSeeked);
      video.removeEventListener("ended", onEnded);
      window.removeEventListener("pagehide", onPageHide);
      document.removeEventListener("visibilitychange", onVisibilityChange);
      // Client-side navigation — the one exit where a cleanup genuinely runs, and
      // the most common way a watch ends. A Server Action is fine here.
      send(false);
    };
  }, [videoId, videoRef, enabled, resumeAt]);
}
