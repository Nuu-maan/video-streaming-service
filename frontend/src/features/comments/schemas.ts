import { z } from "zod";

/** The API's own bound. Mirrored here so a 10,001-character comment fails instantly. */
export const MAX_COMMENT_LENGTH = 10_000;

/** Where the counter appears: silent until the limit is actually in sight. */
export const COMMENT_COUNTER_THRESHOLD = MAX_COMMENT_LENGTH - 500;

export const commentSchema = z.object({
  content: z
    .string()
    .trim()
    .min(1, "Write something first.")
    .max(MAX_COMMENT_LENGTH, `Comments are limited to ${MAX_COMMENT_LENGTH.toLocaleString()} characters.`),
});

export type CommentInput = z.infer<typeof commentSchema>;
