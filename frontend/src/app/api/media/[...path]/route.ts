import { API_BASE } from "@/config/env";
import { getAccessToken, refreshSession } from "@/lib/session";

/**
 * Streams a private video's media through the Next server, attaching the bearer
 * token from the httpOnly cookie.
 *
 * A public video does not come through here — the browser fetches it straight
 * from the API, which is cheaper and cacheable. This exists for the case the
 * direct path cannot serve: a private or unlisted video, where the API demands a
 * bearer token that the browser deliberately does not have. hls.js cannot attach
 * one (the cookie is httpOnly, and it is scoped to this origin, not the API's),
 * and handing the token to JavaScript to solve that would undo the reason it is
 * httpOnly in the first place.
 *
 * It is a catch-all because HLS is self-referential: the master playlist names
 * media playlists by relative path, and those name segments the same way. Serving
 * the master from here makes every subsequent fetch resolve here too, without
 * rewriting a single URL inside the playlist.
 */

/** Whitelisted upstream shapes. Anything else is not media and is not proxied. */
const MEDIA_PATH = /^videos\/[0-9a-f-]{36}\/(hls\/.+|stream\/[a-z0-9]+|thumbnail)$/i;

/**
 * Headers worth forwarding upstream. Range is the one that matters: without it
 * the MP4 fallback cannot seek, and the browser would have to download the whole
 * file to play the last second of it.
 */
const FORWARD_REQUEST_HEADERS = ["range", "if-none-match", "if-modified-since"];

/**
 * Headers worth returning. Content-Range and Accept-Ranges are what make a 206
 * meaningful; dropping them turns a working seek into a silent failure.
 */
const FORWARD_RESPONSE_HEADERS = [
  "content-type",
  "content-length",
  "content-range",
  "accept-ranges",
  "cache-control",
  "etag",
  "last-modified",
];

async function proxyMedia(request: Request, path: string, token: string | null): Promise<Response> {
  const headers = new Headers();
  if (token) headers.set("Authorization", `Bearer ${token}`);

  for (const name of FORWARD_REQUEST_HEADERS) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }

  return fetch(`${API_BASE}/${path}`, {
    headers,
    // The body is a video. Buffering it would hold an entire rendition in memory
    // per viewer; passing the stream through costs a constant.
    cache: "no-store",
  });
}

export async function GET(request: Request, context: { params: Promise<{ path: string[] }> }): Promise<Response> {
  const { path } = await context.params;
  const upstream = path.join("/");

  if (!MEDIA_PATH.test(upstream)) {
    return new Response("Not found", { status: 404 });
  }

  let token = await getAccessToken();
  let response = await proxyMedia(request, upstream, token);

  // Segments are fetched for as long as the video plays, so a session will
  // routinely expire mid-playback. Refresh once and replay, or the video simply
  // stops partway through.
  if (response.status === 401 && token) {
    token = await refreshSession();
    if (token) {
      response = await proxyMedia(request, upstream, token);
    }
  }

  const outHeaders = new Headers();
  for (const name of FORWARD_RESPONSE_HEADERS) {
    const value = response.headers.get(name);
    if (value) outHeaders.set(name, value);
  }

  // The API answers a video the caller may not see with 404, never 403, so that
  // its existence is not observable. Preserve that: turning it into a 403 here
  // would leak exactly what the API refused to.
  return new Response(response.body, {
    status: response.status,
    headers: outHeaders,
  });
}
