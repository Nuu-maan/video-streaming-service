"use server";

import { revalidatePath } from "next/cache";

import { routes } from "@/config/routes";
import type { ActionFailure, WatchLaterResult } from "@/features/watch-later/types";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";

function fail(error: unknown): ActionFailure {
  if (isApiError(error)) {
    if (error.isUnauthorized) return { ok: false, code: "UNAUTHORIZED", message: "Sign in to save videos." };
    if (error.isRateLimited) {
      return { ok: false, code: "RATE_LIMITED", message: "Slow down a moment, then try again." };
    }
    // A private video 404s rather than 403s, so "not found" is all we can honestly say.
    if (error.isNotFound) return { ok: false, code: "NOT_FOUND", message: "Video not found." };
    return { ok: false, code: error.code, message: error.message };
  }
  return { ok: false, code: "UNKNOWN", message: "Something went wrong. Please try again." };
}

/** PUT is idempotent server-side: saving an already-saved video is a no-op, not a 409. */
export async function addToWatchLater(videoId: string): Promise<WatchLaterResult> {
  try {
    await api.put(`/videos/${videoId}/watch-later`);
  } catch (error) {
    return fail(error);
  }
  revalidatePath(routes.watchLater);
  return { ok: true, saved: true };
}

export async function removeFromWatchLater(videoId: string): Promise<WatchLaterResult> {
  try {
    await api.delete(`/videos/${videoId}/watch-later`);
  } catch (error) {
    // Already gone is the state we were aiming for.
    if (!(isApiError(error) && error.isNotFound)) return fail(error);
  }
  revalidatePath(routes.watchLater);
  return { ok: true, saved: false };
}

export async function toggleWatchLater(videoId: string, saved: boolean): Promise<WatchLaterResult> {
  return saved ? removeFromWatchLater(videoId) : addToWatchLater(videoId);
}
