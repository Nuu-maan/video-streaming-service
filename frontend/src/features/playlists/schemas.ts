import { z } from "zod";

export const MAX_PLAYLIST_TITLE = 255;
export const MAX_PLAYLIST_DESCRIPTION = 5000;

export const playlistSchema = z.object({
  title: z
    .string()
    .trim()
    .min(1, "Give the playlist a name.")
    .max(MAX_PLAYLIST_TITLE, `At most ${MAX_PLAYLIST_TITLE} characters.`),
  description: z.string().trim().max(MAX_PLAYLIST_DESCRIPTION).optional(),
  visibility: z.enum(["public", "private", "unlisted"]),
});

export type PlaylistInput = z.infer<typeof playlistSchema>;

/** PATCH accepts a subset, but demands at least one field. */
export const playlistUpdateSchema = playlistSchema.partial().refine(
  (value) => Object.values(value).some((field) => field !== undefined),
  { message: "Nothing to change." },
);

export type PlaylistUpdateInput = z.infer<typeof playlistUpdateSchema>;

export const visibilityOptions = [
  { value: "private", label: "Private", hint: "Only you can open it." },
  { value: "unlisted", label: "Unlisted", hint: "Anyone with the link can open it." },
  { value: "public", label: "Public", hint: "Anyone can find it." },
] as const;
