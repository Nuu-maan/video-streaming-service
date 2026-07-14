import type { StudioVideoRow } from "@/features/studio/types";
import { mediaUrl } from "@/lib/api-client";
import type { Video } from "@/types/common";

/**
 * `Video` → the shape the table renders. Server-only by construction:
 * `mediaUrl()` lives in the server-only api-client, so this runs during the
 * server render and hands the row component plain, already-resolved strings.
 */
export function toStudioRow(video: Video): StudioVideoRow {
  return {
    id: video.id,
    title: video.title,
    thumbnailUrl: resolveThumbnail(video),
    status: video.status,
    visibility: video.visibility,
    transcodingProgress: video.transcoding_progress,
    duration: video.duration,
    viewCount: video.view_count,
    likeCount: video.like_count,
    commentCount: video.comment_count,
    createdAt: video.created_at,
  };
}

/**
 * A public video's thumbnail is fetched straight off the API origin. A private
 * or unlisted one is exactly as private as the video itself — the API 404s it
 * without a bearer token, and the browser deliberately has none — so it must
 * come through this origin's `/api/media` proxy, which attaches the token
 * server-side.
 *
 * The studio is the one screen where this matters constantly: it is the only
 * place a creator's *private* videos are listed at all.
 */
function resolveThumbnail(video: Video): string | null {
  if (!video.thumbnail_url) return null;
  if (video.visibility !== "public") {
    return `/api/media/videos/${video.id}/thumbnail`;
  }
  return mediaUrl(video.thumbnail_url);
}
