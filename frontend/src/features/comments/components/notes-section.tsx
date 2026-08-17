import { ErrorState } from "@/components/common/error-state";
import { getCurrentUser } from "@/features/auth/current-user";
import { listComments } from "@/features/comments/api";
import { NotesPanel } from "@/features/comments/components/notes-panel";
import type { NoteSort } from "@/features/comments/schemas";
import type { CommentViewer } from "@/features/comments/types";
import { isApiError } from "@/lib/api-error";

interface NotesSectionProps {
  videoId: string;
  /** The video's uploader. They may delete any note on their own video. */
  videoOwnerId?: string;
  /**
   * Timeline order by default, which is what makes the rail a review rather
   * than a comment thread. A video with nothing playable falls back to newest —
   * anchors mean little when there is no timeline to anchor to.
   */
  sort?: NoteSort;
}

/**
 * The server half of the rail: resolves who is reading, fetches the first page,
 * and hands both to the client panel. Sits inside a <Suspense> boundary on the
 * review page — the player must paint without waiting on this.
 */
export async function NotesSection({ videoId, videoOwnerId, sort = "timestamp" }: NotesSectionProps) {
  const [user, page] = await Promise.all([
    getCurrentUser(),
    listComments(videoId, { page: 1, limit: 20, sort }).catch((error: unknown) => {
      if (isApiError(error) && error.isRateLimited) return "rate-limited" as const;
      return "failed" as const;
    }),
  ]);

  if (page === "rate-limited") {
    return (
      <ErrorState
        title="Slow down a moment"
        description="You're loading notes faster than we can serve them. Try again shortly."
        className="m-4 min-h-40"
      />
    );
  }

  if (page === "failed") {
    return (
      <ErrorState
        title="Notes didn't load"
        description="Refresh the page to try again."
        className="m-4 min-h-40"
      />
    );
  }

  const viewer: CommentViewer | null = user
    ? {
        id: user.id,
        username: user.username,
        avatarUrl: user.avatar_url,
        /* The API lets the author, the video's owner and any moderator delete a
           note. Editing stays with the author, and the item decides that. */
        canModerate: user.role === "admin" || user.role === "moderator" || user.id === videoOwnerId,
      }
    : null;

  return (
    <NotesPanel
      videoId={videoId}
      initialComments={page.items}
      initialPagination={page.pagination}
      initialSort={sort}
      viewer={viewer}
    />
  );
}
