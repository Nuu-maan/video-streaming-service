"use server";

import { revalidatePath } from "next/cache";

import { routes } from "@/config/routes";
import { listMyPlaylists, listPlaylistVideos } from "@/features/playlists/api";
import {
  playlistSchema,
  playlistUpdateSchema,
  type PlaylistInput,
  type PlaylistUpdateInput,
} from "@/features/playlists/schemas";
import type {
  ActionFailure,
  Membership,
  PlaylistMutationResult,
  PlaylistResult,
  SaveTarget,
  SaveTargetsResult,
} from "@/features/playlists/types";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";
import type { Playlist } from "@/types/common";

function fail(error: unknown): ActionFailure {
  if (isApiError(error)) {
    if (error.isUnauthorized) {
      return { ok: false, code: "UNAUTHORIZED", message: "Sign in to manage playlists." };
    }
    if (error.isForbidden) {
      return { ok: false, code: "FORBIDDEN", message: "That playlist isn't yours." };
    }
    if (error.isRateLimited) {
      return { ok: false, code: "RATE_LIMITED", message: "Slow down a moment, then try again." };
    }
    /* A private playlist you do not own answers 404, exactly as a missing one
       does. "Not found" is the only honest thing to say. */
    if (error.isNotFound) {
      return { ok: false, code: "NOT_FOUND", message: "Playlist not found." };
    }
    return { ok: false, code: error.code, message: error.message };
  }
  return { ok: false, code: "UNKNOWN", message: "Something went wrong. Please try again." };
}

export async function createPlaylist(input: PlaylistInput): Promise<PlaylistResult> {
  const parsed = playlistSchema.safeParse(input);
  if (!parsed.success) {
    return { ok: false, code: "VALIDATION", message: parsed.error.issues[0].message };
  }

  try {
    const playlist = await api.post<Playlist>("/playlists", { body: parsed.data });
    revalidatePath(routes.playlists);
    return { ok: true, playlist };
  } catch (error) {
    return fail(error);
  }
}

export async function updatePlaylist(
  playlistId: string,
  input: PlaylistUpdateInput,
): Promise<PlaylistResult> {
  const parsed = playlistUpdateSchema.safeParse(input);
  if (!parsed.success) {
    return { ok: false, code: "VALIDATION", message: parsed.error.issues[0].message };
  }

  try {
    const playlist = await api.patch<Playlist>(`/playlists/${playlistId}`, { body: parsed.data });
    revalidatePath(routes.playlists);
    revalidatePath(routes.playlist(playlistId));
    return { ok: true, playlist };
  } catch (error) {
    return fail(error);
  }
}

export async function deletePlaylist(playlistId: string): Promise<PlaylistMutationResult> {
  try {
    await api.delete(`/playlists/${playlistId}`);
  } catch (error) {
    if (!(isApiError(error) && error.isNotFound)) return fail(error);
  }
  revalidatePath(routes.playlists);
  return { ok: true };
}

/** Appends to the end of the playlist. A video already in it is a 409 — and a no-op. */
export async function addVideoToPlaylist(
  playlistId: string,
  videoId: string,
): Promise<PlaylistMutationResult> {
  try {
    await api.post(`/playlists/${playlistId}/videos`, { body: { video_id: videoId } });
  } catch (error) {
    // ALREADY_IN_PLAYLIST means the desired end state is already the actual one.
    if (!(isApiError(error) && error.status === 409)) return fail(error);
  }
  revalidatePath(routes.playlist(playlistId));
  return { ok: true };
}

export async function removeVideoFromPlaylist(
  playlistId: string,
  videoId: string,
): Promise<PlaylistMutationResult> {
  try {
    await api.delete(`/playlists/${playlistId}/videos/${videoId}`);
  } catch (error) {
    if (!(isApiError(error) && error.isNotFound)) return fail(error);
  }
  revalidatePath(routes.playlist(playlistId));
  return { ok: true };
}

