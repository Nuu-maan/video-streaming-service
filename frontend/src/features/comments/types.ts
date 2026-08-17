import type { Comment, PaginationMeta } from "@/types/common";

/**
 * Who is reading the thread, and what they are allowed to do to it. Derived on
 * the server (the video's owner and the caller's role are both server facts)
 * and handed to the client components as plain data — the full `User` never
 * crosses the boundary, so an email address never ends up in the RSC payload
 * for a comment list.
 */
export interface CommentViewer {
  id: string;
  username: string;
  avatarUrl?: string;
  /**
   * A moderator, an admin, or the video's owner. The API lets any of them delete
   * a comment; only the author may edit one.
   */
  canModerate: boolean;
}

export interface ActionFailure {
  ok: false;
  code: string;
  message: string;
}

export type CommentResult = { ok: true; comment: Comment } | ActionFailure;
export type CommentDeleteResult = { ok: true } | ActionFailure;
export type CommentPageResult =
  | { ok: true; items: Comment[]; pagination: PaginationMeta }
  | ActionFailure;
