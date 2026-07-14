/** A renderable HLS variant, derived from hls.js levels. */
export interface QualityLevel {
  /** Index into hls.levels; what `hls.currentLevel` expects. */
  index: number;
  /** Vertical resolution, e.g. 720. */
  height: number;
  bitrate: number;
}

/** A playback failure the player can show and the user can retry. */
export interface PlayerFailure {
  title: string;
  description: string;
}

/**
 * The uploader, as much of them as the API will admit to.
 *
 * There is no public user-profile endpoint — a `Video` carries `user_id` and
 * nothing else — so `getUploader` in api.ts reconstructs the rest where it
 * honestly can and answers null where it cannot.
 */
export interface Channel {
  id: string;
  username: string;
  avatarUrl: string | null;
  verified: boolean;
  /** Null when the count could not be read. Zero is a real answer, and is not null. */
  subscriberCount: number | null;
  /** True when the viewer is watching their own upload. */
  isViewer: boolean;
}

/** The speeds the settings menu offers. */
export const PLAYBACK_RATES = [0.5, 0.75, 1, 1.25, 1.5, 1.75, 2] as const;

export type PlaybackRate = (typeof PLAYBACK_RATES)[number];
