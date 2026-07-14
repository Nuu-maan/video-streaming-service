"use client";

import { LoaderCircle, MessageSquare } from "lucide-react";
import Link from "next/link";
import { useOptimistic, useState, useTransition } from "react";
import { toast } from "sonner";

import { EmptyState } from "@/components/common/empty-state";
import { Button } from "@/components/ui/button";
import { createComment, fetchComments } from "@/features/comments/actions";
import { CommentForm } from "@/features/comments/components/comment-form";
import { CommentItem, draftComment } from "@/features/comments/components/comment-item";
import type { CommentViewer } from "@/features/comments/types";
import { routes } from "@/config/routes";
import { formatCompact } from "@/lib/format";
import type { Comment, PaginationMeta } from "@/types/common";

interface CommentListProps {
  videoId: string;
  /** The first page, fetched on the server. */
  initialComments: Comment[];
  initialPagination: PaginationMeta;
  viewer: CommentViewer | null;
}

/**
 * The thread. It owns the list because it owns the optimistic insert: a new
 * comment appears at the top the moment it is submitted, greyed until the
 * server confirms it, and is replaced in place by the real row (with the real
 * id) when it lands. On failure the optimistic entry simply evaporates when the
 * transition ends and a toast explains why.
 *
 * "Load more" comes back through a Server Action rather than a fetch: this is a
 * client component, and a client component has no bearer token by design.
 */
export function CommentList({
  videoId,
  initialComments,
  initialPagination,
  viewer,
}: CommentListProps) {
  const [comments, setComments] = useState(initialComments);
  const [optimisticComments, addOptimisticComment] = useOptimistic(
    comments,
    (current: Comment[], comment: Comment) => [comment, ...current],
  );
  const [pagination, setPagination] = useState(initialPagination);
  const [loadingMore, startLoadMore] = useTransition();
  const [total, setTotal] = useState(initialPagination.total);

  async function handleCreate(content: string): Promise<boolean> {
    if (!viewer) return false;

    addOptimisticComment(draftComment(videoId, viewer, content));

    const result = await createComment(videoId, content);
    if (!result.ok) {
      toast.error(result.message);
      return false;
    }

    setComments((current) => [result.comment, ...current]);
    setTotal((current) => current + 1);
    return true;
  }

  function loadMore() {
    startLoadMore(async () => {
      const result = await fetchComments(videoId, pagination.page + 1);
      if (!result.ok) {
        toast.error(result.message);
        return;
      }
      /* Guard against the duplicate that a comment posted between page 1 and
         page 2 would otherwise cause: the window shifts, and a row we already
         hold reappears on the next page. */
      setComments((current) => {
        const seen = new Set(current.map((comment) => comment.id));
        return [...current, ...result.items.filter((comment) => !seen.has(comment.id))];
      });
      setPagination(result.pagination);
      setTotal(result.pagination.total);
    });
  }

  function updateComment(id: string, next: Comment | null) {
    setComments((current) =>
      next
        ? current.map((comment) => (comment.id === id ? next : comment))
        : current.filter((comment) => comment.id !== id),
    );
    if (!next) setTotal((current) => Math.max(0, current - 1));
  }

  return (
    <section aria-labelledby="comments-heading" className="flex flex-col gap-6">
      <h2 id="comments-heading" className="text-heading">
        <span className="tabular-nums">{formatCompact(total)}</span>{" "}
        {total === 1 ? "comment" : "comments"}
      </h2>

      {viewer ? (
        <CommentForm viewer={viewer} onSubmit={handleCreate} />
      ) : (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-muted/50 px-4 py-3 ring-1 ring-border/60 ring-inset">
          <p className="text-sm text-muted-foreground">Sign in to join the conversation.</p>
          <Button asChild size="sm" variant="secondary">
            <Link href={routes.login}>Sign in</Link>
          </Button>
        </div>
      )}

      {optimisticComments.length === 0 ? (
        <EmptyState
          icon={MessageSquare}
          title="No comments yet"
          description="Be the first to say something about this video."
          className="min-h-40"
        />
      ) : (
        <div className="flex flex-col gap-6">
          {optimisticComments.map((comment) => (
            <CommentItem
              key={comment.id}
              comment={comment}
              viewer={viewer}
              videoId={videoId}
              onChange={(next) => updateComment(comment.id, next)}
            />
          ))}
        </div>
      )}

      {pagination.has_next ? (
        <div className="flex justify-center">
          <Button variant="outline" size="sm" disabled={loadingMore} onClick={loadMore}>
            {loadingMore ? <LoaderCircle aria-hidden className="animate-spin" /> : null}
            Show more comments
          </Button>
        </div>
      ) : null}
    </section>
  );
}
