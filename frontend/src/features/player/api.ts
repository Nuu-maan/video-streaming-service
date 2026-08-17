import "server-only";

import { cache } from "react";

import type { Channel } from "@/features/player/types";
import { api, mediaUrl } from "@/lib/api-client";
import type { User, Video, VideoSearchItem, WatchHistory } from "@/types/common";

/**
 * Server-side reads for the watch page.
 *
 * `getVideo` is wrapped in React's `cache` because the same render pass asks
 * twice — `generateMetadata` and the page itself both need the video — and that
 * must cost one request, not two.
 */
export const getVideo = cache(async (id: string): Promise<Video> => {
  return api.get<Video>(`/videos/${id}`);
});

/**
 * Where the signed-in viewer left off, in seconds — or null when there is
 * nothing worth resuming.
 *
 * The API exposes no per-video progress read, only `GET /me/history`, so this
 * scans the most recent page of history. Best effort by design: a missing entry,
 * an entry buried deeper than one page, or any failure at all all mean "start
 * from the beginning", which is never the wrong answer, only a less helpful one.
 */
export async function getResumePosition(videoId: string): Promise<number | null> {
  try {
    const { items } = await api.page<WatchHistory>("/me/history", { query: { limit: 100 } });
    const entry = items.find((item) => item.video_id === videoId);
    if (!entry || entry.completed) return null;
    // Resuming at 0:03 is noise, not a favour.
    return entry.last_position > 10 ? entry.last_position : null;
  } catch {
    return null;
  }
}

/**
 * The uploader of a video.
 *
 * This is harder than it should be. A `Video` carries `user_id` and no other
 * trace of its author, and the API has no public user-profile endpoint — so
 * there are exactly two honest ways to learn a username:
 *
 *  1. The viewer IS the uploader, in which case we already hold their `User`.
 *  2. The search index, which denormalises `username` and `user_avatar_url`
 *     onto every result. It only indexes public, ready videos — which is
 *     precisely the case where a stranger is looking at a channel row anyway.
 *
 * Anything else answers null, and the channel row degrades to an anonymous
 * uploader rather than inventing a name.
 *
 * TODO(users feature): delete the search hop the moment `GET /users/:id` exists.
 */
export async function getUploader(video: Video, viewer: User | null): Promise<Channel | null> {
  const userId = video.user_id;
  if (!userId) return null;

  const isViewer = Boolean(viewer && viewer.id === userId);

  /*
   * These two reads have nothing to say to each other — the subscriber count
   * needs a user id, the search hop needs a video — so they must not queue. They
   * used to: `await getSubscriberCount(...)` and then, further down,
   * `await findInSearchIndex(...)`, which is one wasted round trip on the watch
   * page for every viewer who is not the uploader (i.e. nearly all of them).
   *
   * The search hop is also skipped outright when the viewer IS the uploader — we
   * already hold their User, so there is nothing to look up. The owner's channel
   * row costs one request; a stranger's costs two, in parallel, instead of two in
   * series.
   */
  const [subscriberCount, indexed] = await Promise.all([
    getSubscriberCount(userId),
    isViewer ? Promise.resolve(null) : findInSearchIndex(video),
  ]);

  if (viewer && isViewer) {
    return {
      id: userId,
      username: viewer.username,
      avatarUrl: viewer.avatar_url ?? viewer.oauth_avatar_url ?? null,
      verified: viewer.email_verified,
      subscriberCount,
      isViewer: true,
    };
  }

  if (!indexed) return null;

  return {
    id: userId,
    username: indexed.username,
    avatarUrl: mediaUrl(indexed.user_avatar_url),
    verified: indexed.user_verified,
    subscriberCount,
    isViewer: false,
  };
}

/**
 * Finds the video's own row in the search index, which is the only public
 * surface that carries its uploader's name.
 *
 * Searching by title can return several videos; the match is made on `video_id`,
 * never on the title, so a same-titled video can never lend its uploader to this
 * one. Cached for five minutes — a username does not change on a watch's
 * timescale, and this must not cost a request per viewer.
 */
async function findInSearchIndex(video: Video): Promise<VideoSearchItem | null> {
  // The index only holds public, ready videos. Anything else is a guaranteed miss.
  if (video.visibility !== "public" || video.status !== "ready") return null;

  const q = video.title.trim().slice(0, 80);
  if (!q) return null;

  try {
    const { items } = await api.page<VideoSearchItem>("/search", {
      auth: false,
      query: { q, limit: 20 },
      revalidate: 300,
    });
    return items.find((item) => item.video_id === video.id) ?? null;
  } catch {
    return null;
  }
}

/**
 * Subscriber count, read off the pagination envelope of the subscriber list —
 * the API exposes no direct count. Best effort: the channel row renders without
 * it rather than not at all.
 */
async function getSubscriberCount(userId: string): Promise<number | null> {
  try {
    const { pagination } = await api.page<unknown>(`/users/${userId}/subscribers`, { query: { limit: 1 } });
    return pagination.total;
  } catch {
    return null;
  }
}
