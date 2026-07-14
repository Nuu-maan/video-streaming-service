# @video-streaming/client

A typed, **zero-dependency** TypeScript client for the video streaming service API. Works in the browser, in Node 18+, and in Next.js server components — anywhere `fetch` exists.

The API is a standalone JSON server (canonical prefix `/api/v1`) designed to be consumed by a separate frontend on a different origin. CORS is configured server-side; this client just talks to it.

## Install

```bash
# from a monorepo / git checkout
npm install ./clients/typescript
# then build it once
cd clients/typescript && npm install && npm run build
```

Or copy `src/` straight into your project — it has no runtime dependencies.

## Five-minute quickstart

### 1. Create a client

```ts
import { createClient } from "@video-streaming/client";

const api = createClient({
  baseUrl: "http://localhost:8080", // the server ORIGIN — no /api/v1, the client adds it
  onTokenRefresh: (tokens) => {
    // Fired on login AND on every automatic refresh. Persist tokens here.
    localStorage.setItem("tokens", JSON.stringify({
      access: tokens.access_token,
      refresh: tokens.refresh_token,
    }));
  },
});

// Restoring a session after a page reload:
const saved = JSON.parse(localStorage.getItem("tokens") ?? "null");
if (saved) api.setTokens(saved.access, saved.refresh);
```

### 2. Sign up / log in

```ts
// Register: username + email + password
await api.auth.register({ username: "alice", email: "alice@example.com", password: "s3cret-pw" });

// Log in: `identifier` is a username OR an email (not `username`!)
const { user } = await api.auth.login({ identifier: "alice", password: "s3cret-pw" });
console.log(user.role); // "user"
```

Access tokens live 15 minutes, refresh tokens 7 days. **You never need to refresh manually**: on any 401 the client calls `/auth/refresh` once, retries the original request, and fires `onTokenRefresh` with the new pair. Concurrent 401s share a single refresh — ten parallel requests trigger one refresh, not ten.

### 3. Browse and search

```ts
// Paginated lists return { items, pagination } — the envelope is unwrapped for you.
const { items, pagination } = await api.videos.list({ page: 1, limit: 20 });
console.log(pagination.total_pages, pagination.has_next);

const results = await api.discovery.search({ q: "golang tutorial", sort: "views" });
const trending = await api.discovery.trending("7d");     // plain array, not paginated
const suggestions = await api.discovery.suggest("gol");  // string[]
```

### 4. Play a video

`video.hls_url` and `video.thumbnail_url` are absolute **paths** (`/api/v1/videos/{id}/...`), not full URLs. Use `api.media` to resolve them:

```tsx
import Hls from "hls.js";

const video = await api.videos.get(id);

// Poster
img.src = api.media.thumbnailUrl(video);

// HLS playback (hls.js, or native on Safari)
const hls = new Hls();
hls.loadSource(api.media.hlsUrl(video)); // resolves hls_url against baseUrl
hls.attachMedia(videoElement);

// Progressive MP4 fallback (supports seeking via Range requests)
videoElement.src = api.media.streamUrl(video.id, "720p");
```

Views are **not** counted automatically by playback. Record one when playback starts:

```ts
// Logged in: deduped by user id
await api.engagement.recordView(video.id, { quality: "720p" });

// Anonymous: session_id is REQUIRED and must be STABLE per browser session —
// a fresh id on every call would defeat the server-side view dedupe.
let sessionId = sessionStorage.getItem("session_id");
if (!sessionId) {
  sessionId = crypto.randomUUID();
  sessionStorage.setItem("session_id", sessionId);
}
await api.engagement.recordView(video.id, { session_id: sessionId });

// Save resume position every ~10s
await api.engagement.saveProgress(video.id, { position: 42, duration: 300, completed: false });
```

### 5. Upload with a progress bar

