import type { VideoStatus, VideoVisibility } from "@/types/common";

/**
 * One row of the creator's video table.
 *
 * The API's `Video` is a 25-field object carrying storage details, mime types
 * and transcoding internals the table has no use for. Narrowing it here does
 * two things: it keeps the row component honest about what it renders, and it
 * means the media paths are resolved once — on the server, where `mediaUrl()`
 * is allowed to run — rather than in a client component that has no way to.
 */
export interface StudioVideoRow {
  id: string;
  title: string;
  /** Resolved and fetchable; null until the worker has written one. */
  thumbnailUrl: string | null;
  status: VideoStatus;
  visibility: VideoVisibility;
  /** 0–100. Meaningful only while `status` is "processing". */
  transcodingProgress: number;
  /** Seconds. Zero until the transcode has probed the file. */
  duration: number;
  viewCount: number;
  likeCount: number;
  commentCount: number;
  /** ISO date-time. */
  createdAt: string;
}
