import { API_BASE } from "@/config/env";
import { getAccessToken, refreshSession } from "@/lib/session";

/**
 * Streams a multipart video upload through to the Go API, attaching the
 * bearer token from the httpOnly cookie.
 *
 * This exists for one reason: upload progress. The browser needs
 * `XMLHttpRequest.upload.onprogress` to show a progress bar, and XHR cannot
 * attach a bearer token it is deliberately unable to read. So the client XHRs
 * the multipart body to this same-origin handler (progress works, the session
 * cookie rides along), and the token is attached here — server-side, where it
 * lives. Sibling of the media proxy at `app/api/media/[...path]/route.ts`.
 *
 * The body is *piped*, never buffered: `request.body` is handed to the
 * upstream fetch as a stream, so a 2 GB upload costs this process a constant
 * amount of memory, not 2 GB per concurrent uploader.
 */

export async function POST(request: Request): Promise<Response> {
  // The access token lasts 15 minutes and a large upload can outlast one, so a
  // missing/stale token is refreshed *before* the transfer starts. There is no
  // refresh-and-replay afterwards — the body is a stream and can only be read
  // once — which is exactly why it must be right up front.
  let token = await getAccessToken();
  if (!token) {
    token = await refreshSession();
  }
  if (!token) {
    return Response.json(
      { success: false, error: { code: "UNAUTHORIZED", message: "Sign in to upload videos." } },
      { status: 401 },
    );
  }

  if (!request.body) {
    return Response.json(
      { success: false, error: { code: "BAD_REQUEST", message: "No upload body." } },
      { status: 400 },
    );
  }

  const headers = new Headers({ Authorization: `Bearer ${token}` });
  // The multipart boundary lives in Content-Type; without it the Go side
  // cannot parse the form at all. Content-Length lets it refuse an oversized
  // upload from the header instead of after reading the whole body.
  const contentType = request.headers.get("content-type");
  if (contentType) headers.set("Content-Type", contentType);
  const contentLength = request.headers.get("content-length");
  if (contentLength) headers.set("Content-Length", contentLength);

  const upstream = await fetch(`${API_BASE}/videos/upload`, {
    method: "POST",
    headers,
    body: request.body,
    cache: "no-store",
    // Required by Node's fetch (undici) whenever the request body is a stream;
    // it declares we will not read the response while still sending.
    duplex: "half",
  } as RequestInit & { duplex: "half" });

  // Pass the API's envelope through verbatim — the client already knows how to
  // read `{success, data}` and `{success, error}`.
  return new Response(upstream.body, {
    status: upstream.status,
    headers: {
      "Content-Type": upstream.headers.get("content-type") ?? "application/json",
    },
  });
}
