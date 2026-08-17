"use client";

import { LoaderCircle, MessageSquareText } from "lucide-react";
import Link from "next/link";
import { useEffect, useOptimistic, useState, useTransition } from "react";
import { toast } from "sonner";

import { EmptyState } from "@/components/common/empty-state";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { createComment, fetchComments } from "@/features/comments/actions";
import { CommentForm } from "@/features/comments/components/comment-form";
import { CommentItem, draftComment } from "@/features/comments/components/comment-item";
import type { NoteSort } from "@/features/comments/schemas";
import type { CommentViewer } from "@/features/comments/types";
import { useReview } from "@/features/review/review-context";
import { formatCompact } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { Comment, PaginationMeta } from "@/types/common";

interface NotesPanelProps {
  videoId: string;
  /** The first page, fetched on the server in the initial sort. */
  initialComments: Comment[];
  initialPagination: PaginationMeta;
  initialSort: NoteSort;
  viewer: CommentViewer | null;
}

/** The anchors, ascending and de-duplicated — what the scrub bar draws. */
function anchorsOf(notes: Comment[]): number[] {
  const seconds = new Set<number>();
  for (const note of notes) {
    if (typeof note.video_timestamp === "number") seconds.add(note.video_timestamp);
  }
  return [...seconds].sort((a, b) => a - b);
}

/**
 * Where an optimistic note belongs.
 *
 * In timestamp order a new note goes where its anchor puts it, not at the top:
 * inserting at the top and letting the server's next page correct it would make
 * the note visibly jump moments after it was written, which reads as a bug. An
 * unanchored note sorts after every anchored one, matching the API's NULLS LAST.
 */
function insertNote(notes: Comment[], note: Comment, sort: NoteSort): Comment[] {
  if (sort === "newest" || typeof note.video_timestamp !== "number") {
    return sort === "newest" ? [note, ...notes] : [...notes, note];
  }

  const at = notes.findIndex(
    (existing) =>
      typeof existing.video_timestamp !== "number" || existing.video_timestamp > note.video_timestamp!,
  );
  if (at === -1) return [...notes, note];
  return [...notes.slice(0, at), note, ...notes.slice(at)];
}

/**
 * The review rail: every note on a video, anchored to the moment it is about.
 *
 * It is a three-row grid — header, scrolling list, composer — rather than a
 * single scrolling column, so the composer stays reachable at the bottom of a
 * hundred-note thread and the sort control stays reachable at the top of one.
 * Only the middle row scrolls.
 *
 * The player is a sibling, reached through the review context. Where there is no
 * context (a video still transcoding has notes but nothing to play) every
 * timestamp affordance simply disappears: no seek, no anchor button, and the
 * chips render as plain text.
 */
