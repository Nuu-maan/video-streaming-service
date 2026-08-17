import "server-only";

import type { Rating } from "@/features/likes/types";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";
import type { Like } from "@/types/common";

/**
 * The caller's rating on a video, or null when they haven't rated it (404) or
 * aren't signed in (401). Both non-answers mean the same thing to the UI — no
 * active thumb — so neither is worth an exception.
 */
export async function getMyRating(videoId: string): Promise<Rating | null> {
  try {
    const like = await api.get<Like>(`/videos/${videoId}/like`);
    return like.is_like ? "like" : "dislike";
  } catch (error) {
    if (isApiError(error) && (error.isNotFound || error.isUnauthorized)) return null;
    throw error;
  }
}
