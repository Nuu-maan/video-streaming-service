import "server-only";

import type { HistoryRow } from "@/features/history/types";
import { getVideo } from "@/features/videos/api";
import { toVideoCard } from "@/features/videos/card-data";
import { api } from "@/lib/api-client";
import type { Page, PageParams, WatchHistory } from "@/types/common";

/**
 * The page size is small on purpose. See below — every row costs a request.
 */
export const HISTORY_PAGE_SIZE = 12;

/**
 * The caller's watch history, most recently watched first.
 *
 * `GET /me/history` returns entries carrying only a `video_id`: no title, no
 * thumbnail, no duration. There is no bulk video-by-ids endpoint either (the
 * only list endpoint, `GET /videos`, cannot filter by id), so hydrating a page
 * of history genuinely costs one request per row. Two consequences, both
 * deliberate:
 *
 *  - The page size is 12, not the 24 the other listings use. The `user_api`
 *    budget is 60 requests a minute; a 24-row page would spend nearly half of it
 *    on a single page view, and a viewer who paged twice would be rate-limited
 *    for reading their own history.
 *  - The lookups are issued in parallel, so the wall-clock cost is one round
 *    trip rather than twelve sequential ones.
 *
 * An entry whose video no longer resolves is DROPPED, not rendered as a broken
 * row. That covers a video deleted since it was watched, and one whose owner
 * made it private (which 404s for everyone else). In both cases the video is
 * genuinely gone as far as this viewer is concerned, and there is nothing honest
 * to put in the row — a title we no longer have, or a thumbnail that 404s.
 *
 * A dropped row means a page can come back with fewer than `limit` items while
 * `pagination.total` still counts it. That is correct: the entry exists, it just
 * has nothing to show. Renumbering the pagination to hide it would mean lying
 * about how many pages there are.
 */
export async function listHistory(params: PageParams = {}): Promise<Page<HistoryRow>> {
  const history = await api.page<WatchHistory>("/me/history", {
    query: { page: params.page, limit: params.limit ?? HISTORY_PAGE_SIZE },
  });

  const rows = await Promise.all(
    history.items.map(async (entry): Promise<HistoryRow | null> => {
      const video = await getVideo(entry.video_id).catch(() => null);
      if (!video) return null;

      return {
        entryId: entry.id,
        video: toVideoCard(video),
        watchedAt: entry.watched_at,
        progressPercent: toPercent(entry.last_position, video.duration, entry.completed),
        completed: entry.completed,
      };
    }),
  );

  return {
    items: rows.filter((row): row is HistoryRow => row !== null),
    pagination: history.pagination,
  };
}

/**
 * Clamped both ends. `last_position` can exceed `duration` by a hair on a video
 * whose duration was re-measured after transcoding, and a 103%-wide progress bar
 * overflows its track.
 */
function toPercent(position: number, duration: number, completed: boolean): number {
  if (completed) return 100;
  if (!Number.isFinite(duration) || duration <= 0) return 0;
  return Math.min(100, Math.max(0, Math.round((position / duration) * 100)));
}
