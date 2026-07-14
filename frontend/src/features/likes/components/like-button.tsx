"use client";

import { ThumbsDown, ThumbsUp } from "lucide-react";
import { useOptimistic, useState, useTransition } from "react";
import { toast } from "sonner";

import { clearRating, rateVideo } from "@/features/likes/actions";
import type { Rating } from "@/features/likes/types";
import { useSignInPrompt } from "@/hooks/use-sign-in-prompt";
import { formatCompact } from "@/lib/format";
import { cn } from "@/lib/utils";

interface LikeButtonProps {
  videoId: string;
  /** The video's like count as the server last reported it. */
  likeCount: number;
  /** The caller's rating, or null when they have not rated (or are signed out). */
  initialRating: Rating | null;
  isAuthenticated: boolean;
  className?: string;
}

interface RatingState {
  rating: Rating | null;
  likeCount: number;
}

/**
 * The count only tracks likes — the API stores one row per user with an
 * `is_like` flag and reports `like_count`, not a dislike count — so a dislike
 * moves the thumb but not the number, exactly as it does everywhere else.
 */
function applyRating(state: RatingState, next: Rating | null): RatingState {
  const was = state.rating === "like" ? 1 : 0;
  const is = next === "like" ? 1 : 0;
  return { rating: next, likeCount: Math.max(0, state.likeCount + is - was) };
}

/**
 * Rating is one row server-side, so liking a video you disliked flips it rather
 * than stacking. Clicking the thumb you already chose clears the rating.
 *
 * The count moves on click, not on round-trip: `useOptimistic` renders the
 * intended state for the life of the transition and React discards it when the
 * action settles, at which point either the committed state matches (success)
 * or it snaps back to the truth (failure) and we say why.
 */
export function LikeButton({
  videoId,
  likeCount,
  initialRating,
  isAuthenticated,
  className,
}: LikeButtonProps) {
  const [state, setState] = useState<RatingState>({ rating: initialRating, likeCount });
  const [optimistic, applyOptimistic] = useOptimistic(state, applyRating);
  const [, startTransition] = useTransition();
  const promptSignIn = useSignInPrompt();

  function handle(target: Rating) {
    if (!isAuthenticated) {
      promptSignIn(target === "like" ? "Sign in to like videos." : "Sign in to rate videos.");
      return;
    }

    // Pressing the active thumb again is "un-rate", not "rate the same way twice".
    const next: Rating | null = state.rating === target ? null : target;

    startTransition(async () => {
      applyOptimistic(next);
      const result = next ? await rateVideo(videoId, next === "like") : await clearRating(videoId);

      if (result.ok) {
        setState((current) => applyRating(current, next));
        return;
      }
      // No setState: the optimistic value is dropped when the transition ends,
      // so the thumb and the count return to the last known-good state on their own.
      toast.error(result.message);
    });
  }

  const liked = optimistic.rating === "like";
  const disliked = optimistic.rating === "dislike";
  const likes = formatCompact(optimistic.likeCount);

  return (
    /*
     * The PILL is the pressable object, so the pill is what scales — 0.96, the
     * house value, on a press of either half. Scaling a half on its own would
     * shrink it away from the divider and read as a broken seam; and putting the
     * press transform only on the *inactive* icon (what this did) meant that the
     * moment you liked something, pressing the thumb again to un-like gave no
     * feedback at all — the feedback disappeared exactly where the control is
     * used most.
     */
    <div
      className={cn(
        "flex h-9 items-center rounded-full bg-muted/60 ring-1 ring-border/60 ring-inset",
        // `:active` propagates up the ancestor chain, so pressing either half
        // puts the PILL in :active — no :has() needed.
        "transition-transform duration-(--motion-fast) ease-out-quart active:scale-[0.96]",
        className,
      )}
    >
      <button
        type="button"
        onClick={() => handle("like")}
        aria-pressed={liked}
        /* The count lives inside the button, so a bare "Like" label would erase
           it from the accessible name — a screen-reader user would never be told
           how many likes the video has. */
        aria-label={`${liked ? "Remove like" : "Like"}, ${likes} likes`}
        className={cn(
          "flex h-full items-center gap-2 rounded-l-full pr-3 pl-3.5 text-sm font-medium outline-none transition-colors duration-(--motion-fast) hover:bg-muted focus-visible:ring-3 focus-visible:ring-ring/50",
          // brand-500 as text is 3.8:1 on light — under AA. Scoped: 6.5 / 6.6.
          liked && "text-brand-700 dark:text-brand-400",
        )}
      >
        <ThumbsUp
          aria-hidden
          className={cn(
            "size-4 transition-transform duration-(--motion-fast) ease-out-quart",
            // State indication only. Press feedback belongs to the pill above.
            liked && "scale-110 fill-current",
          )}
        />
        <span className="tabular-nums">{likes}</span>
      </button>

      <span aria-hidden className="h-5 w-px bg-border" />

      <button
        type="button"
        onClick={() => handle("dislike")}
        aria-pressed={disliked}
        aria-label={disliked ? "Remove dislike" : "Dislike"}
        className={cn(
          "flex h-full items-center rounded-r-full px-3.5 outline-none transition-colors duration-(--motion-fast) hover:bg-muted focus-visible:ring-3 focus-visible:ring-ring/50",
          disliked && "text-foreground",
        )}
      >
        <ThumbsDown
          aria-hidden
          className={cn(
            "size-4 transition-transform duration-(--motion-fast) ease-out-quart",
            disliked && "scale-110 fill-current",
          )}
        />
      </button>
    </div>
  );
}
