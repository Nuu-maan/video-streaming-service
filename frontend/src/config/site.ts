import { env } from "@/config/env";

export const site = {
  name: "Reel",
  description: "Upload, transcode, and stream video with adaptive bitrate HLS.",
  url: env.NEXT_PUBLIC_SITE_URL,
} as const;

/** Bounds that mirror the API's own. Sending a request the server will refuse is a wasted round trip. */
export const limits = {
  /** STORAGE_MAX_FILE_SIZE on the API. Checked here so a 2 GB upload fails instantly, not after uploading. */
  maxUploadBytes: 2 * 1024 * 1024 * 1024,
  /** The API allowlist: mp4, mov, avi, mkv, webm. */
  acceptedVideoTypes: ["video/mp4", "video/quicktime", "video/x-msvideo", "video/x-matroska", "video/webm"],
  acceptedVideoExtensions: [".mp4", ".mov", ".avi", ".mkv", ".webm"],
  maxTitleLength: 255,
  maxDescriptionLength: 5000,
  maxCommentLength: 10000,
  /** The API rejects a limit above 100. */
  pageSize: 24,
} as const;
