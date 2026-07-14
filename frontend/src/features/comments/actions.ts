"use server";

import { revalidatePath } from "next/cache";

import { routes } from "@/config/routes";
import { listComments, listReplies } from "@/features/comments/api";
import { commentSchema } from "@/features/comments/schemas";
import type {
  ActionFailure,
  CommentDeleteResult,
  CommentPageResult,
  CommentResult,
} from "@/features/comments/types";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";
import type { Comment } from "@/types/common";

function fail(error: unknown): ActionFailure {
  if (isApiError(error)) {
    if (error.isUnauthorized) {
      return { ok: false, code: "UNAUTHORIZED", message: "Sign in to join the conversation." };
    }
    if (error.isForbidden) {
      return { ok: false, code: "FORBIDDEN", message: "That isn't yours to change." };
    }
    if (error.isRateLimited) {
      return { ok: false, code: "RATE_LIMITED", message: "Slow down a moment, then try again." };
    }
    if (error.isNotFound) {
      return { ok: false, code: "NOT_FOUND", message: "That comment is no longer there." };
    }
    return { ok: false, code: error.code, message: error.message };
  }
  return { ok: false, code: "UNKNOWN", message: "Something went wrong. Please try again." };
}

/**
 * Every action re-validates with the same schema the form used. A Server
 * Function is a public POST endpoint; the 10,000-character bound in the
 * textarea is a courtesy, not a control.
 */
export async function createComment(
  videoId: string,
  content: string,
  parentId?: string,
): Promise<CommentResult> {
  const parsed = commentSchema.safeParse({ content });
  if (!parsed.success) {
    return { ok: false, code: "VALIDATION", message: parsed.error.issues[0].message };
  }

  try {
    const comment = await api.post<Comment>(`/videos/${videoId}/comments`, {
      body: { content: parsed.data.content, parent_id: parentId },
    });
    revalidatePath(routes.video(videoId));
    return { ok: true, comment };
  } catch (error) {
    return fail(error);
  }
}

/** Author only. The API stamps `edited_at`, which the thread then shows. */
export async function updateComment(
  commentId: string,
  content: string,
  videoId: string,
): Promise<CommentResult> {
  const parsed = commentSchema.safeParse({ content });
  if (!parsed.success) {
    return { ok: false, code: "VALIDATION", message: parsed.error.issues[0].message };
  }

  try {
    const comment = await api.patch<Comment>(`/comments/${commentId}`, {
      body: { content: parsed.data.content },
    });
    revalidatePath(routes.video(videoId));
    return { ok: true, comment };
  } catch (error) {
    return fail(error);
  }
}

/** Author, video owner, or moderator. Soft delete server-side. */
export async function deleteComment(commentId: string, videoId: string): Promise<CommentDeleteResult> {
  try {
    await api.delete(`/comments/${commentId}`);
  } catch (error) {
    // Already gone is the outcome we wanted.
    if (!(isApiError(error) && error.isNotFound)) return fail(error);
  }
  revalidatePath(routes.video(videoId));
  return { ok: true };
}

/**
 * Paging through a thread from the client. The list is a client component (it
 * owns optimistic inserts), and a client component has no token — so "load
 * more" comes back through a Server Action rather than a fetch.
 */
export async function fetchComments(videoId: string, page: number): Promise<CommentPageResult> {
  try {
    const result = await listComments(videoId, { page });
    return { ok: true, items: result.items, pagination: result.pagination };
  } catch (error) {
    return fail(error);
  }
}

export async function fetchReplies(commentId: string, page: number): Promise<CommentPageResult> {
  try {
    const result = await listReplies(commentId, { page });
    return { ok: true, items: result.items, pagination: result.pagination };
  } catch (error) {
    return fail(error);
  }
}
