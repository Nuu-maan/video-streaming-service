"use server";

import { revalidatePath } from "next/cache";

import { routes } from "@/config/routes";
import type { HistoryResult } from "@/features/history/types";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";

function fail(error: unknown): HistoryResult {
  if (isApiError(error)) {
    if (error.isUnauthorized) return { ok: false, message: "Sign in to manage your history." };
    if (error.isRateLimited) return { ok: false, message: "Slow down a moment, then try again." };
    return { ok: false, message: error.message };
  }
  return { ok: false, message: "Something went wrong. Please try again." };
}

/** Remove one video from the history. Already-gone is the state we were aiming for. */
export async function removeFromHistory(videoId: string): Promise<HistoryResult> {
  try {
    await api.delete(`/me/history/${videoId}`);
  } catch (error) {
    if (!(isApiError(error) && error.isNotFound)) return fail(error);
  }
  revalidatePath(routes.history);
  return { ok: true };
}

/**
 * Delete the whole history. Irreversible — the API has no undo — which is why
 * the button that calls this sits behind a ConfirmDialog that names exactly what
 * is destroyed.
 */
export async function clearHistory(): Promise<HistoryResult> {
  try {
    await api.delete("/me/history");
  } catch (error) {
    return fail(error);
  }
  revalidatePath(routes.history);
  return { ok: true };
}
