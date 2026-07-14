"use client";

import { useEffect, useRef } from "react";

import type { PlayerActions } from "@/features/player/hooks/use-player-state";

interface UsePlayerKeyboardOptions {
  actions: PlayerActions;
  duration: number;
  /**
   * The player frame. The shortcuts are live only while focus is inside it —
   * see below. It must be focusable (`tabIndex={0}`) for that to ever be true.
   */
  containerRef: React.RefObject<HTMLElement | null>;
  /** Called on every handled key, so the controls wake up and show what changed. */
  onCommand: () => void;
  enabled?: boolean;
}

/**
 * Anything that owns the keystroke already. Typing "l" into the comment box must
 * not skip the video forward ten seconds, and Space on a focused button must
 * press the button once — not press it and toggle playback.
 */
const CLAIMS_KEYS =
  'input, textarea, select, button, a[href], [contenteditable=""], [contenteditable="true"], [role="menuitem"], [role="menuitemradio"], [role="slider"], [role="textbox"]';

/**
 * The keyboard contract every video player is expected to honour. A player
 * without it is not a player, it is a rectangle.
 *
 *   Space / K   play-pause          J / L   back / forward 10s
 *   ← / →       back / forward 5s   ↑ / ↓   volume ±5%
 *   M           mute                F       fullscreen
 *   0-9         seek to 0%-90%      Home/End  start / end
 *
 * ─────────────────────────────────────────────────────────────────────────────
 * THESE KEYS ARE SCOPED TO THE PLAYER, AND THAT IS NOT NEGOTIABLE.
 *
 * The listener still hangs off the window (it has to — the player's own element
 * cannot see a key pressed while focus is on the body), but it does nothing
 * unless focus is actually INSIDE the player frame.
 *
 * Bound to the window unconditionally, as this was, the handler called
 * preventDefault() on Space, the arrows, Home and End for anyone whose focus sat
 * on <body> — which is where focus sits by default on the watch page. A
 * keyboard-only visitor pressing Space or End to scroll down to the comments
 * would silently seek the video instead, and the page would not move. The
 * player did not even have to be on screen.
 *
 * It is also WCAG 2.1.4 (Character Key Shortcuts): single-character shortcuts
 * (k/j/l/m/f/0-9) must be remappable, switchable off, or ACTIVE ONLY ON FOCUS.
 * This is the third option, and the cheapest honest one.
 *
 * CLAIMS_KEYS stays as the second line of defence, for focus that is inside the
 * player but on a control that owns the key itself.
 * ─────────────────────────────────────────────────────────────────────────────
 */
export function usePlayerKeyboard({
  actions,
  duration,
  containerRef,
  onCommand,
  enabled = true,
}: UsePlayerKeyboardOptions): void {
  // The handler is bound once; everything it needs is read fresh through a ref,
  // which is synced in an effect rather than during render (a ref written while
  // rendering is a value React is free to discard).
  const latest = useRef({ actions, duration, onCommand });
  useEffect(() => {
    latest.current = { actions, duration, onCommand };
  }, [actions, duration, onCommand]);

  useEffect(() => {
    if (!enabled) return;

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.metaKey || event.ctrlKey || event.altKey) return;

      // Not our keystroke unless the viewer has deliberately put focus in the
      // player — by clicking it (the frame is focusable, so a click lands there)
      // or by tabbing to it. Anywhere else, Space still scrolls the page.
      const container = containerRef.current;
      if (!container || !container.contains(document.activeElement)) return;

      const target = event.target;
      if (target instanceof Element && target.closest(CLAIMS_KEYS)) return;

      const { actions: act, duration: total, onCommand: wake } = latest.current;

      const handled = (): boolean => {
        switch (event.key) {
          case " ":
          case "k":
          case "K":
            act.togglePlay();
            return true;
          case "j":
          case "J":
            act.seekBy(-10);
            return true;
          case "l":
          case "L":
            act.seekBy(10);
            return true;
          case "ArrowLeft":
            act.seekBy(-5);
            return true;
          case "ArrowRight":
            act.seekBy(5);
            return true;
          case "ArrowUp":
            act.nudgeVolume(0.05);
            return true;
          case "ArrowDown":
            act.nudgeVolume(-0.05);
            return true;
          case "m":
          case "M":
            act.toggleMute();
            return true;
          case "f":
          case "F":
            act.toggleFullscreen();
            return true;
          case "Home":
            act.seekTo(0);
            return true;
          case "End":
            if (total > 0) act.seekTo(total);
            return true;
          default:
            // 0-9 jump to that tenth of the video. Meaningless without a duration.
            if (/^[0-9]$/.test(event.key) && total > 0) {
              act.seekTo((Number(event.key) / 10) * total);
              return true;
            }
            return false;
        }
      };

      if (handled()) {
        // Safe now, and only now: the player holds focus, so Space and the arrows
        // are ours to take. This line is why the focus guard above exists.
        event.preventDefault();
        wake();
      }
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [enabled, containerRef]);
}
