"use client";

import { Settings } from "lucide-react";
import { memo } from "react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { PlayerButton } from "@/features/player/components/player-button";
import type { PlaybackRate, QualityLevel } from "@/features/player/types";
import { PLAYBACK_RATES } from "@/features/player/types";

interface PlayerSettingsProps {
  levels: QualityLevel[];
  /** -1 is automatic ABR. */
  currentLevel: number;
  /** The height actually playing, so "Auto" can say what Auto chose. */
  activeHeight: number | null;
  rate: PlaybackRate;
  onLevelChange: (index: number) => void;
  onRateChange: (rate: PlaybackRate) => void;
  /** The controls must not auto-hide out from under an open menu. */
  onOpenChange: (open: boolean) => void;
}

/**
 * Quality and speed, in one menu.
 *
 * Quality is driven entirely by the levels hls.js parsed out of the manifest —
 * never by the video's `available_qualities`, which describes what was
 * transcoded, not what this stream actually offers right now. "Auto" is the
 * default and stays first, and it reports the rendition ABR settled on, because
 * "Auto" alone answers the wrong question: the viewer wants to know what they
 * are getting, not who chose it.
 *
 * On Safari the browser owns HLS and exposes no levels at all, so the quality
 * group simply does not render — an empty menu section is a broken promise.
 *
 * `memo`, and it is the one in this subtree that most needed it: it renders a
 * whole dropdown-menu tree, and its owner re-renders four times a second for the
 * entire length of a film because `timeupdate` mirrors `currentTime` into
 * VideoPlayer's state. None of that has anything to do with quality or speed.
 * Every prop here is either a primitive or a callback that `usePlayerState` /
 * `useHls` hand out from a `useMemo`/`useCallback`, so the memo actually holds.
 */
function PlayerSettingsImpl({
  levels,
  currentLevel,
  activeHeight,
  rate,
  onLevelChange,
  onRateChange,
  onOpenChange,
}: PlayerSettingsProps) {
  const hasQuality = levels.length > 1;

  return (
    <DropdownMenu onOpenChange={onOpenChange}>
      <DropdownMenuTrigger asChild>
        <PlayerButton label="Settings">
          <Settings aria-hidden />
        </PlayerButton>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" side="top" sideOffset={8} className="min-w-44">
        {hasQuality ? (
          <>
            <DropdownMenuLabel>Quality</DropdownMenuLabel>
            <DropdownMenuRadioGroup
              value={String(currentLevel)}
              onValueChange={(value) => onLevelChange(Number(value))}
            >
              <DropdownMenuRadioItem value="-1">
                Auto
                {activeHeight ? (
                  <span className="ml-auto pl-3 text-xs text-muted-foreground tabular-nums">{activeHeight}p</span>
                ) : null}
              </DropdownMenuRadioItem>
              {levels.map((level) => (
                <DropdownMenuRadioItem key={level.index} value={String(level.index)} className="tabular-nums">
                  {level.height}p
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
            <DropdownMenuSeparator />
          </>
        ) : null}

        <DropdownMenuLabel>Speed</DropdownMenuLabel>
        <DropdownMenuRadioGroup
          value={String(rate)}
          onValueChange={(value) => onRateChange(Number(value) as PlaybackRate)}
        >
          {PLAYBACK_RATES.map((option) => (
            <DropdownMenuRadioItem key={option} value={String(option)} className="tabular-nums">
              {option === 1 ? "Normal" : `${option}×`}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export const PlayerSettings = memo(PlayerSettingsImpl);
