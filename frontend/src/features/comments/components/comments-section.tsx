import { ErrorState } from "@/components/common/error-state";
import { getCurrentUser } from "@/features/auth/current-user";
import { listComments } from "@/features/comments/api";
import { CommentList } from "@/features/comments/components/comment-list";
import type { CommentViewer } from "@/features/comments/types";
import { isApiError } from "@/lib/api-error";

interface CommentsSectionProps {
  videoId: string;
  /** The video's uploader. They may delete any comment on their own video. */
  videoOwnerId?: string;
}

/**
 * The server half of the thread: it resolves who is reading, fetches the first
 * page, and hands both to the client list. Drop it into a watch page inside a
 * <Suspense> boundary — the video should paint without waiting on comments.
 */
export async function CommentsSection({ videoId, videoOwnerId }: CommentsSectionProps) {
  const [user, page] = await Promise.all([
    getCurrentUser(),
    listComments(videoId, { page: 1, limit: 20 }).catch((error: unknown) => {
      if (isApiError(error) && error.isRateLimited) return "rate-limited" as const;
      return "failed" as const;
    }),
  ]);

  if (page === "rate-limited") {
    return (
      <ErrorState
        title="Slow down a moment"
        description="You're loading comments faster than we can serve them. Try again shortly."
        className="min-h-40"
      />
    );
  }

  if (page === "failed") {
    return (
      <ErrorState
        title="Comments didn't load"
        description="Refresh the page to try again."
        className="min-h-40"
      />
    );
  }

  const viewer: CommentViewer | null = user
    ? {
        id: user.id,
        username: user.username,
        avatarUrl: user.avatar_url,
        /* The API lets the author, the video's owner and any moderator delete a
           comment. Editing stays with the author, and the item decides that. */
        canModerate:
          user.role === "admin" || user.role === "moderator" || user.id === videoOwnerId,
      }
    : null;

  return (
    <CommentList
      videoId={videoId}
      initialComments={page.items}
      initialPagination={page.pagination}
      viewer={viewer}
    />
  );
}
