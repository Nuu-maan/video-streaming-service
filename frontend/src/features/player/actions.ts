"use server";

import { api } from "@/lib/api-client";

/**
 * Engagement writes, called from the client player through Server Actions —
 * the browser holds no bearer token (it is an httpOnly cookie), so a client
 * component cannot talk to the Go API directly.
 *
 * Both actions swallow failures. Telemetry must never interrupt playback: a
 * lost view ping or progress beat costs a statistic, while a thrown error
 * costs the viewer an error boundary over a video that was playing fine.
 */

const UUID_SHAPE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

interface RecordViewInput {
  /** Rendition being watched when the view counted, e.g. "720p" or "auto". */
  quality?: string;
  /** Required for anonymous viewers; the API dedupes repeat views on it. */
  sessionId?: string;
}

export async function recordView(videoId: string, input: RecordViewInput): Promise<void> {
  if (!UUID_SHAPE.test(videoId)) return;

  try {
    await api.post(`/videos/${videoId}/view`, {
      body: {
        quality: input.quality,
        source: "web",
        session_id: input.sessionId,
      },
    });
  } catch {
    // A view that fails to count is not the viewer's problem.
  }
}

interface SaveProgressInput {
  /** Seconds into the video. Clamped to duration — the API 400s past it. */
  position: number;
  /** Seconds of total runtime. */
  duration: number;
  completed: boolean;
}

export async function saveWatchProgress(videoId: string, input: SaveProgressInput): Promise<void> {
  if (!UUID_SHAPE.test(videoId)) return;
  if (!Number.isFinite(input.position) || !Number.isFinite(input.duration) || input.duration <= 0) return;

  // The API rejects `position > duration` outright; a floating-point tail from
  // the media element (duration 63.9999…) must never produce a 400.
  const duration = Math.floor(input.duration);
  const position = Math.min(Math.max(Math.floor(input.position), 0), duration);

  try {
    await api.post(`/videos/${videoId}/progress`, {
      body: { position, duration, completed: input.completed },
    });
  } catch {
    // Progress is retried every ten seconds; a dropped beat heals itself.
  }
}
