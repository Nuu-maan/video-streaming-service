"use client";

import { VideoPlayer } from "@/features/player/components/video-player";
import { useReview } from "@/features/review/review-context";

interface ReviewPlayerProps {
  videoId: string;
  src: string;
  poster?: string | null;
  title: string;
  trackProgress?: boolean;
  resumeAt?: number | null;
}

/**
 * The player, wired to the review context.
 *
 * A thin wrapper on purpose: <VideoPlayer> takes plain props and knows nothing
 * about notes, so it stays usable anywhere — a studio preview, an embed — and
 * only this adapter carries the coupling.
 */
export function ReviewPlayer(props: ReviewPlayerProps) {
  const review = useReview();

  return <VideoPlayer {...props} ref={review?.attachPlayer} markers={review?.markers} />;
}
