"use client";

import { useCallback, useEffect, useRef, useState } from "react";

import type Hls from "hls.js";

import type { PlayerFailure, QualityLevel } from "@/features/player/types";

interface UseHlsOptions {
  videoRef: React.RefObject<HTMLVideoElement | null>;
  src: string;
}

interface UseHlsResult {
  /** Variants the stream offers, highest first. Empty on native HLS (Safari). */
  levels: QualityLevel[];
  /** Selected level index; -1 means automatic ABR. */
  currentLevel: number;
  /** Height actually playing right now — labels "Auto (720p)". */
  activeHeight: number | null;
  /** True on Safari, where the browser owns HLS and quality selection. */
  isNative: boolean;
  failure: PlayerFailure | null;
  setLevel: (index: number) => void;
  retry: () => void;
}

const FAILURES = {
  unavailable: {
    title: "This video is unavailable",
    description: "The stream could not be found. It may have been removed.",
  },
  network: {
    title: "Connection problem",
    description: "The video stopped loading. Check your connection and try again.",
  },
  playback: {
    title: "Playback error",
    description: "Something went wrong while decoding this video.",
  },
  unsupported: {
    title: "Browser not supported",
    description: "This browser cannot play HLS video. Try a current version of Chrome, Firefox, or Safari.",
  },
} as const satisfies Record<string, PlayerFailure>;

/**
 * Owns the hls.js lifecycle: dynamic import (it stays out of the initial
 * bundle), Safari's native path, quality levels, fatal-error mapping, and —
 * critically — destroying the instance on unmount so a navigation does not
 * leak a media pipeline.
 */
export function useHls({ videoRef, src }: UseHlsOptions): UseHlsResult {
  const hlsRef = useRef<Hls | null>(null);
  const [levels, setLevels] = useState<QualityLevel[]>([]);
  const [currentLevel, setCurrentLevel] = useState(-1);
  const [activeHeight, setActiveHeight] = useState<number | null>(null);
  const [isNative, setIsNative] = useState(false);
  const [failure, setFailure] = useState<PlayerFailure | null>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !src) return;

    setLevels([]);
    setCurrentLevel(-1);
    setActiveHeight(null);
    setFailure(null);

    // Safari plays HLS natively — hand it the manifest and skip hls.js
    // entirely. It also runs its own ABR, so there is no quality menu there.
    if (video.canPlayType("application/vnd.apple.mpegurl")) {
      setIsNative(true);
      const onError = () => setFailure(FAILURES.network);
      video.addEventListener("error", onError);
      video.src = src;

      return () => {
        video.removeEventListener("error", onError);
        video.removeAttribute("src");
        video.load();
      };
    }

    setIsNative(false);
    let disposed = false;
    let hls: Hls | null = null;
    // One free MEDIA_ERROR recovery before giving up — hls.js's own docs
    // recommend a single recoverMediaError() attempt for transient decode
    // stalls before treating the error as fatal.
    let mediaRecoveryUsed = false;

    void (async () => {
      const { default: HlsLib } = await import("hls.js");
      if (disposed) return;

      if (!HlsLib.isSupported()) {
        setFailure(FAILURES.unsupported);
        return;
      }

      hls = new HlsLib();
      hlsRef.current = hls;

      hls.on(HlsLib.Events.MANIFEST_PARSED, () => {
        if (!hls) return;
        const parsed = hls.levels
          .map((level, index) => ({ index, height: level.height, bitrate: level.bitrate }))
          .sort((a, b) => b.height - a.height);
        setLevels(parsed);
      });

      hls.on(HlsLib.Events.LEVEL_SWITCHED, (_event, data) => {
        if (!hls) return;
        setActiveHeight(hls.levels[data.level]?.height ?? null);
      });

      hls.on(HlsLib.Events.ERROR, (_event, data) => {
        if (!data.fatal || !hls) return;

        if (data.type === HlsLib.ErrorTypes.MEDIA_ERROR && !mediaRecoveryUsed) {
          mediaRecoveryUsed = true;
          hls.recoverMediaError();
          return;
        }

        if (data.type === HlsLib.ErrorTypes.NETWORK_ERROR) {
          setFailure(data.response?.code === 404 ? FAILURES.unavailable : FAILURES.network);
        } else {
          setFailure(FAILURES.playback);
        }

        hls.destroy();
        hls = null;
        hlsRef.current = null;
      });

      hls.loadSource(src);
      hls.attachMedia(video);
    })();

    return () => {
      disposed = true;
      hls?.destroy();
      hls = null;
      hlsRef.current = null;
    };
  }, [videoRef, src, attempt]);

  const setLevel = useCallback((index: number) => {
    setCurrentLevel(index);
    const hls = hlsRef.current;
    if (hls) hls.currentLevel = index;
  }, []);

  const retry = useCallback(() => setAttempt((n) => n + 1), []);

  return { levels, currentLevel, activeHeight, isNative, failure, setLevel, retry };
}
