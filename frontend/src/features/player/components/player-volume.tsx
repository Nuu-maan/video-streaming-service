"use client";

import { Volume1, Volume2, VolumeX } from "lucide-react";
import { memo, useCallback, useRef, useState } from "react";

import { PlayerButton } from "@/features/player/components/player-button";

interface PlayerVolumeProps {
  volume: number;
  muted: boolean;
  onVolumeChange: (volume: number) => void;
  onToggleMute: () => void;
}

const STEP = 0.05;

/**
 * Mute button plus a slider.
 *
 * Written by hand rather than with the shadcn/Radix slider for one reason: that
 * component's thumb — the element that actually carries `role="slider"` — cannot
 * be given an accessible name from the outside, so it announces itself as
 * "Value". A control a screen-reader user reaches for as often as volume has to
 * say what it is.
 *
 * Full width at all times. A slider that expands on hover animates `width`,
 * which relays out the entire control bar every time the pointer drifts past.
 *
 * `memo`, because its owner re-renders four times a second for the whole length
 * of a film (every `timeupdate` mirrors `currentTime` into VideoPlayer's state)
 * and none of it is any of this component's business. Its props are volume,
 * muted, and two callbacks that `usePlayerState` hands out from a `useMemo` — so
 * they are referentially stable and this genuinely stops re-rendering rather
 * than merely pretending to.
 */
function PlayerVolumeImpl({ volume, muted, onVolumeChange, onToggleMute }: PlayerVolumeProps) {
  const trackRef = useRef<HTMLDivElement>(null);
  const [dragging, setDragging] = useState(false);

  const level = muted ? 0 : volume;
  const Icon = level === 0 ? VolumeX : level < 0.5 ? Volume1 : Volume2;
  const percent = Math.round(level * 100);

  const setFromPointer = useCallback(
    (clientX: number) => {
      const track = trackRef.current;
      if (!track) return;
      const rect = track.getBoundingClientRect();
      if (rect.width === 0) return;
      onVolumeChange(Math.min(Math.max((clientX - rect.left) / rect.width, 0), 1));
    },
    [onVolumeChange],
  );

  const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    const set = (next: number) => {
      event.preventDefault();
      onVolumeChange(Math.min(Math.max(next, 0), 1));
    };
    switch (event.key) {
      case "ArrowLeft":
      case "ArrowDown":
        return set(level - STEP);
      case "ArrowRight":
      case "ArrowUp":
        return set(level + STEP);
      case "Home":
        return set(0);
      case "End":
        return set(1);
      default:
        return;
    }
  };

  return (
    <div className="flex items-center gap-0.5">
      <PlayerButton label={level === 0 ? "Unmute (m)" : "Mute (m)"} onClick={onToggleMute}>
        <Icon aria-hidden />
      </PlayerButton>

      <div
        role="slider"
        tabIndex={0}
        aria-label="Volume"
        aria-orientation="horizontal"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={percent}
        aria-valuetext={`${percent}%`}
        onKeyDown={handleKeyDown}
        onPointerDown={(event) => {
          if (event.button !== 0) return;
          event.currentTarget.setPointerCapture(event.pointerId);
          setDragging(true);
          setFromPointer(event.clientX);
        }}
        onPointerMove={(event) => {
          if (dragging) setFromPointer(event.clientX);
        }}
        onPointerUp={(event) => {
          if (!dragging) return;
          event.currentTarget.releasePointerCapture(event.pointerId);
          setDragging(false);
        }}
        // @container so the thumb can be placed in `cqw` — see player-scrubber
        // for why `left` had to go and why the offset is a standalone `translate`.
        className="group/vol @container relative hidden h-8 w-20 shrink-0 cursor-pointer touch-none items-center rounded-full outline-none select-none focus-visible:ring-2 focus-visible:ring-white/80 sm:flex"
      >
        <div ref={trackRef} className="relative h-1 w-full overflow-hidden rounded-full bg-white/25">
          <div
            className="absolute inset-0 origin-left rounded-full bg-white"
            style={{ transform: `scaleX(${level})` }}
          />
        </div>
        <div
          aria-hidden
          className="pointer-events-none absolute top-1/2 left-0 size-3 scale-0 rounded-full bg-white shadow-sm transition-[scale] duration-(--motion-fast) ease-out-quart group-hover/vol:scale-100 group-focus-visible/vol:scale-100"
          style={{ translate: `calc(${level * 100}cqw - 50%) -50%` }}
        />
      </div>
    </div>
  );
}

export const PlayerVolume = memo(PlayerVolumeImpl);
