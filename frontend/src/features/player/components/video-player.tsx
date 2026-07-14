"use client";

import { LoaderCircle, Maximize, Minimize, Pause, PictureInPicture2, Play, RotateCcw } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import { IconSwap } from "@/components/common/icon-swap";
import { PlayerButton } from "@/features/player/components/player-button";
import { PlayerError } from "@/features/player/components/player-error";
import { PlayerScrubber } from "@/features/player/components/player-scrubber";
import { PlayerSettings } from "@/features/player/components/player-settings";
import { PlayerVolume } from "@/features/player/components/player-volume";
import { useControlsVisibility } from "@/features/player/hooks/use-controls-visibility";
import { useHls } from "@/features/player/hooks/use-hls";
import { usePlayerKeyboard } from "@/features/player/hooks/use-player-keyboard";
import { usePlayerState } from "@/features/player/hooks/use-player-state";
import { useViewTracking } from "@/features/player/hooks/use-view-tracking";
import { useWatchProgress } from "@/features/player/hooks/use-watch-progress";
import { formatDuration } from "@/lib/format";
import { cn } from "@/lib/utils";

interface VideoPlayerProps {
  videoId: string;
  /**
   * The manifest URL. The PAGE decides this, not the player: a public video is
   * fetched straight from the API origin (cheap, cacheable), and a private or
   * unlisted one goes through this origin's `/api/media` proxy, which attaches
   * the bearer token that hls.js cannot. That decision needs the video's
   * visibility and a server, and this component has neither.
   */
  src: string;
  poster?: string | null;
  /** Used for the fullscreen title bar and the media-element label. */
  title: string;
  /** Progress is a per-user record; anonymous viewers have nowhere to store one. */
  trackProgress?: boolean;
  /** Seconds to resume from, read from the viewer's history on the server. */
  resumeAt?: number | null;
  className?: string;
}

/**
 * The player.
 *
 * Custom controls rather than the browser's, because the browser's cannot show
 * an HLS quality menu — it does not know the levels exist. Everything else
 * follows from owning the chrome: keyboard shortcuts, auto-hide, a scrub bar
 * with buffer and hover preview, PiP, speed.
 *
 * State lives in the media element and is mirrored into React by
 * `usePlayerState`; hls.js lives in `useHls`, which owns its lifecycle and — the
 * part that matters — destroys the instance on unmount, so a viewer who watches
 * twenty videos does not accumulate twenty media pipelines and a dead tab.
 */
