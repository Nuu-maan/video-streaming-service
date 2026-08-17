import "server-only";

import { api } from "@/lib/api-client";
import type { NoteSort } from "@/features/comments/schemas";
import type { Comment, Page, PageParams } from "@/types/common";

/**
 * A video's top-level notes, pinned first. Replies are not included — each note
 * reports its `reply_count`, and the thread fetches them on demand. Auth is
 * optional: a signed-out visitor reads the same thread.
 *
 * `sort` defaults to timestamp order, which is the order a review reads in.
 * The API's own default is `newest`; this asks explicitly rather than relying on
 * a default that belongs to the API and could reasonably change.
 */
export async function listComments(
  videoId: string,
  params: PageParams & { sort?: NoteSort } = {},
): Promise<Page<Comment>> {
  return api.page<Comment>(`/videos/${videoId}/comments`, {
    query: {
      page: params.page,
      limit: params.limit ?? 20,
      sort: params.sort ?? "timestamp",
    },
  });
}

/** A note's replies, oldest first — a conversation reads forwards. */
export async function listReplies(commentId: string, params: PageParams = {}): Promise<Page<Comment>> {
  return api.page<Comment>(`/comments/${commentId}/replies`, {
    query: { page: params.page, limit: params.limit ?? 20 },
  });
}
