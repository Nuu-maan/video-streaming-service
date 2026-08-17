"use server";

import { api } from "@/lib/api-client";
import type { VideoStatusReport } from "@/types/common";

/*
 * There WAS a `deleteVideo` here. It is gone, and deliberately.
 *
 * It was a second, complete implementation of the delete — with its own error
 * copy and its own cache-invalidation strategy — and nothing in the app imported
 * it. The live delete button reaches for `features/studio/actions.ts#deleteVideo`.
 * Two functions with the same name and different behaviour, one of them dead, is
 * how the cache bug survived review: this one busted the `videos` tag, the real
 * one did not, and reading either in isolation looked correct.
 *
 * One delete action, in `features/studio/actions.ts`. It busts both the studio
 * path and the `videos` tag.
 */

/**
 * Status probe for the client-side ProcessingPoller. Read-only; a failed poll
 * answers null rather than throwing — the poller just tries again on the next
 * tick, and its own timeout is the backstop.
 */
export async function pollVideoStatus(videoId: string): Promise<VideoStatusReport | null> {
  try {
    return await api.get<VideoStatusReport>(`/videos/${videoId}/status`);
  } catch {
    return null;
  }
}
