import "server-only";

import { toVideoCard } from "@/features/videos/card-data";
import type { PlaylistRow } from "@/features/playlists/types";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";
import type { Page, PageParams, Playlist, PlaylistItem } from "@/types/common";

/** The caller's playlists, private ones included. */
export async function listMyPlaylists(params: PageParams = {}): Promise<Page<Playlist>> {
  return api.page<Playlist>("/me/playlists", {
    query: { page: params.page, limit: params.limit ?? 24 },
  });
}

/**
 * A playlist, or null when it does not exist — or exists but is private and not
 * yours, which the API answers with the same 404 on purpose. The two are
 * deliberately indistinguishable, so the UI must say "not found" and nothing
 * more; claiming "you don't have permission" would confirm the playlist exists.
 */
export async function getPlaylist(playlistId: string): Promise<Playlist | null> {
  try {
    return await api.get<Playlist>(`/playlists/${playlistId}`);
  } catch (error) {
    if (isApiError(error) && error.isNotFound) return null;
    throw error;
  }
}

/** A playlist's videos in position order. Same 404-for-private rule as above. */
export async function listPlaylistVideos(
  playlistId: string,
  params: PageParams = {},
): Promise<Page<PlaylistItem>> {
  return api.page<PlaylistItem>(`/playlists/${playlistId}/videos`, {
    query: { page: params.page, limit: params.limit ?? 24 },
  });
}

/** Normalises the API's items into the shape the rows render, on the server. */
export function toPlaylistRows(items: PlaylistItem[]): PlaylistRow[] {
  return items.map((item) => ({
    position: item.position,
    addedAt: item.added_at,
    video: toVideoCard(item.video),
  }));
}
