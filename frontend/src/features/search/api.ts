import "server-only";

import type { SearchQuery, TrendingWindow } from "@/features/search/types";
import { api } from "@/lib/api-client";
import type { CategoryCount, Page, VideoSearchItem } from "@/types/common";

/**
 * Discovery reads. Everything here is a public endpoint (`security: []` in the
 * spec) — search only ever answers with ready, public videos, which is a
 * property of the API, not something this layer filters. `auth: false` keeps
 * anonymous visitors from paying a cookie read, and lets the truly static
 * reads (categories, trending, related) opt into time-based revalidation.
 */

/** `GET /search` — full-text search, standard paginated envelope. */
export function search(query: SearchQuery): Promise<Page<VideoSearchItem>> {
  return api.page<VideoSearchItem>("/search", {
    auth: false,
    query: {
      q: query.q,
      sort: query.sort,
      category: query.category,
      language: query.language,
      tags: query.tags,
      min_duration: query.minDuration,
      max_duration: query.maxDuration,
      page: query.page,
      limit: query.limit,
    },
  });
}

/** `GET /search/suggest` — up to ten title suggestions, a plain string array. */
export async function suggest(q: string): Promise<string[]> {
  const suggestions = await api.get<string[] | null>("/search/suggest", {
    auth: false,
    query: { q },
  });
  return suggestions ?? [];
}

/** `GET /videos/:id/related` — similar by tags/category, topped up from trending. */
export async function getRelated(videoId: string, limit?: number): Promise<VideoSearchItem[]> {
  const items = await api.get<VideoSearchItem[] | null>(`/videos/${videoId}/related`, {
    auth: false,
    query: { limit },
    revalidate: 120,
  });
  return items ?? [];
}

/** `GET /categories` — distinct categories in use, with video counts. */
export async function getCategories(): Promise<CategoryCount[]> {
  const categories = await api.get<CategoryCount[] | null>("/categories", {
    auth: false,
    revalidate: 300,
    tags: ["categories"],
  });
  return categories ?? [];
}

/** `GET /videos/trending` — most engaged-with public videos inside a window. */
export async function getTrending(window?: TrendingWindow, limit?: number): Promise<VideoSearchItem[]> {
  const items = await api.get<VideoSearchItem[] | null>("/videos/trending", {
    auth: false,
    query: { window, limit },
    revalidate: 60,
  });
  return items ?? [];
}
