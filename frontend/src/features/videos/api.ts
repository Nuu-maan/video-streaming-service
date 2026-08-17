import "server-only";

import { api } from "@/lib/api-client";
import type { TrendingWindow } from "@/features/videos/types";
import type {
  CategoryCount,
  Page,
  PageParams,
  Video,
  VideoSearchItem,
  VideoStatusReport,
} from "@/types/common";

/**
 * Read layer for the video domain. Server Components call these directly;
 * client components go through `actions.ts`.
 *
 * Caching policy: strictly-public listings opt into short time-based
 * revalidation under the `videos` tag (which `deleteVideo` busts). They are
 * fetched WITHOUT auth on purpose — a cached response must never be one user's
 * personalised view served to another, and an anonymous request is the only
 * request that is the same for everyone. Anything owner- or viewer-specific
 * stays no-store.
 */

interface ListVideosParams extends PageParams {
  /** List the caller's own videos (requires a session). */
  mine?: boolean;
}

export function listVideos({ page, limit, mine }: ListVideosParams = {}): Promise<Page<Video>> {
  if (mine) {
    // The caller's own library: personalised, never cached.
    return api.page<Video>("/videos", { query: { page, limit, mine: "true" } });
  }
  return api.page<Video>("/videos", {
    query: { page, limit },
    auth: false,
    revalidate: 60,
    tags: ["videos"],
  });
}

/**
 * Authenticated and uncached: the owner of a private video must see it, and a
 * 404 here genuinely means "not found or not yours to know about".
 */
export function getVideo(id: string): Promise<Video> {
  return api.get<Video>(`/videos/${id}`);
}

export function getVideoStatus(id: string): Promise<VideoStatusReport> {
  return api.get<VideoStatusReport>(`/videos/${id}/status`);
}

export async function getTrending(window: TrendingWindow = "24h", limit = 12): Promise<VideoSearchItem[]> {
  // The API answers an empty list with `data: null`, not `[]` — same quirk the
  // paginated client already absorbs; the plain-array endpoints absorb it here.
  const items = await api.get<VideoSearchItem[] | null>("/videos/trending", {
    query: { window, limit },
    auth: false,
    revalidate: 300,
    tags: ["videos"],
  });
  return items ?? [];
}

/** The subscription feed — personalised by definition, so never cached. */
export function getFeed({ page, limit }: PageParams = {}): Promise<Page<VideoSearchItem>> {
  return api.page<VideoSearchItem>("/me/feed", { query: { page, limit } });
}

export async function getCategories(): Promise<CategoryCount[]> {
  const categories = await api.get<CategoryCount[] | null>("/categories", {
    auth: false,
    revalidate: 300,
    tags: ["videos"],
  });
  return categories ?? [];
}
