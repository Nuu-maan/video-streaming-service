import { API_BASE } from "@/config/env";
import { getAccessToken, refreshSession } from "@/lib/session";

/**
 * The terminal watch-progress save, delivered by `navigator.sendBeacon`.
 *
 * This exists because a Server Action cannot be relied on while a page is being
 * torn down. `useWatchProgress` writes progress on a ten-second beat, and its
 * effect cleanup fires one last save — but React does NOT run effect cleanups
 * when a tab is closed or the document is discarded. So the single case the hook
 * was written for, and the one its own comment named ("a viewer who closes the
 * tab mid-video must not lose the last nine seconds of their place"), was
 * precisely the case where the last beat was silently dropped.
 *
 * `sendBeacon` is the only mechanism the browser guarantees to flush on unload:
 * it hands the request to the user agent, which sends it independently of the
 * page's lifetime. It cannot set headers — so it cannot carry a bearer token,
 * and would not be allowed to read one anyway (the session is an httpOnly
 * cookie). It CAN carry cookies to its own origin. Hence this handler: the
 * beacon lands here with the session cookie attached, and the token goes on
 * server-side. Same trick, same reason, as `app/api/upload/route.ts` and the
 * media proxy — the token only ever exists on this side of the wire.
 *
 * Fire-and-forget by construction: the browser discards the response of a
 * beacon, so this always answers 204 and never asks the client to handle
 * anything. A lost progress beat costs a bookmark, not a session.
 */

const UUID_SHAPE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

interface Beacon {
  videoId?: unknown;
  position?: unknown;
  duration?: unknown;
  completed?: unknown;
}

/** 204 either way. Nothing is listening, and an error page for an unloading tab is nobody's. */
const done = () => new Response(null, { status: 204 });

export async function POST(request: Request): Promise<Response> {
  let payload: Beacon;
  try {
    payload = (await request.json()) as Beacon;
  } catch {
    return done();
  }

  const { videoId, position, duration, completed } = payload;

  if (typeof videoId !== "string" || !UUID_SHAPE.test(videoId)) return done();
  if (typeof position !== "number" || !Number.isFinite(position)) return done();
  if (typeof duration !== "number" || !Number.isFinite(duration) || duration <= 0) return done();

  // The API rejects `position > duration` with a 400, and a media element will
  // happily report a currentTime a hair past its own duration at the moment it
  // ends. Clamped here as well as in the action — this path never goes through it.
  const safeDuration = Math.floor(duration);
  const safePosition = Math.min(Math.max(Math.floor(position), 0), safeDuration);

  let token = await getAccessToken();
  if (!token) token = await refreshSession();
  if (!token) return done();

  try {
    await fetch(`${API_BASE}/videos/${videoId}/progress`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        position: safePosition,
        duration: safeDuration,
        completed: completed === true,
      }),
      cache: "no-store",
    });
  } catch {
    // Telemetry. It failed; the viewer will never know and should not.
  }

  return done();
}
