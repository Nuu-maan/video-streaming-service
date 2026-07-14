import type { VideoStatus } from "@/types/common";

/**
 * The one shape a video card renders. The API hands back videos in two
 * different schemas — `Video` from /videos and `VideoSearchItem` from
 * trending/feed/search — and the card should not care which. The mappers in
 * `card-data.ts` normalise both into this (and resolve media paths to
 * absolute URLs while they are at it, which is why the mappers are
 * server-only and this file is not).
 */
export interface VideoCardData {
  id: string;
  title: string;
  /** Absolute URL, already resolved via `mediaUrl()`; null until a thumbnail exists. */
  thumbnailUrl: string | null;
  /** Seconds. */
  duration: number;
  viewCount: number;
  /** ISO date-time. */
  createdAt: string;
  /** Only search-shaped items carry the uploader — plain `Video` does not. */
  channelName?: string;
  channelId?: string;
  /** Absent means ready — search and trending only ever surface ready videos. */
  status?: VideoStatus;
  /** 0–100, meaningful while `status` is "processing". */
  transcodingProgress?: number;
}

export type TrendingWindow = "24h" | "7d" | "30d";
