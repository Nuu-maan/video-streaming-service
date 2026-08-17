"use client";

import { createContext, useCallback, useContext, useMemo, useRef, useState } from "react";

import type { PlayerHandle } from "@/features/player/components/video-player";

/**
 * The wire between the player and the notes rail.
 *
 * They are siblings, not parent and child, and they have to be: the page streams
 * the player immediately and the notes in behind their own Suspense boundary,
 * which only works if neither is nested in the other. So the thing they share
 * lives above both.
 *
 * It is deliberately small. Notes state stays in the rail that owns it; the only
 * things that cross are the two facts each side cannot compute alone — where the
 * playhead is, and where the notes are.
 */
interface ReviewValue {
  /** Callback ref for <VideoPlayer>. Stable, so it never re-attaches on render. */
  attachPlayer: (handle: PlayerHandle | null) => void;
  /**
   * Seek the player. A no-op when there is no player — a video that is still
   * transcoding has notes but nothing to seek, and that must not throw.
   */
  seekTo: (seconds: number) => void;
  /** The playhead now, or null when there is no player to ask. */
  captureTime: () => number | null;
  /** Anchored seconds, drawn as ticks on the scrub bar. */
  markers: number[];
  /** The rail publishes its anchors here whenever its list changes. */
  publishMarkers: (markers: number[]) => void;
}

const ReviewContext = createContext<ReviewValue | null>(null);

export function ReviewProvider({ children }: { children: React.ReactNode }) {
  const playerRef = useRef<PlayerHandle | null>(null);
  const [markers, setMarkers] = useState<number[]>([]);

  const attachPlayer = useCallback((handle: PlayerHandle | null) => {
    playerRef.current = handle;
  }, []);

  const seekTo = useCallback((seconds: number) => {
    playerRef.current?.seekTo(seconds);
  }, []);

  const captureTime = useCallback(() => playerRef.current?.currentTime() ?? null, []);

  /**
   * Compared before storing. The rail republishes on every list change —
   * including edits and deletes that touch no anchor at all — and setting an
   * equal-but-new array would re-render the player, and with it the whole media
   * chrome, for nothing.
   */
  const publishMarkers = useCallback((next: number[]) => {
    setMarkers((current) =>
      current.length === next.length && current.every((value, index) => value === next[index])
        ? current
        : next,
    );
  }, []);

  const value = useMemo(
    () => ({ attachPlayer, seekTo, captureTime, markers, publishMarkers }),
    [attachPlayer, seekTo, captureTime, markers, publishMarkers],
  );

  return <ReviewContext.Provider value={value}>{children}</ReviewContext.Provider>;
}

/**
 * Returns null outside a provider rather than throwing.
 *
 * The notes rail is also used on pages with no player — and a component that
 * explodes when it is reused somewhere reasonable is a component nobody reuses.
 * Callers branch on null: no player, no timestamp affordances.
 */
export function useReview(): ReviewValue | null {
  return useContext(ReviewContext);
}
