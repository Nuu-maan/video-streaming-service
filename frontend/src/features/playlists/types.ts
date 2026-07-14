import type { VideoCardData } from "@/features/videos/types";
import type { Playlist } from "@/types/common";

export interface ActionFailure {
  ok: false;
  code: string;
  message: string;
}

export type PlaylistResult = { ok: true; playlist: Playlist } | ActionFailure;
export type PlaylistMutationResult = { ok: true } | ActionFailure;

/**
 * One row of a playlist.
 *
 * `position` is the server's own ordering key and it is *not* an index: removing
 * a video leaves a gap (the API does not renumber), so position 7 can be the
 * third row on the page. Never derive one from the other — the ordinal a reader
 * sees is computed from the page offset, and every mutation addresses the video
 * by id.
 */
export interface PlaylistRow {
  position: number;
  addedAt: string;
  video: VideoCardData;
}

/**
 * Whether the video is already in a given playlist.
 *
 * `"unknown"` is a real answer, not a shrug, and it is the whole point of this
 * type. The API has no "which playlists contain this video" endpoint, so
 * membership is resolved by reading playlists — and a read can fail (a 429, most
 * likely), or run deeper than we are willing to page. The old code swallowed
 * both cases and returned `false`, which is a LIE with teeth: the row renders
 * unchecked, the user clicks the playlist their video is already in, the toggle
 * takes the ADD branch, and the removal they asked for silently never happens.
 *
 * So the three states are distinct, and the UI shows the third one honestly: an
 * indeterminate checkbox that can still be used to add (adding is idempotent —
 * a 409 is swallowed) but never claims the video is absent.
 */
export type Membership = boolean | "unknown";

/** A playlist as offered in the save dialog, with the video's membership resolved. */
export interface SaveTarget {
  id: string;
  title: string;
  videoCount: number;
  visibility: Playlist["visibility"];
  contains: Membership;
}

export type SaveTargetsResult = { ok: true; targets: SaveTarget[] } | ActionFailure;
