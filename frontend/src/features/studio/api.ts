import "server-only";

import { limits } from "@/config/site";
import { api } from "@/lib/api-client";
import type { Page, TimeSeriesData, Video, VideoAnalytics } from "@/types/common";

/**
 * The creator's own videos, every visibility and status included —
 * `mine=true` is what flips the API from "what the public sees" to "what I
 * own". Requires a token; the studio layout has already guaranteed one.
 */
export async function getMyVideos(params: { page?: number } = {}): Promise<Page<Video>> {
  return api.page<Video>("/videos", {
    query: {
      mine: "true",
      page: params.page,
      limit: limits.pageSize,
    },
  });
}

/**
 * A creator's own numbers.
 *
 * `/videos/:id/analytics`, NOT `/admin/analytics/videos/:id`. The two return the
 * identical payload, but the admin route is gated on `view_analytics` — a
 * permission only `premium` and `admin` hold — so for most of the year an
 * uploader could not see how their own video was doing. This route resolves
 * ownership instead: yours, or you hold the permission, or 404.
 *
 * Which means a 404 here is ambiguous by design and must be reported as
 * "not found", never as "not allowed".
 */
export async function getMyVideoAnalytics(videoId: string): Promise<VideoAnalytics> {
  return api.get<VideoAnalytics>(`/videos/${videoId}/analytics`);
}

/** Views over time for a video you own. Same authorization rule as above. */
export async function getMyViewsTimeSeries(
  videoId: string,
  interval: "hour" | "day" | "week" | "month" = "day",
): Promise<TimeSeriesData> {
  return api.get<TimeSeriesData>(`/videos/${videoId}/analytics/views`, {
    query: { interval },
  });
}