/**
 * `contains` is the membership we BELIEVE, and only a confirmed `true` is
 * allowed to trigger a removal. "unknown" adds — which is the safe direction: an
 * add is idempotent (a 409 means the desired end state was already the actual
 * one), whereas a removal fired on a guess destroys something the user did not
 * ask to lose.
 */
export async function toggleVideoInPlaylist(
  playlistId: string,
  videoId: string,
  contains: Membership,
): Promise<PlaylistMutationResult> {
  return contains === true
    ? removeVideoFromPlaylist(playlistId, videoId)
    : addVideoToPlaylist(playlistId, videoId);
}

/**
 * The save dialog's data: the caller's playlists, each flagged with whether the
 * video is already in it.
 *
 * ── THE BUDGET ───────────────────────────────────────────────────────────────
 * There is no "which playlists contain this video" endpoint, so membership can
 * only be resolved by READING playlists — one request each. That is a fan-out,
 * and a fan-out against a 60-requests-per-minute limit needs a hard ceiling, not
 * a hopeful one. This used to read up to 25 playlists × 2 pages = up to 51
 * requests to open a popover: 85% of the viewer's entire minute, spent because
 * they hovered a button. Opening it twice guaranteed a 429.
 *
 * Now: one request for the list, plus at most PROBE_BUDGET probes, and an empty
 * playlist is not probed at all because it demonstrably contains nothing. Worst
 * case is 13 requests. Every playlist the user owns is still listed and still
 * addable — nothing is hidden from them.
 *
 * ── AND WHY `unknown` EXISTS ─────────────────────────────────────────────────
 * Whatever we do not read, and whatever fails to read, is reported as `unknown`
 * rather than `false`. This is the part that was an actual correctness bug: the
 * old `catch { return false }` turned a rate-limited read into the positive
 * claim "this video is not in that playlist". The dialog then rendered the
 * checkbox unchecked, the user clicked the playlist their video was ALREADY in
 * meaning to remove it, `toggleVideoInPlaylist` was handed `contains: false` and
 * took the ADD branch — and the 409 swallow hid the whole thing. The user's
 * removal silently did nothing, and the code looked like it worked.
 *
 * A read that failed is not evidence of absence. It is the absence of evidence.
 */
const TARGET_LIMIT = 25;
const PROBE_BUDGET = 12;
const MEMBERSHIP_LIMIT = 100;

async function playlistContains(playlist: Playlist, videoId: string): Promise<Membership> {
  // Free, and correct: an empty playlist contains nothing. No request.
  if (playlist.video_count === 0) return false;

  try {
    const result = await listPlaylistVideos(playlist.id, { page: 1, limit: MEMBERSHIP_LIMIT });
    if (result.items.some((item) => item.video.id === videoId)) return true;
    // Not in the first hundred, and there are more — we genuinely do not know.
    return result.pagination.has_next ? "unknown" : false;
  } catch {
    return "unknown";
  }
}

export async function loadSaveTargets(videoId: string): Promise<SaveTargetsResult> {
  try {
    const playlists = await listMyPlaylists({ page: 1, limit: TARGET_LIMIT });

    // Probes are spent in list order (newest first), on non-empty playlists only.
    let probesLeft = PROBE_BUDGET;

    const targets: SaveTarget[] = await Promise.all(
      playlists.items.map((playlist) => {
        const affordable = playlist.video_count === 0 || probesLeft > 0;
        if (playlist.video_count > 0 && affordable) probesLeft -= 1;

        const contains: Promise<Membership> = affordable
          ? playlistContains(playlist, videoId)
          : Promise.resolve("unknown");

        return contains.then((resolved) => ({
          id: playlist.id,
          title: playlist.title,
          videoCount: playlist.video_count,
          visibility: playlist.visibility,
          contains: resolved,
        }));
      }),
    );

    return { ok: true, targets };
  } catch (error) {
    return fail(error);
  }
}
