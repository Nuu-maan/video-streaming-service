"use server";

import { revalidatePath } from "next/cache";

import { routes } from "@/config/routes";
import type { LikeActionResult } from "@/features/likes/types";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";
import type { Like } from "@/types/common";

function fail(error: unknown): { ok: false; code: string; message: string } {
  if (isApiError(error)) {
    if (error.isRateLimited) {
      return { ok: false, code: "RATE_LIMITED", message: "You're rating a bit fast — give it a moment and try again." };
    }
    if (error.isUnauthorized) {
      return { ok: false, code: "UNAUTHORIZED", message: "Sign in to rate videos." };
    }
    return { ok: false, code: error.code, message: error.message };
  }
  return { ok: false, code: "UNKNOWN", message: "Something went wrong. Please try again." };
}

/** PUT models like AND dislike — one row per user, so this flips an existing rating. */
export async function rateVideo(videoId: string, isLike: boolean): Promise<LikeActionResult> {
  try {
    await api.put<Like>(`/videos/${videoId}/like`, { body: { is_like: isLike } });
  } catch (error) {
    return fail(error);
  }
  revalidatePath(routes.video(videoId));
  return { ok: true };
}

export async function clearRating(videoId: string): Promise<LikeActionResult> {
  try {
    await api.delete<unknown>(`/videos/${videoId}/like`);
  } catch (error) {
    // "Not rated" is the state we were aiming for anyway.
    if (!(isApiError(error) && error.isNotFound)) return fail(error);
  }
  revalidatePath(routes.video(videoId));
  return { ok: true };
}
