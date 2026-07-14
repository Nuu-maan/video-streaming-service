import "server-only";

import { api } from "@/lib/api-client";
import type { Comment, Page, PageParams } from "@/types/common";

/**
 * A video's top-level comments, pinned first. Replies are not included — each
 * comment reports its `reply_count`, and the thread fetches them on demand.
 * Auth is optional: a signed-out visitor reads the same thread.
 */
export async function listComments(videoId: string, params: PageParams = {}): Promise<Page<Comment>> {
  return api.page<Comment>(`/videos/${videoId}/comments`, {
    query: { page: params.page, limit: params.limit ?? 20 },
  });
}

/** A comment's replies, oldest first — a conversation reads forwards. */
export async function listReplies(commentId: string, params: PageParams = {}): Promise<Page<Comment>> {
  return api.page<Comment>(`/comments/${commentId}/replies`, {
    query: { page: params.page, limit: params.limit ?? 20 },
  });
}