```tsx
const uploaded = await api.videos.upload({
  file: fileInput.files![0],
  title: "My first video",
  visibility: "public", // public | private | unlisted
  onProgress: (fraction) => setProgress(Math.round(fraction * 100)),
});

// Poll transcoding until it's ready
const poll = setInterval(async () => {
  const s = await api.videos.status(uploaded.id);
  if (s.status === "ready" || s.status === "failed") clearInterval(poll);
  else setProgress(s.progress); // 0-100
}, 3000);
```

`onProgress` only fires in the browser — `fetch` cannot report upload progress, so the client uses `XMLHttpRequest` when available and falls back to `fetch` (without progress) server-side.

### 6. Social features

```ts
await api.social.like(videoId, true);            // true = like, false = dislike
await api.social.comment(videoId, { content: "Great video!" });
await api.social.comment(videoId, { content: "Agreed", parent_id: commentId }); // reply
await api.social.subscribe(creatorId);
const feed = await api.discovery.feed();          // new videos from subscriptions

const pl = await api.social.playlists.create({ title: "Watch later-er", visibility: "private" });
await api.social.playlists.addVideo(pl.id, videoId);
await api.social.watchLater.add(videoId);
```

## Error handling

Every failed call throws a typed `ApiError` — you never inspect `.success` yourself:

```ts
import { ApiError } from "@video-streaming/client";

try {
  await api.videos.get(id);
} catch (e) {
  if (e instanceof ApiError) {
    e.status;    // 404
    e.code;      // "NOT_FOUND", "VALIDATION_ERROR", "USER_BANNED", ...
    e.message;   // human-readable
    e.requestId; // X-Request-ID, for bug reports
  }
}
```

Worth knowing:

- **Private videos return 404, never 403** — the API does not reveal whether a private video exists.
- Login failures are a 401 with an identical body for "unknown account" and "wrong password".
- Duplicate reports are `409 DUPLICATE_REPORT`; re-adding a playlist video is `409 ALREADY_IN_PLAYLIST`.
- Uploads can fail with `413 FILE_TOO_LARGE` (the limit is deployment-configured — `STORAGE_MAX_FILE_SIZE`, 2 GiB by default) or `415 INVALID_FORMAT`.

## Next.js notes

- **Server components / route handlers**: `createClient` works as-is (native fetch, no browser APIs required). Create a per-request client and pass the user's token: `createClient({ baseUrl, token })`.
- **Client components**: create one shared client (e.g. in a module or context) so the automatic-refresh single-flight applies across your whole app.
- Media URLs from `api.media.*` are plain URLs — safe to put in `<video>`, `<img>`, hls.js, or `next/image` (add the API host to `images.remotePatterns`).

## Method map

| Group | Methods |
| --- | --- |
| `auth` | `register`, `login`, `refresh`, `me`, `logout`, `logoutAll`, `sendVerificationEmail`, `verifyEmail`, `forgotPassword`, `resetPassword`, `changePassword` |
| `videos` | `list`, `get`, `status`, `upload`, `delete` |
| `social` | `like`, `getLike`, `removeLike`, `comments`, `comment`, `replies`, `updateComment`, `deleteComment`, `subscribe`, `unsubscribe`, `subscribers`, `subscriptions`, `report`, `playlists.*`, `watchLater.*`, `notifications.*` |
| `discovery` | `search`, `suggest`, `categories`, `trending`, `related`, `feed` |
| `engagement` | `recordView`, `saveProgress`, `history.list/clear/remove` |
| `media` | `resolve`, `hlsUrl`, `hlsQualityUrl`, `streamUrl`, `thumbnailUrl` |
| `admin` | `retryVideo`, `queueStats`, `workers`, `clearVideoCache`, `pendingReports`, `reviewReport`, `banUser`, `unbanUser`, `analytics.*`, `monitoring.*` |

Admin endpoints require a moderator/admin token (`moderate_content`, `manage_users`, or `view_analytics` permission depending on the route). Bootstrap the first admin with the server's `cmd/admin` CLI: `admin promote --username alice --role admin`.
