"use server";

import { revalidatePath, revalidateTag } from "next/cache";

import { routes } from "@/config/routes";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";

/**
 * Deletes a video — the file, the renditions, the stats. Irreversible, which
 * is why every caller sits behind a ConfirmDialog.
 *
 * Returns a result instead of throwing: the row's dropdown wants to show a
 * toast, not unwind into the nearest error boundary.
 *
 * BOTH invalidations are required, and for a while only one of them was here.
 * `revalidatePath(routes.studio)` refreshes the creator's own table — which is
 * the page they are standing on, so the row vanishes and everything LOOKS
 * finished. But `features/videos/api.ts` caches `listVideos` (60s), `getTrending`
 * (300s) and `getCategories` (300s) under the `videos` tag, and nothing was
 * busting it: the only `revalidateTag("videos")` in the codebase lived in a
 * duplicate `deleteVideo` in `features/videos/actions.ts` that no component ever
 * imported. Dead code. So a creator could delete a video — possibly one they were
 * deleting because it had to come down — and it would keep being served on the
 * home page, /videos and /trending, clickable, for up to five minutes.
 *
 * The duplicate has since been removed. There is exactly one delete action, and
 * it is this one.
 */
export async function deleteVideo(
  videoId: string,
): Promise<{ ok: true } | { ok: false; message: string }> {
  try {
    await api.delete(`/videos/${videoId}`);
  } catch (error) {
    if (isApiError(error)) {
      if (error.isRateLimited) {
        return { ok: false, message: "Slow down — too many requests. Try again in a moment." };
      }
      if (error.isNotFound) {
        // Already gone; the refresh below will drop the row. Not a failure — and
        // the public lists may still be holding it, so they get busted too.
        revalidateTag("videos", "max");
        revalidatePath(routes.studio);
        return { ok: true };
      }
      return { ok: false, message: error.message };
    }
    return { ok: false, message: "Couldn't reach the server." };
  }

  revalidateTag("videos", "max");
  revalidatePath(routes.studio);
  return { ok: true };
}
