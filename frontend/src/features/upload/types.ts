import type { Video } from "@/types/common";

/** Live numbers for the transfer, derived from XHR progress events. */
export interface UploadProgress {
  loadedBytes: number;
  totalBytes: number;
  /** 0–100, floored so it never shows 100% while bytes remain. */
  percent: number;
  /** Smoothed over a sliding window; 0 until there is enough signal. */
  bytesPerSecond: number;
  /** Seconds remaining at the current speed; null until the speed settles. */
  etaSeconds: number | null;
}

/**
 * The honest lifecycle of an upload, as a discriminated union so a component
 * can never read a progress number that does not exist in the current phase.
 *
 *   idle → uploading → processing → ready
 *                    ↘ failed (at any point)
 */
export type UploadState =
  | { phase: "idle" }
  | { phase: "uploading"; progress: UploadProgress }
  | { phase: "processing"; video: Video; transcodingProgress: number }
  | { phase: "ready"; video: Video }
  | { phase: "failed"; message: string };

export type UploadPhase = UploadState["phase"];
