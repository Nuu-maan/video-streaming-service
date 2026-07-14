"use client";

import { ChevronDown, LoaderCircle, Pin } from "lucide-react";
import { useOptimistic, useState } from "react";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { createComment, deleteComment, fetchReplies, updateComment } from "@/features/comments/actions";
import { CommentForm } from "@/features/comments/components/comment-form";
import { CommentMenu } from "@/features/comments/components/comment-menu";
import type { CommentViewer } from "@/features/comments/types";
import { ReportDialog } from "@/features/reports/components/report-dialog";
import { useSignInPrompt } from "@/hooks/use-sign-in-prompt";
import { formatDate, formatRelativeTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { Comment } from "@/types/common";

interface CommentItemProps {
  comment: Comment;
  viewer: CommentViewer | null;
  videoId: string;
  /** Reports the comment's new state upwards; null means it was deleted. */
  onChange: (next: Comment | null) => void;
  /**
   * Replies are one level deep. A reply renders as a reply, and its reply box
   * posts into the parent's thread — which is where the parent's optimistic
   * list lives, so the parent hands its own submit handler down.
   */
  isReply?: boolean;
  onReply?: (content: string) => Promise<boolean>;
}

/** A stand-in with a temporary id, swapped for the server's row when it lands. */
export function draftComment(
  videoId: string,
  viewer: CommentViewer,
  content: string,
  parentId?: string,
): Comment {
  const now = new Date().toISOString();
  return {
    id: `optimistic-${now}-${Math.random().toString(36).slice(2)}`,
    video_id: videoId,
    user_id: viewer.id,
    parent_id: parentId,
    content,
    like_count: 0,
    reply_count: 0,
    pinned: false,
    created_at: now,
    updated_at: now,
    username: viewer.username,
    avatar_url: viewer.avatarUrl,
  };
}

export function isDraft(comment: Comment): boolean {
  return comment.id.startsWith("optimistic-");
}

export function CommentItem({
  comment,
  viewer,
  videoId,
  onChange,
  isReply = false,
  onReply,
}: CommentItemProps) {
  const [editing, setEditing] = useState(false);
  const [replying, setReplying] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [reportOpen, setReportOpen] = useState(false);
  const promptSignIn = useSignInPrompt();

  // Replies load on demand, never with the thread: a page of comments that
  // eagerly pulled every reply would be a page of a hundred requests.
  const [replies, setReplies] = useState<Comment[]>([]);
  const [optimisticReplies, addOptimisticReply] = useOptimistic(
    replies,
    (current: Comment[], reply: Comment) => [...current, reply],
  );
  const [repliesOpen, setRepliesOpen] = useState(false);
  const [repliesLoaded, setRepliesLoaded] = useState(false);
  const [loadingReplies, setLoadingReplies] = useState(false);
  const [replyCount, setReplyCount] = useState(comment.reply_count);

  const isAuthor = Boolean(viewer && viewer.id === comment.user_id);
  const isDeleted = Boolean(comment.deleted_at);
  const author = comment.username ?? "Someone";
  const initial = author.slice(0, 1).toUpperCase();

  async function loadReplies() {
    setLoadingReplies(true);
    const result = await fetchReplies(comment.id, 1);
    setLoadingReplies(false);

    if (!result.ok) {
      toast.error(result.message);
      return;
    }
    setReplies(result.items);
    setReplyCount(result.pagination.total);
    setRepliesLoaded(true);
  }

  function toggleReplies() {
    const next = !repliesOpen;
    setRepliesOpen(next);
    if (next && !repliesLoaded && !loadingReplies) void loadReplies();
  }

  async function handleEdit(content: string): Promise<boolean> {
    const result = await updateComment(comment.id, content, videoId);
    if (!result.ok) {
      toast.error(result.message);
      return false;
    }
    onChange(result.comment);
    setEditing(false);
    return true;
  }

  async function handleDelete() {
    const result = await deleteComment(comment.id, videoId);
    if (!result.ok) {
      toast.error(result.message);
      return;
    }
    onChange(null);
  }

  /** Posting a reply to *this* comment. Only depth-0 items own this. */
  async function submitReply(content: string): Promise<boolean> {
    if (!viewer) return false;

    addOptimisticReply(draftComment(videoId, viewer, content, comment.id));

    const result = await createComment(videoId, content, comment.id);
    if (!result.ok) {
      toast.error(result.message);
      return false;
    }

    setReplies((current) => [...current, result.comment]);
    setReplyCount((current) => current + 1);
    setRepliesLoaded(true);
    setReplying(false);
    return true;
  }

  const handleReply = isReply ? onReply : submitReply;

  function startReply() {
    if (!viewer) {
      promptSignIn("Sign in to reply.");
      return;
    }
    // Open the thread first, outside any transition, so the optimistic reply has
    // somewhere visible to land the moment it is submitted.
    if (!isReply) {
      setRepliesOpen(true);
      if (!repliesLoaded && !loadingReplies && comment.reply_count > 0) void loadReplies();
    }
    setReplying(true);
  }

  function updateReply(id: string, next: Comment | null) {
    setReplies((current) =>
      next
        ? current.map((reply) => (reply.id === id ? next : reply))
        : current.filter((reply) => reply.id !== id),
    );
    if (!next) setReplyCount((current) => Math.max(0, current - 1));
  }

  return (
    <article className={cn("group/comment flex gap-3", isDraft(comment) && "opacity-60")}>
      <Avatar className={cn("mt-0.5 shrink-0", isReply ? "size-7" : "size-9")}>
        {comment.avatar_url ? <AvatarImage src={comment.avatar_url} alt="" /> : null}
        <AvatarFallback className="bg-brand-800 text-xs font-medium text-brand-100">
          {initial}
        </AvatarFallback>
      </Avatar>

      <div className="min-w-0 flex-1">
        <div className="flex items-start gap-2">
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
              {comment.pinned ? (
                <span
                  title="Pinned by the creator"
                  className="inline-flex items-center gap-1 rounded-full bg-muted px-1.5 py-0.5 text-[0.6875rem] font-medium text-muted-foreground"
                >
                  <Pin aria-hidden className="size-3" />
                  Pinned
                </span>
              ) : null}
              <span className="truncate text-sm font-medium">{author}</span>
              <time
                dateTime={comment.created_at}
                title={formatDate(comment.created_at)}
                suppressHydrationWarning
                className="text-xs text-muted-foreground"
              >
                {formatRelativeTime(comment.created_at)}
              </time>
              {comment.edited_at ? (
                <span className="text-xs text-muted-foreground" title="Edited after posting">
                  (edited)
                </span>
              ) : null}
            </div>

            {editing && viewer ? (
              <CommentForm
                viewer={viewer}
                initialValue={comment.content}
                submitLabel="Save"
                onSubmit={handleEdit}
                onCancel={() => setEditing(false)}
                autoFocus
                compact
                className="mt-2"
              />
            ) : (
              <p
                className={cn(
                  "mt-1 text-sm leading-relaxed break-words whitespace-pre-wrap",
                  isDeleted && "text-muted-foreground italic",
                )}
              >
                {isDeleted ? "This comment was removed." : comment.content}
              </p>
            )}
          </div>

          {/* The permissions are computed HERE, so the items are chosen here.
              The menu renders nothing when it is handed nothing. */}
          {!editing && !isDeleted && !isDraft(comment) ? (
            <CommentMenu>
              {isAuthor ? <CommentMenu.Edit onSelect={() => setEditing(true)} /> : null}
              {isAuthor || viewer?.canModerate ? (
                <CommentMenu.Delete onSelect={() => setConfirmOpen(true)} />
              ) : null}
              {viewer && !isAuthor ? (
                <CommentMenu.Report onSelect={() => setReportOpen(true)} />
              ) : null}
            </CommentMenu>
          ) : null}
        </div>

        {/* Outside the menu, deliberately: a Radix dialog nested inside a menu
            item unmounts with the menu the instant it opens. */}
        <ConfirmDialog
          open={confirmOpen}
          onOpenChange={setConfirmOpen}
          title="Delete this comment?"
          description="It will be removed from the thread. This cannot be undone."
          confirmLabel="Delete"
          destructive
          onConfirm={handleDelete}
        />

        <ReportDialog
          target={{ kind: "comment", id: comment.id }}
          isAuthenticated={Boolean(viewer)}
          open={reportOpen}
          onOpenChange={setReportOpen}
        />

        {!editing && !isDeleted ? (
          <div className="mt-1.5 flex items-center gap-1">
            {handleReply ? (
              <Button
                variant="ghost"
                size="sm"
                onClick={startReply}
                className="h-7 rounded-full px-2.5 text-xs text-muted-foreground"
              >
                Reply
              </Button>
            ) : null}

            {!isReply && replyCount > 0 ? (
              <Button
                variant="ghost"
                size="sm"
                aria-expanded={repliesOpen}
                onClick={toggleReplies}
                /* Hover must GAIN contrast in both themes: darker in light,
                   lighter in dark. The old brand-300 hover took this control to
                   1.9:1 on a light card — hovering made it less legible. */
                className="h-7 rounded-full px-2.5 text-xs font-medium text-brand-700 hover:text-brand-800 dark:text-brand-400 dark:hover:text-brand-300"
              >
                {loadingReplies ? (
                  <LoaderCircle aria-hidden className="size-3.5 animate-spin" />
                ) : (
                  <ChevronDown
                    aria-hidden
                    className={cn(
                      "size-3.5 transition-transform duration-(--motion-fast) ease-out-quart",
                      repliesOpen && "rotate-180",
                    )}
                  />
                )}
                <span className="tabular-nums">
                  {replyCount} {replyCount === 1 ? "reply" : "replies"}
                </span>
              </Button>
            ) : null}
          </div>
        ) : null}

        {replying && viewer && handleReply ? (
          <CommentForm
            viewer={viewer}
            placeholder={`Reply to ${author}…`}
            submitLabel="Reply"
            initialValue={isReply ? `@${author} ` : ""}
            onSubmit={async (content) => {
              const posted = await handleReply(content);
              if (posted) setReplying(false);
              return posted;
            }}
            onCancel={() => setReplying(false)}
            autoFocus
            compact
            className="mt-2"
          />
        ) : null}

        {repliesOpen && !isReply ? (
          <div
            className={cn(
              "mt-3 flex flex-col gap-4 border-l border-border/60 pl-4",
              optimisticReplies.length === 0 && "hidden",
            )}
          >
            {optimisticReplies.map((reply) => (
              <CommentItem
                key={reply.id}
                comment={reply}
                viewer={viewer}
                videoId={videoId}
                onChange={(next) => updateReply(reply.id, next)}
                onReply={submitReply}
                isReply
              />
            ))}
          </div>
        ) : null}
      </div>
    </article>
  );
}
