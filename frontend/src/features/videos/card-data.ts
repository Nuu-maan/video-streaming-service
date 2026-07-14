import { mediaUrl } from "@/lib/api-client";
import type { VideoCardData } from "@/features/videos/types";
import type { Video, VideoSearchItem } from "@/types/common";

/**
 * Normalisers from the API's two video shapes into the one the card renders.
 * Server-only by construction (mediaUrl lives in the server-only api-client):
 * pages map on the server and pass plain `VideoCardData` to components, so the
 * components themselves stay usable inside client boundaries.
 */

/**
 * A public video's thumbnail is fetched straight off the API origin. A
 * private/unlisted one is exactly as private as the video (the API 404s it
 * without a token), and the browser deliberately has no token — so it must go
 * through this origin's `/api/media` proxy, which attaches the bearer
 * server-side. Same rule as playback, decided in the same place: on the server.
 */
function resolveThumbnail(video: Video): string | null {
  if (!video.thumbnail_url) return null;
  if (video.visibility !== "public") {
    return `/api/media/videos/${video.id}/thumbnail`;
  }
  return mediaUrl(video.thumbnail_url);
}

export function toVideoCard(video: Video): VideoCardData {
  return {
    id: video.id,
    title: video.title,
    thumbnailUrl: resolveThumbnail(video),
    duration: video.duration,
    viewCount: video.view_count,
    createdAt: video.created_at,
    channelId: video.user_id,
    status: video.status,
    transcodingProgress: video.transcoding_progress,
  };
}

export function searchItemToVideoCard(item: VideoSearchItem): VideoCardData {
  return {
    id: item.video_id,
    title: item.title,
    thumbnailUrl: mediaUrl(item.thumbnail_url),
    duration: item.duration,
    viewCount: item.views,
    createdAt: item.created_at,
    channelName: item.username,
    channelId: item.user_id,
  };
}
