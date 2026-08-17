import "server-only";

import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";
import type { Page, PageParams, WatchLaterItem } from "@/types/common";

/** The caller's saved videos, most recently saved first. */
export async function listWatchLater(params: PageParams = {}): Promise<Page<WatchLaterItem>> {
  return api.page<WatchLaterItem>("/me/watch-later", {
    query: { page: params.page, limit: params.limit ?? 24 },
  });
}

/**
 * Whether a video is already saved.
 *
 * There is no "is this saved" endpoint, so this walks the caller's list. It is
 * bounded deliberately: a watch-later list is a queue, not an archive, and
 * paging forever to decide the initial state of one button is a worse failure
 * than occasionally showing "Save" for a video sitting at position 501. The
 * save itself is idempotent (PUT), so a stale button is harmless.
 */
const SCAN_LIMIT = 100;
const SCAN_MAX_PAGES = 5;

export async function isInWatchLater(videoId: string): Promise<boolean> {
  try {
    for (let page = 1; page <= SCAN_MAX_PAGES; page += 1) {
      const result = await api.page<WatchLaterItem>("/me/watch-later", {
        query: { page, limit: SCAN_LIMIT },
      });
      if (result.items.some((item) => item.video.id === videoId)) return true;
      if (!result.pagination.has_next) return false;
    }
    return false;
  } catch (error) {
    // Signed out is not an error here — it is simply "not saved".
    if (isApiError(error) && error.isUnauthorized) return false;
    throw error;
  }
}
