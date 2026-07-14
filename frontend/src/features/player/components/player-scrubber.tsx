"use client";

import { useCallback, useRef, useState } from "react";

import { formatDuration } from "@/lib/format";
import { cn } from "@/lib/utils";

interface PlayerScrubberProps {
  currentTime: number;
  duration: number;
  /** Absolute seconds buffered ahead of the playhead. */
  bufferedTo: number;
  onSeek: (seconds: number) => void;
  /** Fires on pointer-down and on release: the caller pins the controls open while dragging. */
  onScrubbingChange: (scrubbing: boolean) => void;
  className?: string;
}

/** Arrow keys move 5s; PageUp/PageDown move a minute. Same contract as the global keys. */
const STEP_SECONDS = 5;
const PAGE_SECONDS = 60;

/**
 * The scrub bar: played, buffered, and a hover preview of the time under the
 * cursor.
 *
 * NOTHING HERE IS POSITIONED WITH A LAYOUT PROPERTY. The bar grows on hover by
 * `scaleY`, never `height`. The fill is a `scaleX` on a full-width element,
 * never a percentage `width`. And the thumb and the hover label are placed with
 * `translate`, never `left` — which is what they used to use, and which quietly
 * undid the whole point: `left` was being rewritten on every timeupdate (4/s for
 * the length of a film) and on every pointermove (60/s just to drag the mouse
 * across the bar), forcing layout and paint each time. `transform`/`translate`
 * skip both and run on the compositor.
 *
 * The positioning unit is `cqw`, which is why the root is a `@container`:
 * `100cqw` is the track's own width, so `translate: calc(37cqw - 50%)` means
 * "37% along the track, centred on that point" without ever asking the layout
 * engine anything.
 *
 * `translate` (the standalone property) rather than folding the offset into
 * `transform`: the thumb's reveal is a `scale`, and the CSS transform order is
 * translate → rotate → scale → transform, so a `transform: translateX()` would
 * be multiplied by the scale — at `scale(0)` the thumb would collapse to the far
 * left of the bar and then fly in from there as it grew. Standalone `translate`
 * is applied AFTER `scale`, so the two compose cleanly.
 *
 * It is a real `role="slider"`: focusable, arrow-key seekable, and it announces
 * "1:23 of 4:56" rather than "37 percent".
 */
