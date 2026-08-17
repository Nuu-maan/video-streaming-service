import "server-only";

import { api } from "@/lib/api-client";
import type { QueueStats, RealtimeMetrics, WorkerInfo, WorkerList } from "@/features/admin/types";
import type {
  ContentReport,
  DashboardStats,
  Page,
  PageParams,
  Video,
  VideoAnalytics,
} from "@/types/common";

/**
 * The read layer for the admin surface.
 *
 * Nothing here is cached, and that is deliberate. Every one of these endpoints
 * reports the *current* state of the platform — a queue depth, a live viewer
 * count, the reports nobody has actioned yet — and a moderator acting on a
 * sixty-second-old queue is a moderator reviewing a report that a colleague
 * already resolved. `api` defaults to `no-store`; none of these opt out.
 *
 * They are also all authenticated, and the API gates them on permissions the
 * frontend cannot see (`view_analytics`, `moderate_content`, `manage_users`).
 * A caller who lacks one gets a 403, which the pages surface as its own state
 * rather than a generic failure.
 */

/** Platform-wide totals. Requires `view_analytics`. */
export function getDashboard(): Promise<DashboardStats> {
  return api.get<DashboardStats>("/admin/analytics/dashboard");
}

/** Live counters — active viewers, jobs in flight, CPU. Never cached, by definition. */
export function getRealtime(): Promise<RealtimeMetrics> {
  return api.get<RealtimeMetrics>("/admin/analytics/realtime");
}

/**
 * The most-viewed videos of the past week, as a plain array — this endpoint is
 * not paginated.
 *
 * `limit` is the one place in the API where a malformed value is a 400 rather
 * than being silently defaulted, so it is clamped to the documented 1–50 window
 * here instead of being passed through and hoping.
 */
export async function getTopVideos(limit = 10): Promise<VideoAnalytics[]> {
  const bounded = Math.min(Math.max(Math.trunc(limit) || 10, 1), 50);
  const videos = await api.get<VideoAnalytics[] | null>("/admin/analytics/top-videos", {
    query: { limit: bounded },
  });
  // Like every list on this API, an empty result is `null`, not `[]`.
  return videos ?? [];
}

/** The engagement breakdown for one video. Requires `view_analytics`. */
export function getVideoAnalytics(id: string): Promise<VideoAnalytics> {
  return api.get<VideoAnalytics>(`/admin/analytics/videos/${id}`);
}

/** The moderation queue: reports nobody has actioned. Requires `moderate_content`. */
export function getPendingReports({ page, limit }: PageParams = {}): Promise<Page<ContentReport>> {
  return api.page<ContentReport>("/admin/reports/pending", { query: { page, limit } });
}

/**
 * Videos whose transcode failed, so the queue page can offer a one-click retry
 * instead of asking an admin to paste a UUID.
 *
 * This is the ordinary `/videos` listing with `status=failed`, not an admin
 * endpoint — which is exactly its limitation. `/videos` without `mine=true`
 * shows public videos, so a *private* video that failed to transcode will not
 * appear here. That is why the queue page keeps a retry-by-ID form alongside
 * this list rather than treating it as the complete picture.
 */
export function getFailedVideos({ page, limit = 20 }: PageParams = {}): Promise<Page<Video>> {
  return api.page<Video>("/videos", { query: { status: "failed", page, limit } });
}

/** Asynq's default-queue depth. Requires `moderate_content`. */
export function getQueueStats(): Promise<QueueStats> {
  return api.get<QueueStats>("/admin/queue/stats");
}

/**
 * The transcoding workers currently checked in.
 *
 * Note the path: `/admin/workers`, not `/admin/queue/workers`. The queue stats
 * and the worker list are siblings in the UI but not in the API.
 */
export async function getWorkers(): Promise<WorkerList> {
  const result = await api.get<{ workers: WorkerInfo[] | null; count?: number } | null>("/admin/workers");
  const workers = result?.workers ?? [];
  return { workers, count: result?.count ?? workers.length };
}
