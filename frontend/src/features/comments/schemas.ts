import { z } from "zod";

/** The API's own bound. Mirrored here so a 10,001-character note fails instantly. */
export const MAX_COMMENT_LENGTH = 10_000;

/** Where the counter appears: silent until the limit is actually in sight. */
export const COMMENT_COUNTER_THRESHOLD = MAX_COMMENT_LENGTH - 500;

export const commentSchema = z.object({
  content: z
    .string()
    .trim()
    .min(1, "Write something first.")
    .max(MAX_COMMENT_LENGTH, `Notes are limited to ${MAX_COMMENT_LENGTH.toLocaleString()} characters.`),
  /**
   * Seconds into the video. Optional — a note about the video as a whole has no
   * anchor, which is a different thing from an anchor at second zero, so this is
   * `.optional()` and never defaulted.
   *
   * Integer because the API column is one; a fractional playhead would be
   * truncated server-side and the note would come back anchored somewhere the
   * client did not choose. Rounding here keeps the two agreeing.
   */
  videoTimestamp: z.number().int().min(0).optional(),
});

export type CommentInput = z.infer<typeof commentSchema>;

/** Mirrors the API's `sort` parameter on the comment listing. */
export const noteSorts = ["timestamp", "newest"] as const;
export type NoteSort = (typeof noteSorts)[number];