export function PlayerScrubber({
  currentTime,
  duration,
  bufferedTo,
  onSeek,
  onScrubbingChange,
  className,
}: PlayerScrubberProps) {
  const trackRef = useRef<HTMLDivElement>(null);
  const [dragging, setDragging] = useState(false);
  const [hoverRatio, setHoverRatio] = useState<number | null>(null);

  const ratioAt = useCallback((clientX: number): number => {
    const track = trackRef.current;
    if (!track) return 0;
    const rect = track.getBoundingClientRect();
    if (rect.width === 0) return 0;
    return Math.min(Math.max((clientX - rect.left) / rect.width, 0), 1);
  }, []);

  const seekToPointer = useCallback(
    (clientX: number) => {
      if (duration > 0) onSeek(ratioAt(clientX) * duration);
    },
    [duration, onSeek, ratioAt],
  );

  const played = duration > 0 ? Math.min(currentTime / duration, 1) : 0;
  const buffered = duration > 0 ? Math.min(Math.max(bufferedTo / duration, played), 1) : 0;
  const previewRatio = hoverRatio ?? 0;

  const handlePointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    if (event.button !== 0 || duration <= 0) return;
    // Capture on the element: the pointer will leave the 8px track almost
    // immediately during a drag, and without capture the seek dies with it.
    event.currentTarget.setPointerCapture(event.pointerId);
    setDragging(true);
    onScrubbingChange(true);
    seekToPointer(event.clientX);
  };

  const handlePointerMove = (event: React.PointerEvent<HTMLDivElement>) => {
    setHoverRatio(ratioAt(event.clientX));
    if (dragging) seekToPointer(event.clientX);
  };

  const endDrag = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!dragging) return;
    event.currentTarget.releasePointerCapture(event.pointerId);
    setDragging(false);
    onScrubbingChange(false);
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (duration <= 0) return;
    const jump = (delta: number) => {
      event.preventDefault();
      onSeek(Math.min(Math.max(currentTime + delta, 0), duration));
    };
    switch (event.key) {
      case "ArrowLeft":
        return jump(-STEP_SECONDS);
      case "ArrowRight":
        return jump(STEP_SECONDS);
      case "PageDown":
        return jump(-PAGE_SECONDS);
      case "PageUp":
        return jump(PAGE_SECONDS);
      case "Home":
        event.preventDefault();
        return onSeek(0);
      case "End":
        event.preventDefault();
        return onSeek(duration);
      default:
        return;
    }
  };

  return (
    <div
      role="slider"
      tabIndex={0}
      aria-label="Seek"
      aria-orientation="horizontal"
      aria-valuemin={0}
      aria-valuemax={Math.round(duration)}
      aria-valuenow={Math.round(currentTime)}
      aria-valuetext={`${formatDuration(currentTime)} of ${formatDuration(duration)}`}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={endDrag}
      onPointerCancel={endDrag}
      onPointerLeave={() => setHoverRatio(null)}
      onKeyDown={handleKeyDown}
      data-dragging={dragging || undefined}
      className={cn(
        // @container: makes `cqw` below resolve against this element's width.
        // pointer-coarse:h-11 — the visual track stays 1px, but on touch this is
        // the only seek target in the app and 24px is not a target. The bar is
        // vertically centred, so growing the wrapper costs nothing visually.
        "group/scrub @container relative flex h-6 w-full cursor-pointer touch-none items-center rounded-full outline-none select-none pointer-coarse:h-11",
        "focus-visible:ring-2 focus-visible:ring-white/80",
        className,
      )}
    >
      {/* Hover preview. Clamped away from the edges so the label never hangs off
          the end of the player at 0:00 or at the last frame — the clamp is inside
          the translate now, so following the pointer costs a composite, not a
          layout pass. */}
      {hoverRatio !== null && duration > 0 ? (
        <div
          aria-hidden
          className="pointer-events-none absolute bottom-full left-0 mb-1.5 rounded-md bg-black/85 px-1.5 py-0.5 text-xs font-medium text-white tabular-nums shadow-md"
          style={{
            translate: `calc(clamp(1.75rem, ${previewRatio * 100}cqw, calc(100cqw - 1.75rem)) - 50%)`,
          }}
        >
          {formatDuration(previewRatio * duration)}
        </div>
      ) : null}

      <div
        ref={trackRef}
        className="relative h-1 w-full origin-center overflow-hidden rounded-full bg-white/25 transition-transform duration-(--motion-fast) ease-out-quart group-hover/scrub:scale-y-[2] group-focus-visible/scrub:scale-y-[2] group-data-[dragging]/scrub:scale-y-[2]"
      >
        <div
          className="absolute inset-0 origin-left rounded-full bg-white/35 transition-transform duration-(--motion-medium) ease-out-quart"
          style={{ transform: `scaleX(${buffered})` }}
        />
        {/* No transition: this tracks the playhead, and an eased fill lags
            visibly behind the frame it is supposed to be describing. */}
        <div
          className="absolute inset-0 origin-left rounded-full bg-primary"
          style={{ transform: `scaleX(${played})` }}
        />
      </div>

      {/* transition-[scale], NOT transition-transform: the latter also covers
          `translate`, which would ease the thumb's POSITION and leave it trailing
          120ms behind the playhead it exists to mark. Only the reveal eases. */}
      <div
        aria-hidden
        className="pointer-events-none absolute top-1/2 left-0 size-3.5 scale-0 rounded-full bg-primary shadow-sm transition-[scale] duration-(--motion-fast) ease-out-quart group-hover/scrub:scale-100 group-focus-visible/scrub:scale-100 group-data-[dragging]/scrub:scale-110"
        style={{ translate: `calc(${played * 100}cqw - 50%) -50%` }}
      />
    </div>
  );
}
