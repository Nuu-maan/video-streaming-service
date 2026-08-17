"use server";

import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";
import type { VideoStatusReport } from "@/types/common";

/**
 * One poll of `GET /videos/:id/status`, callable from the client. The upload
 * stages component polls this after the transfer finishes, because a client
 * component cannot ask the Go API directly — the bearer token lives in an
 * httpOnly cookie that only the server can read.
 *
 * Returns a plain result instead of throwing: a transient network blip during
 * polling should surface as "still working, retrying", not as an error page.
 */
export async function pollVideoStatus(
  videoId: string,
): Promise<{ ok: true; report: VideoStatusReport } | { ok: false; message: string }> {
  try {
    const report = await api.get<VideoStatusReport>(`/videos/${videoId}/status`);
    return { ok: true, report };
  } catch (error) {
    if (isApiError(error)) {
      if (error.isRateLimited) {
        return { ok: false, message: "Checking too often — slowing down." };
      }
      return { ok: false, message: error.message };
    }
    return { ok: false, message: "Couldn't reach the server." };
  }
}
