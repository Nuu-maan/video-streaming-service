import { z } from "zod";

import { limits } from "@/config/site";

/**
 * Client-side mirror of the API's own validation. The point is not to replace
 * the server's rules — it re-checks everything — but to fail before a
 * multi-gigabyte upload leaves the machine, and to phrase the failure next to
 * the field that caused it.
 */
export const uploadDetailsSchema = z.object({
  title: z
    .string()
    .trim()
    .min(1, "Give your video a title.")
    .max(limits.maxTitleLength, `Titles are limited to ${limits.maxTitleLength} characters.`),
  description: z
    .string()
    .max(limits.maxDescriptionLength, `Descriptions are limited to ${limits.maxDescriptionLength.toLocaleString("en")} characters.`),
  visibility: z.enum(["public", "unlisted", "private"]),
});

export type UploadDetails = z.infer<typeof uploadDetailsSchema>;

/**
 * Pre-flight check on the file itself, run the moment it is picked. Rejecting
 * a 2 GB file after uploading it is unforgivable, so this runs before a single
 * byte moves. Returns an error message, or null when the file is acceptable.
 */
export function validateVideoFile(file: File): string | null {
  const dot = file.name.lastIndexOf(".");
  const extension = dot === -1 ? "" : file.name.slice(dot).toLowerCase();

  if (!(limits.acceptedVideoExtensions as readonly string[]).includes(extension)) {
    return `That file type isn't supported. Use ${formatExtensionList()}.`;
  }
  if (file.size === 0) {
    return "That file is empty.";
  }
  if (file.size > limits.maxUploadBytes) {
    return "That file is over the 2 GB limit. Try exporting it at a lower bitrate or resolution.";
  }
  return null;
}

export function formatExtensionList(): string {
  const names = limits.acceptedVideoExtensions.map((ext) => ext.slice(1).toUpperCase());
  return `${names.slice(0, -1).join(", ")} or ${names[names.length - 1]}`;
}
