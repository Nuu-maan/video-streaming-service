"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import type { PlaybackRate } from "@/features/player/types";

interface UsePlayerStateOptions {
  videoRef: React.RefObject<HTMLVideoElement | null>;
  /** The element that goes fullscreen — the frame, not the <video>, so controls come with it. */
  containerRef: React.RefObject<HTMLElement | null>;
}

export interface PlayerState {
  playing: boolean;
  /** Stalled waiting for data. Distinct from paused: the user did not ask for this. */
  waiting: boolean;
  ended: boolean;
  /** True until the first frame is decodable — the poster is still showing. */
  loading: boolean;
  currentTime: number;
  duration: number;
  /** Seconds buffered ahead of the playhead, absolute (not a delta). */
  bufferedTo: number;
  volume: number;
  muted: boolean;
  rate: PlaybackRate;
  fullscreen: boolean;
  pip: boolean;
  fullscreenSupported: boolean;
  pipSupported: boolean;
}

export interface PlayerActions {
  play: () => void;
  pause: () => void;
  togglePlay: () => void;
  /** Absolute seek, in seconds. Clamped to [0, duration]. */
  seekTo: (seconds: number) => void;
  /** Relative seek. `seekBy(-10)` is the J key. */
  seekBy: (delta: number) => void;
  setVolume: (volume: number) => void;
  /** Relative volume nudge, for the arrow keys. */
  nudgeVolume: (delta: number) => void;
  toggleMute: () => void;
  setRate: (rate: PlaybackRate) => void;
  toggleFullscreen: () => void;
  togglePip: () => void;
}

/** Volume survives the session — re-muting every video is a small, constant insult. */
const VOLUME_KEY = "reel.volume";

const INITIAL: PlayerState = {
  playing: false,
  waiting: false,
  ended: false,
  loading: true,
  currentTime: 0,
  duration: 0,
  bufferedTo: 0,
  volume: 1,
  muted: false,
  rate: 1,
  fullscreen: false,
  pip: false,
  fullscreenSupported: false,
  pipSupported: false,
};

/**
 * Everything the custom controls need to know about the media element, and
 * every way they are allowed to change it.
 *
 * The <video> element is the single source of truth — React state only mirrors
 * it, driven by media events. Nothing here writes state optimistically: press
 * play and the state flips when the element says it is playing, not when we
 * asked it to. That is what keeps the button honest when autoplay is blocked or
 * a seek is refused.
 */