export function NotesPanel({
  videoId,
  initialComments,
  initialPagination,
  initialSort,
  viewer,
}: NotesPanelProps) {
  const review = useReview();

  const [sort, setSort] = useState<NoteSort>(initialSort);
  const [notes, setNotes] = useState(initialComments);
  const [optimisticNotes, addOptimisticNote] = useOptimistic(
    notes,
    (current: Comment[], note: Comment) => insertNote(current, note, sort),
  );
  const [pagination, setPagination] = useState(initialPagination);
  const [total, setTotal] = useState(initialPagination.total);
  const [busy, startTransition] = useTransition();

  /**
   * Publish the anchors upward so the scrub bar can draw them.
   *
   * Keyed on the notes the panel is actually holding, so adding, editing or
   * deleting a note moves its tick immediately. The provider compares before
   * storing, so a change that touches no anchor costs nothing.
   */
  const publishMarkers = review?.publishMarkers;
  useEffect(() => {
    publishMarkers?.(anchorsOf(optimisticNotes));
  }, [publishMarkers, optimisticNotes]);

  async function handleCreate(content: string, videoTimestamp?: number): Promise<boolean> {
    if (!viewer) return false;

    addOptimisticNote(draftComment(videoId, viewer, content, undefined, videoTimestamp));

    const result = await createComment(videoId, content, undefined, videoTimestamp);
    if (!result.ok) {
      toast.error(result.message);
      return false;
    }

    setNotes((current) => insertNote(current, result.comment, sort));
    setTotal((current) => current + 1);
    return true;
  }

  function changeSort(next: NoteSort) {
    if (next === sort || busy) return;
    startTransition(async () => {
      const result = await fetchComments(videoId, 1, next);
      if (!result.ok) {
        toast.error(result.message);
        return;
      }
      setSort(next);
      setNotes(result.items);
      setPagination(result.pagination);
      setTotal(result.pagination.total);
    });
  }

  function loadMore() {
    startTransition(async () => {
      const result = await fetchComments(videoId, pagination.page + 1, sort);
      if (!result.ok) {
        toast.error(result.message);
        return;
      }
      /* Guard against the duplicate that a note posted between page 1 and page 2
         would otherwise cause: the window shifts, and a row we already hold
         reappears on the next page. */
      setNotes((current) => {
        const seen = new Set(current.map((note) => note.id));
        return [...current, ...result.items.filter((note) => !seen.has(note.id))];
      });
      setPagination(result.pagination);
      setTotal(result.pagination.total);
    });
  }

  function updateNote(id: string, next: Comment | null) {
    setNotes((current) =>
      next ? current.map((note) => (note.id === id ? next : note)) : current.filter((note) => note.id !== id),
    );
    if (!next) setTotal((current) => Math.max(0, current - 1));
  }

  return (
    <section
      aria-labelledby="notes-heading"
      className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)_auto] xl:h-full"
    >
      <div className="flex items-center justify-between gap-3 border-b border-border/60 px-4 py-3">
        <h2 id="notes-heading" className="text-sm font-semibold tracking-tight">
          Notes <span className="ml-1 tabular-nums text-muted-foreground">{formatCompact(total)}</span>
        </h2>
        <SortToggle value={sort} onChange={changeSort} disabled={busy} />
      </div>

      <div className="min-h-0 overflow-y-auto overscroll-contain px-4 py-4">
        {optimisticNotes.length === 0 ? (
          <EmptyState
            icon={MessageSquareText}
            title="No notes yet"
            description={
              review
                ? "Play the video and write down what you notice. Notes remember the moment you wrote them at."
                : "Notes appear here once someone leaves one."
            }
            className="min-h-48 border-0 bg-transparent"
          />
        ) : (
          <div className="flex flex-col gap-5">
            {optimisticNotes.map((note) => (
              <CommentItem
                key={note.id}
                comment={note}
                viewer={viewer}
                videoId={videoId}
                onChange={(next) => updateNote(note.id, next)}
                onSeek={review?.seekTo}
                captureTime={review?.captureTime}
              />
            ))}

            {pagination.has_next ? (
              <Button variant="outline" size="sm" disabled={busy} onClick={loadMore} className="self-center">
                {busy ? <LoaderCircle aria-hidden className="animate-spin" /> : null}
                Show older notes
              </Button>
            ) : null}
          </div>
        )}
      </div>

      <div className="border-t border-border/60 px-4 py-3">
        {viewer ? (
          <CommentForm viewer={viewer} onSubmit={handleCreate} captureTime={review?.captureTime} compact />
        ) : (
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-muted-foreground">Sign in to leave a note.</p>
            <Button asChild size="sm" variant="secondary">
              <Link href={routes.login}>Sign in</Link>
            </Button>
          </div>
        )}
      </div>
    </section>
  );
}

/**
 * Two states, shown as two adjacent buttons rather than a select.
 *
 * A select hides the alternative behind a click; this is a binary the reader
 * flips constantly while reviewing, and both options should be one press away
 * and visibly the same control.
 */
function SortToggle({
  value,
  onChange,
  disabled,
}: {
  value: NoteSort;
  onChange: (next: NoteSort) => void;
  disabled: boolean;
}) {
  const options: Array<{ id: NoteSort; label: string; hint: string }> = [
    { id: "timestamp", label: "Timeline", hint: "Order notes by their moment in the video" },
    { id: "newest", label: "Newest", hint: "Order notes by when they were written" },
  ];

  return (
    <div role="group" aria-label="Note order" className="flex rounded-lg bg-muted p-0.5">
      {options.map((option) => (
        <button
          key={option.id}
          type="button"
          title={option.hint}
          disabled={disabled}
          aria-pressed={value === option.id}
          onClick={() => onChange(option.id)}
          className={cn(
            "rounded-[calc(var(--radius-lg)-2px)] px-2 py-1 text-xs font-medium outline-none",
            "transition-[background-color,color] duration-(--motion-fast) ease-out-quart",
            "focus-visible:ring-2 focus-visible:ring-ring/60 disabled:opacity-60",
            value === option.id
              ? "bg-background text-foreground shadow-xs"
              : "text-muted-foreground hover:text-foreground",
          )}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}
