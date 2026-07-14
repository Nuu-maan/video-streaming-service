"use client";

import { useCallback, useEffect, useRef, useState } from "react";

interface UseControlsVisibilityOptions {
  /** The player frame. Pointer movement anywhere inside it wakes the controls. */
  containerRef: React.RefObject<HTMLElement | null>;
  /** Controls only hide during playback. A paused player keeps them up, always. */
  playing: boolean;
  /** Pin them open: a menu is showing, the scrubber is being dragged, focus is inside. */
  hold: boolean;
}

/** Long enough not to feel twitchy, short enough to get out of the way. */
const HIDE_AFTER_MS = 2_500;

interface UseControlsVisibilityResult {
  visible: boolean;
  /** Force the controls awake and restart the timer — call it after any command. */
  wake: () => void;
}

/**
 * Auto-hiding controls.
 *
 * The rules are the ones every good player has converged on, and each is worth
 * stating because breaking any one of them is instantly irritating:
 *   - hidden only while playing, never while paused;
 *   - any pointer movement brings them back immediately;
 *   - they never vanish out from under an open menu, a drag, or keyboard focus.
 *
 * The cursor hides with them — a mouse pointer parked over a fullscreen film is
 * the one piece of UI nobody has ever wanted.
 *
 * Two implementation details carry the whole thing. The timer lives in a ref, so
 * a pointer moving across the frame restarts it sixty times a second without
 * costing a single render. And "visible while paused" is *derived*, not stored:
 * pausing cannot forget to show the controls, because there is no state to
 * forget with.
 */
export function useControlsVisibility({
  containerRef,
  playing,
  hold,
}: UseControlsVisibilityOptions): UseControlsVisibilityResult {
  /** The auto-hide flag alone. Pause and hold are layered on top, at read time. */
  const [awake, setAwake] = useState(true);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Listeners read the live values through refs so the pointer subscription
  // never has to be torn down and rebuilt as playback state changes.
  const holdRef = useRef(hold);
  const playingRef = useRef(playing);
  useEffect(() => {
    holdRef.current = hold;
    playingRef.current = playing;
  }, [hold, playing]);

  /** Restart the countdown — or cancel it outright when hiding is not allowed. */
  const arm = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    if (!playingRef.current || holdRef.current) return;
    timerRef.current = setTimeout(() => setAwake(false), HIDE_AFTER_MS);
  }, []);

  const wake = useCallback(() => {
    // Already true — React bails out, so a moving pointer costs nothing.
    setAwake(true);
    arm();
  }, [arm]);

  // Playing, pausing, or opening a menu re-arms (or cancels) the countdown. No
  // state is touched synchronously here: the only setState is inside the timeout.
  useEffect(() => {
    arm();
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [playing, hold, arm]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const onPointerMove = (event: PointerEvent) => {
      // Touch "movement" is a drag, not a hover. A tap on the frame is handled
      // by the frame itself.
      if (event.pointerType === "touch") return;
      wake();
    };
    const onPointerLeave = () => {
      if (!playingRef.current || holdRef.current) return;
      if (timerRef.current) clearTimeout(timerRef.current);
      setAwake(false);
    };

    container.addEventListener("pointermove", onPointerMove);
    container.addEventListener("pointerleave", onPointerLeave);

    return () => {
      container.removeEventListener("pointermove", onPointerMove);
      container.removeEventListener("pointerleave", onPointerLeave);
    };
  }, [containerRef, wake]);

  return { visible: awake || !playing || hold, wake };
}