export function usePlayerState({ videoRef, containerRef }: UsePlayerStateOptions): [PlayerState, PlayerActions] {
  const [state, setState] = useState<PlayerState>(INITIAL);
  const patch = useCallback((next: Partial<PlayerState>) => {
    setState((prev) => ({ ...prev, ...next }));
  }, []);

  /** Volume before the last mute, so unmuting restores it rather than jumping to 1. */
  const volumeBeforeMute = useRef(1);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    // Capability probes have to run in the browser: on the server there is no
    // document, and rendering a PiP button that does nothing is worse than not
    // rendering one.
    patch({
      fullscreenSupported: typeof document !== "undefined" && document.fullscreenEnabled,
      pipSupported: typeof document !== "undefined" && document.pictureInPictureEnabled,
    });

    const stored = Number(localStorage.getItem(VOLUME_KEY));
    if (Number.isFinite(stored) && stored >= 0 && stored <= 1) {
      video.volume = stored;
      volumeBeforeMute.current = stored || 1;
    }

    const readBuffered = (): number => {
      const ranges = video.buffered;
      const time = video.currentTime;
      for (let i = 0; i < ranges.length; i += 1) {
        if (ranges.start(i) <= time && time <= ranges.end(i)) return ranges.end(i);
      }
      return time;
    };

    const onTimeUpdate = () => patch({ currentTime: video.currentTime, bufferedTo: readBuffered() });
    const onProgress = () => patch({ bufferedTo: readBuffered() });
    const onDurationChange = () =>
      patch({ duration: Number.isFinite(video.duration) ? video.duration : 0 });
    const onPlay = () => patch({ playing: true, ended: false });
    const onPause = () => patch({ playing: false });
    const onPlaying = () => patch({ playing: true, waiting: false, loading: false, ended: false });
    const onWaiting = () => patch({ waiting: true });
    const onEnded = () => patch({ playing: false, ended: true, waiting: false });
    const onLoadedMetadata = () =>
      patch({
        duration: Number.isFinite(video.duration) ? video.duration : 0,
        loading: false,
      });
    const onVolumeChange = () => {
      if (!video.muted && video.volume > 0) volumeBeforeMute.current = video.volume;
      try {
        localStorage.setItem(VOLUME_KEY, String(video.volume));
      } catch {
        // Storage denied. The volume still works; it just forgets.
      }
      patch({ volume: video.volume, muted: video.muted || video.volume === 0 });
    };
    const onRateChange = () => patch({ rate: video.playbackRate as PlaybackRate });
    const onSeeking = () => patch({ currentTime: video.currentTime });
    const onSeeked = () => patch({ currentTime: video.currentTime, bufferedTo: readBuffered() });
    const onEnterPip = () => patch({ pip: true });
    const onLeavePip = () => patch({ pip: false });

    const events: Array<[string, () => void]> = [
      ["timeupdate", onTimeUpdate],
      ["progress", onProgress],
      ["durationchange", onDurationChange],
      ["loadedmetadata", onLoadedMetadata],
      ["canplay", onLoadedMetadata],
      ["play", onPlay],
      ["pause", onPause],
      ["playing", onPlaying],
      ["waiting", onWaiting],
      ["ended", onEnded],
      ["volumechange", onVolumeChange],
      ["ratechange", onRateChange],
      ["seeking", onSeeking],
      ["seeked", onSeeked],
      ["enterpictureinpicture", onEnterPip],
      ["leavepictureinpicture", onLeavePip],
    ];

    for (const [name, handler] of events) video.addEventListener(name, handler);

    // The element may already be further along than the initial state believes —
    // hls.js can attach and parse before this effect ever runs.
    onVolumeChange();
    onDurationChange();

    const onFullscreenChange = () =>
      patch({ fullscreen: document.fullscreenElement === containerRef.current });
    document.addEventListener("fullscreenchange", onFullscreenChange);

    return () => {
      for (const [name, handler] of events) video.removeEventListener(name, handler);
      document.removeEventListener("fullscreenchange", onFullscreenChange);
    };
  }, [videoRef, containerRef, patch]);

  const actions = useMemo<PlayerActions>(() => {
    const el = () => videoRef.current;

    const play = () => {
      // A rejected play() (autoplay policy, no user gesture) is not an error the
      // viewer needs to read — the play button simply stays a play button.
      void el()?.play().catch(() => undefined);
    };
    const pause = () => el()?.pause();

    const seekTo = (seconds: number) => {
      const video = el();
      if (!video || !Number.isFinite(video.duration)) return;
      video.currentTime = Math.min(Math.max(seconds, 0), video.duration);
    };

    const setVolume = (volume: number) => {
      const video = el();
      if (!video) return;
      const next = Math.min(Math.max(volume, 0), 1);
      video.volume = next;
      // Dragging the slider off zero is an unmute; dragging it to zero is a mute.
      video.muted = next === 0;
    };

    return {
      play,
      pause,
      togglePlay: () => {
        const video = el();
        if (!video) return;
        if (video.paused || video.ended) play();
        else pause();
      },
      seekTo,
      seekBy: (delta: number) => {
        const video = el();
        if (video) seekTo(video.currentTime + delta);
      },
      setVolume,
      nudgeVolume: (delta: number) => {
        const video = el();
        if (video) setVolume((video.muted ? 0 : video.volume) + delta);
      },
      toggleMute: () => {
        const video = el();
        if (!video) return;
        if (video.muted || video.volume === 0) {
          video.muted = false;
          if (video.volume === 0) video.volume = volumeBeforeMute.current;
        } else {
          video.muted = true;
        }
      },
      setRate: (rate: PlaybackRate) => {
        const video = el();
        if (video) video.playbackRate = rate;
      },
      toggleFullscreen: () => {
        const container = containerRef.current;
        if (!container) return;
        if (document.fullscreenElement) void document.exitFullscreen().catch(() => undefined);
        else void container.requestFullscreen().catch(() => undefined);
      },
      togglePip: () => {
        const video = el();
        if (!video) return;
        if (document.pictureInPictureElement) void document.exitPictureInPicture().catch(() => undefined);
        else void video.requestPictureInPicture().catch(() => undefined);
      },
    };
  }, [videoRef, containerRef]);

  return [state, actions];
}