export function VideoPlayer({
  videoId,
  src,
  poster,
  title,
  trackProgress = false,
  resumeAt,
  className,
}: VideoPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const [state, actions] = usePlayerState({ videoRef, containerRef });
  const { levels, currentLevel, activeHeight, failure, setLevel, retry } = useHls({ videoRef, src });

  // Pin the controls open whenever the viewer is clearly still using them.
  const [menuOpen, setMenuOpen] = useState(false);
  const [scrubbing, setScrubbing] = useState(false);
  const [focusWithin, setFocusWithin] = useState(false);
  const { visible, wake } = useControlsVisibility({
    containerRef,
    playing: state.playing,
    hold: menuOpen || scrubbing || focusWithin,
  });

  usePlayerKeyboard({ actions, duration: state.duration, containerRef, onCommand: wake });

  /**
   * Keep focus inside the frame when the viewer clicks the picture or the big
   * play button.
   *
   * Two reasons. The shortcuts are scoped to "focus is inside the player" (see
   * usePlayerKeyboard — they preventDefault the page's own scroll keys, so they
   * must not be live when the player is merely on the page). And the big play
   * button goes `inert` the instant playback starts, which would otherwise BLOW
   * AWAY the focus the click just gave it and dump the viewer back on <body> —
   * where their next space bar scrolls the page instead of pausing the film.
   *
   * preventScroll, because the player is already in view and nudging the page
   * under a click nobody asked to navigate with is its own small betrayal.
   */
  const grabFocus = useCallback(() => {
    containerRef.current?.focus({ preventScroll: true });
  }, []);

  /**
   * The rendition the view is being watched at, for the view ping. A ref, not a
   * prop: the tracking hook must read the value at the moment the threshold is
   * crossed without re-subscribing to media events every time ABR switches.
   */
  const activeQuality =
    currentLevel >= 0
      ? `${levels.find((level) => level.index === currentLevel)?.height ?? activeHeight ?? ""}p`
      : activeHeight
        ? `${activeHeight}p`
        : "auto";
  const qualityRef = useRef(activeQuality);
  useEffect(() => {
    qualityRef.current = activeQuality;
  }, [activeQuality]);

  useViewTracking({ videoId, videoRef, qualityRef });

  const onResume = useCallback(
    (seconds: number) => {
      toast(`Resumed from ${formatDuration(seconds)}`, {
        action: {
          label: "Start over",
          onClick: () => actions.seekTo(0),
        },
      });
    },
    [actions],
  );

  useWatchProgress({ videoId, videoRef, enabled: trackProgress, resumeAt, onResume });

  const handleRetry = () => {
    retry();
    wake();
  };

  const showBigPlay = !failure && !state.playing && !state.waiting;

  return (
    /*
     * The frame is focusable, and that is load-bearing rather than decorative:
     * it is the thing the keyboard shortcuts are scoped to. Clicking the picture
     * focuses it, Tab reaches it, and only then do J/K/L/Space/arrows belong to
     * the player rather than to the page.
     */
    <div
      ref={containerRef}
      tabIndex={0}
      role="region"
      aria-label={`Video player: ${title}`}
      onFocus={wake}
      data-controls={visible ? "visible" : "hidden"}
      className={cn(
        "group/player relative aspect-video w-full overflow-hidden rounded-xl bg-black outline-none select-none",
        "focus-visible:ring-3 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background",
        // The cursor is chrome too — it has no business sitting over a film.
        !visible && "cursor-none",
        className,
      )}
    >
      <video
        ref={videoRef}
        poster={poster ?? undefined}
        preload="metadata"
        playsInline
        aria-label={title}
        className="size-full bg-black object-contain"
        onClick={() => {
          actions.togglePlay();
          wake();
          grabFocus();
        }}
        onDoubleClick={actions.toggleFullscreen}
      />

      {/*
       * The live region is mounted always and only its TEXT changes. A region
       * that is inserted into the DOM already carrying its message is the classic
       * aria-live bug: the accessibility tree has nothing to diff it against, and
       * screen readers routinely say nothing at all. Mounted empty and filled
       * later, it announces. The visible spinner stays conditional — it has no
       * such constraint.
       */}
      <span className="sr-only" role="status">
        {state.waiting && !failure ? "Buffering" : ""}
      </span>

      {/* Buffering. Distinct from paused: the viewer asked for this and is
          waiting, so the spinner belongs where their eyes already are. */}
      {state.waiting && !failure ? (
        <div className="pointer-events-none absolute inset-0 z-10 flex items-center justify-center">
          <LoaderCircle aria-hidden className="size-10 animate-spin text-white/80" />
        </div>
      ) : null}

      {/* The one control big enough to hit without looking. It fades and scales
          rather than appearing, so a pause reads as a state change, not a cut. */}
      <div
        inert={!showBigPlay}
        className={cn(
          "pointer-events-none absolute inset-0 z-10 flex items-center justify-center transition-[opacity,scale] duration-(--motion-medium) ease-out-quart",
          showBigPlay ? "scale-100 opacity-100" : "scale-90 opacity-0",
        )}
      >
        <button
          type="button"
          aria-label={state.ended ? "Replay" : "Play"}
          onClick={() => {
            actions.togglePlay();
            wake();
            // This button is about to become `inert`. Hand focus back to the
            // frame before it does, or the browser drops it on <body>.
            grabFocus();
          }}
          className={cn(
            "flex size-16 items-center justify-center rounded-full bg-black/55 text-white outline-none backdrop-blur-sm",
            "transition-[background-color,scale] duration-(--motion-fast) ease-out-quart",
            "hover:bg-black/70 focus-visible:ring-2 focus-visible:ring-white/80 active:scale-96",
            showBigPlay && "pointer-events-auto",
          )}
        >
          {state.ended ? (
            <RotateCcw aria-hidden className="size-7" />
          ) : (
            /* Optical, not geometric: a play triangle's visual mass sits left of
               its bounding box, so it is nudged right to look centred. */
            <Play aria-hidden className="size-7 translate-x-0.5 fill-current" />
          )}
        </button>
      </div>

      {/* Fullscreen puts the video where the page's own title no longer is. */}
      {state.fullscreen ? (
        <div
          className={cn(
            "pointer-events-none absolute inset-x-0 top-0 z-10 bg-gradient-to-b from-black/70 to-transparent px-4 pt-3 pb-10 transition-opacity duration-(--motion-medium) ease-out-quart",
            visible ? "opacity-100" : "opacity-0",
          )}
        >
          <p className="line-clamp-1 text-sm font-medium text-white/90">{title}</p>
        </div>
      ) : null}

      {failure ? <PlayerError failure={failure} onRetry={handleRetry} /> : null}

      {/* `inert` rather than a pile of tabIndex={-1}: hidden controls must be
          unreachable by Tab as well as by the pointer, and a menu trigger inside
          a portal-owning component cannot be given a tabIndex from out here. */}
      <div
        inert={!visible || Boolean(failure)}
        onFocusCapture={() => setFocusWithin(true)}
        onBlurCapture={(event) => {
          if (!event.currentTarget.contains(event.relatedTarget)) setFocusWithin(false);
        }}
        className={cn(
          "absolute inset-x-0 bottom-0 z-10 flex flex-col gap-0.5 bg-gradient-to-t from-black/85 via-black/45 to-transparent px-2 pt-12 pb-1 transition-opacity duration-(--motion-medium) ease-out-quart sm:px-3",
          visible && !failure ? "opacity-100" : "pointer-events-none opacity-0",
        )}
      >
        <div className="px-1">
          <PlayerScrubber
            currentTime={state.currentTime}
            duration={state.duration}
            bufferedTo={state.bufferedTo}
            onSeek={actions.seekTo}
            onScrubbingChange={setScrubbing}
          />
        </div>

        <div className="flex items-center gap-0.5">
          {/* Deliberately NOT an IconSwap. Play/pause is the space bar, pressed
              dozens of times in a single sitting, and the animation-frequency
              rule is unambiguous about keys used that often: no animation, ever.
              A cross-fade here would read as lag on the one control that must
              feel instantaneous. Fullscreen and Subscribe, below and elsewhere,
              are rare and deliberate — those animate. */}
          <PlayerButton
            label={state.playing ? "Pause (k)" : "Play (k)"}
            onClick={actions.togglePlay}
          >
            {state.playing ? <Pause aria-hidden className="fill-current" /> : <Play aria-hidden className="fill-current" />}
          </PlayerButton>

          <PlayerVolume
            volume={state.volume}
            muted={state.muted}
            onVolumeChange={actions.setVolume}
            onToggleMute={actions.toggleMute}
          />

          <p className="ml-1.5 text-xs font-medium text-white/90 tabular-nums">
            <span>{formatDuration(state.currentTime)}</span>
            <span className="mx-1 text-white/45">/</span>
            <span className="text-white/70">{formatDuration(state.duration)}</span>
          </p>

          <div className="ml-auto flex items-center gap-0.5">
            <PlayerSettings
              levels={levels}
              currentLevel={currentLevel}
              activeHeight={activeHeight}
              rate={state.rate}
              onLevelChange={setLevel}
              onRateChange={actions.setRate}
              onOpenChange={setMenuOpen}
            />

            {state.pipSupported ? (
              <PlayerButton
                label={state.pip ? "Exit picture in picture" : "Picture in picture"}
                onClick={actions.togglePip}
                className="hidden sm:inline-flex"
              >
                <PictureInPicture2 aria-hidden />
              </PlayerButton>
            ) : null}

            {state.fullscreenSupported ? (
              <PlayerButton
                label={state.fullscreen ? "Exit fullscreen (f)" : "Fullscreen (f)"}
                onClick={actions.toggleFullscreen}
              >
                {/* Occasional and deliberate — this is exactly the swap that
                    wants the state-change animation. */}
                <IconSwap
                  className="size-5"
                  active={state.fullscreen}
                  from={<Maximize aria-hidden className="size-5" />}
                  to={<Minimize aria-hidden className="size-5" />}
                />
              </PlayerButton>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
}
