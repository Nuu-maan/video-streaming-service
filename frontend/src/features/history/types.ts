import type { VideoCardData } from "@/features/videos/types";

/** A watch-history entry, hydrated with the video it points at. */
export interface HistoryRow {
  /** The history entry's id, not the video's — a rewatch is a new entry. */
  entryId: string;
  video: VideoCardData;
  /** ISO date-time of the last watch. */
  watchedAt: string;
  /** 0–100, how far through they got. Drives the resume bar under the thumbnail. */
  progressPercent: number;
  completed: boolean;
}

export type HistoryResult = { ok: true } | { ok: false; message: string };
