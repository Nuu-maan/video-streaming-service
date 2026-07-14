import "server-only";

import { limits } from "@/config/site";
import { api } from "@/lib/api-client";
import type { Page, Video } from "@/types/common";

/**
 * The creator's own videos, every visibility and status included —
 * `mine=true` is what flips the API from "what the public sees" to "what I
 * own". Requires a token; the studio layout has already guaranteed one.
 */
export async function getMyVideos(params: { page?: number } = {}): Promise<Page<Video>> {
  return api.page<Video>("/videos", {
    query: {
      mine: "true",
      page: params.page,
      limit: limits.pageSize,
    },
  });
}
