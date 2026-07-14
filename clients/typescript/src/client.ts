/**
 * Zero-dependency typed client for the video streaming service API.
 *
 * Works anywhere `fetch` exists: browsers, Node 18+, Next.js server
 * components, edge runtimes. The only browser-specific code path is upload
 * progress (XMLHttpRequest), which degrades gracefully server-side.
 *
 * Canonical API prefix: /api/v1 (prepended automatically — pass only the
 * server origin as baseUrl, e.g. "https://api.example.com").
 */

import type {
  AllMetrics,
  ApiErrorCode,
  BanUserRequest,
  CategoryCount,
  Comment,
  ContentReport,
  CreateCommentRequest,
  CreatePlaylistRequest,
  CreateReportRequest,
  DashboardStats,
  DatabaseMetrics,
  ErrorEnvelope,
  Like,
  ListVideosParams,
  LoginRequest,
  MessageResponse,
  Notification,
  Page,
  PageParams,
  PaginationMeta,
  Playlist,
  PlaylistItem,
  PlaylistVideo,
  QueueMetrics,
  QueueStats,
  RealtimeMetrics,
  RecordViewRequest,
  RecordViewResult,
  RedisMetrics,
  RegisterRequest,
  ReviewReportRequest,
  SaveProgressRequest,
  SearchParams,
  SubscriptionEntry,
  SystemMetrics,
  TimeSeriesData,
  TimeSeriesInterval,
  TokenPair,
  TrendingWindow,
  UpdatePlaylistRequest,
  UploadVideoParams,
  User,
  Video,
  VideoAnalytics,
  VideoQuality,
  VideoSearchItem,
  VideoStatusInfo,
  WatchHistory,
  WatchLaterItem,
  WorkersResponse,
} from "./types.js";

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

/**
 * Thrown for every non-success response. Callers never see the raw
 * `{success, error}` envelope — catch this instead.
 */
export class ApiError extends Error {
  /** HTTP status code (0 when the response could not be parsed at all). */
  readonly status: number;
  /** Machine-readable code from the error envelope, e.g. "NOT_FOUND". */
  readonly code: ApiErrorCode;
  /** X-Request-ID response header, when the server sent one. Useful in bug reports. */
  readonly requestId?: string;

  constructor(status: number, code: ApiErrorCode, message: string, requestId?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

// ---------------------------------------------------------------------------
// Options
// ---------------------------------------------------------------------------

export interface ClientOptions {
  /**
   * Server origin, WITHOUT the /api/v1 prefix and without a trailing slash.
   * Example: "https://api.example.com" or "http://localhost:8080".
   */
  baseUrl: string;
  /** Initial access token (JWT), if you already have one. */
  token?: string;
  /** Initial refresh token. Required for automatic token refresh. */
  refreshToken?: string;
  /**
   * Fired whenever the client obtains a fresh token pair — after an automatic
   * refresh, and after login/register/refresh calls. Persist the tokens here
   * (localStorage, cookie, ...) so a page reload can restore the session.
   */
  onTokenRefresh?: (tokens: TokenPair) => void;
  /** Custom fetch implementation (testing, Next.js instrumented fetch, ...). */
  fetch?: typeof fetch;
}

type Query = Record<string, string | number | boolean | undefined>;

interface RequestOptions {
  query?: Query;
  body?: unknown;
  /** Skip the automatic 401 → refresh → retry dance (used by auth endpoints). */
  noRefresh?: boolean;
  signal?: AbortSignal;
}

interface RawResult {
  status: number;
  requestId?: string;
  json: unknown;
}

// ---------------------------------------------------------------------------
// createClient
// ---------------------------------------------------------------------------

export function createClient(options: ClientOptions) {
  const baseUrl = options.baseUrl.replace(/\/+$/, "");
  const origin = new URL(baseUrl).origin;
  const apiBase = `${baseUrl}/api/v1`;
  const fetchImpl: typeof fetch = options.fetch ?? fetch;
  const onTokenRefresh = options.onTokenRefresh;

  let accessToken: string | undefined = options.token;
  let refreshToken: string | undefined = options.refreshToken;

  /**
   * The single in-flight refresh. When ten parallel requests all hit a 401 at
   * once, they all await THIS promise — exactly one POST /auth/refresh goes
   * out, and every waiter retries with the token it produced.
   */
  let refreshInFlight: Promise<void> | null = null;

  function storeTokens(pair: TokenPair): void {
    accessToken = pair.access_token;
    refreshToken = pair.refresh_token;
  }

  function buildUrl(path: string, query?: Query): string {
    let url = apiBase + path;
    if (query) {
      const params = new URLSearchParams();
      for (const [key, value] of Object.entries(query)) {
        if (value !== undefined) params.set(key, String(value));
      }
      const qs = params.toString();
      if (qs) url += `?${qs}`;
    }
    return url;
  }

  async function rawRequest(
    method: string,
    url: string,
    body: unknown,
    signal?: AbortSignal,
  ): Promise<RawResult> {
    const headers: Record<string, string> = { Accept: "application/json" };
    if (accessToken) headers["Authorization"] = `Bearer ${accessToken}`;
    let payload: string | undefined;
    if (body !== undefined) {
      headers["Content-Type"] = "application/json";
      payload = JSON.stringify(body);
    }
    const res = await fetchImpl(url, { method, headers, body: payload, signal });
    const requestId = res.headers.get("X-Request-ID") ?? undefined;
    const text = await res.text();
    let json: unknown = undefined;
    if (text) {
      try {
        json = JSON.parse(text);
      } catch {
        throw new ApiError(res.status, "INTERNAL_ERROR", `Non-JSON response from ${url}`, requestId);
      }
    }
    return { status: res.status, requestId, json };
  }

  function throwFrom(result: RawResult): never {
    const env = result.json as ErrorEnvelope | undefined;
    if (env && env.success === false && env.error) {
      throw new ApiError(result.status, env.error.code, env.error.message, result.requestId);
    }
    throw new ApiError(
      result.status,
      result.status === 401 ? "UNAUTHORIZED" : "INTERNAL_ERROR",
      `Request failed with status ${result.status}`,
      result.requestId,
    );
  }

  /**
   * Refresh the access token, serialised: concurrent callers share one
   * network call. Never called for /auth/refresh itself (noRefresh guards
   * against refreshing a refresh — the infinite-loop case).
   */
  function refreshTokens(): Promise<void> {
    if (refreshInFlight) return refreshInFlight;
    const current = refreshToken;
    if (!current) return Promise.reject(new ApiError(401, "UNAUTHORIZED", "No refresh token held"));
    refreshInFlight = (async () => {
      try {
        const result = await rawRequest("POST", buildUrl("/auth/refresh"), {
          refresh_token: current,
        });
        if (result.status !== 200) {
          // Refresh token expired or revoked: drop it so we don't retry forever.
          refreshToken = undefined;
          throwFrom(result);
        }
        const pair = (result.json as { data: TokenPair }).data;
        storeTokens(pair);
        onTokenRefresh?.(pair);
      } finally {
        refreshInFlight = null;
      }
    })();
    return refreshInFlight;
  }

  /**
   * Core request: send, and on a 401 refresh once and retry once.
   * Returns the unwrapped `data` — callers never touch the envelope.
   */
  async function request<T>(method: string, path: string, opts: RequestOptions = {}): Promise<T> {
    const url = buildUrl(path, opts.query);
    let result = await rawRequest(method, url, opts.body, opts.signal);
    if (result.status === 401 && !opts.noRefresh && refreshToken) {
      await refreshTokens(); // throws if the refresh itself fails
      result = await rawRequest(method, url, opts.body, opts.signal);
    }
    if (result.status < 200 || result.status >= 300) throwFrom(result);
    return (result.json as { data: T }).data;
  }

  /** Like request(), but returns `{items, pagination}` from a list envelope. */
  async function requestPage<T>(
    method: string,
    path: string,
    opts: RequestOptions = {},
  ): Promise<Page<T>> {
    const url = buildUrl(path, opts.query);
    let result = await rawRequest(method, url, opts.body, opts.signal);
    if (result.status === 401 && !opts.noRefresh && refreshToken) {
      await refreshTokens();
      result = await rawRequest(method, url, opts.body, opts.signal);
    }
    if (result.status < 200 || result.status >= 300) throwFrom(result);
    const env = result.json as { data: T[]; pagination: PaginationMeta };
    return { items: env.data ?? [], pagination: env.pagination };
  }

  // -------------------------------------------------------------------------
  // Upload (multipart)
  //
  // fetch cannot report REQUEST body progress — the upload stream is opaque
  // to it. XMLHttpRequest is the only web API that fires upload.onprogress,
  // so when onProgress is wanted and XHR exists (i.e. in a browser) we use
  // XHR; in Node / server components we fall back to fetch and simply don't
  // report progress.
  // -------------------------------------------------------------------------

  function xhrUpload(
    url: string,
    form: FormData,
    params: UploadVideoParams,
  ): Promise<RawResult> {
    return new Promise<RawResult>((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open("POST", url);
      xhr.responseType = "text";
      if (accessToken) xhr.setRequestHeader("Authorization", `Bearer ${accessToken}`);
      xhr.setRequestHeader("Accept", "application/json");
      if (params.onProgress) {
        xhr.upload.onprogress = (event: ProgressEvent) => {
          if (event.lengthComputable && event.total > 0) {
            params.onProgress!(event.loaded / event.total, event.loaded, event.total);
          }
        };
      }
      if (params.signal) {
        // An already-aborted signal must reject before send(): xhr.abort()
        // before send() is a no-op and the upload would proceed anyway.
        if (params.signal.aborted) {
          reject(new ApiError(0, "INTERNAL_ERROR", "Upload aborted"));
          return;
        }
        params.signal.addEventListener("abort", () => xhr.abort(), { once: true });
      }
      xhr.onload = () => {
        const requestId = xhr.getResponseHeader("X-Request-ID") ?? undefined;
        let json: unknown = undefined;
        if (xhr.responseText) {
          try {
            json = JSON.parse(xhr.responseText);
          } catch {
            reject(new ApiError(xhr.status, "INTERNAL_ERROR", "Non-JSON upload response", requestId));
            return;
          }
        }
        resolve({ status: xhr.status, requestId, json });
      };
      xhr.onerror = () => reject(new ApiError(0, "INTERNAL_ERROR", "Network error during upload"));
      xhr.onabort = () => reject(new ApiError(0, "INTERNAL_ERROR", "Upload aborted"));
      xhr.send(form);
    });
  }

  async function fetchUpload(url: string, form: FormData, signal?: AbortSignal): Promise<RawResult> {
    const headers: Record<string, string> = { Accept: "application/json" };
    if (accessToken) headers["Authorization"] = `Bearer ${accessToken}`;
    // Do NOT set Content-Type: fetch adds the multipart boundary itself.
    const res = await fetchImpl(url, { method: "POST", headers, body: form, signal });
    const requestId = res.headers.get("X-Request-ID") ?? undefined;
    const text = await res.text();
    let json: unknown = undefined;
    if (text) {
      try {
        json = JSON.parse(text);
      } catch {
        throw new ApiError(res.status, "INTERNAL_ERROR", "Non-JSON upload response", requestId);
      }
    }
    return { status: res.status, requestId, json };
  }

  async function upload(params: UploadVideoParams): Promise<Video> {
    const url = buildUrl("/videos/upload");
    const form = new FormData();
    const filename =
      params.filename ??
      (typeof File !== "undefined" && params.file instanceof File ? params.file.name : "video");
    form.append("video", params.file, filename);
    form.append("title", params.title);
    if (params.description !== undefined) form.append("description", params.description);
    if (params.visibility !== undefined) form.append("visibility", params.visibility);

    const send = (): Promise<RawResult> =>
      typeof XMLHttpRequest !== "undefined"
        ? xhrUpload(url, form, params)
        : fetchUpload(url, form, params.signal);

    let result = await send();
    if (result.status === 401 && refreshToken) {
      await refreshTokens();
      result = await send();
    }
    if (result.status < 200 || result.status >= 300) throwFrom(result);
    return (result.json as { data: Video }).data;
  }

  // -------------------------------------------------------------------------
  // The client surface
  // -------------------------------------------------------------------------

  const client = {
    /** Read the tokens the client currently holds (e.g. to persist them). */
    getTokens(): { accessToken?: string; refreshToken?: string } {
      return { accessToken, refreshToken };
    },
    /** Replace the tokens (e.g. after restoring a session from storage). */
    setTokens(access?: string, refresh?: string): void {
      accessToken = access;
      refreshToken = refresh;
    },

    // -----------------------------------------------------------------------
    auth: {
      /** POST /auth/register — create an account. Tokens are stored on the client. */
      async register(body: RegisterRequest): Promise<TokenPair> {
        const pair = await request<TokenPair>("POST", "/auth/register", { body, noRefresh: true });
        storeTokens(pair);
        onTokenRefresh?.(pair);
        return pair;
      },
      /**
       * POST /auth/login — `identifier` is a username OR an email.
       * Tokens are stored on the client.
       */
      async login(body: LoginRequest): Promise<TokenPair> {
        const pair = await request<TokenPair>("POST", "/auth/login", { body, noRefresh: true });
        storeTokens(pair);
        onTokenRefresh?.(pair);
        return pair;
      },
      /** POST /auth/refresh — manually exchange the held refresh token for new tokens. */
      async refresh(): Promise<{ accessToken?: string; refreshToken?: string }> {
        await refreshTokens();
        return { accessToken, refreshToken };
      },
      /** GET /auth/me — the authenticated caller's own account. */
      me(): Promise<User> {
        return request<User>("GET", "/auth/me");
      },
      /** POST /auth/logout — revoke the current access token (and refresh token). Clears held tokens. */
      async logout(): Promise<MessageResponse> {
        const body = refreshToken ? { refresh_token: refreshToken } : undefined;
        try {
          return await request<MessageResponse>("POST", "/auth/logout", { body, noRefresh: true });
        } finally {
          accessToken = undefined;
          refreshToken = undefined;
        }
      },
      /** POST /auth/logout-all — revoke every session on every device. Clears held tokens. */
      async logoutAll(): Promise<MessageResponse> {
        try {
          return await request<MessageResponse>("POST", "/auth/logout-all");
        } finally {
          accessToken = undefined;
          refreshToken = undefined;
        }
      },
      /** POST /auth/verify-email/send — (re)send the verification email. */
      sendVerificationEmail(): Promise<MessageResponse> {
        return request<MessageResponse>("POST", "/auth/verify-email/send");
      },
      /** POST /auth/verify-email — consume an emailed verification token. */
      verifyEmail(token: string): Promise<MessageResponse> {
        return request<MessageResponse>("POST", "/auth/verify-email", {
          body: { token },
          noRefresh: true,
        });
      },
      /** POST /auth/forgot-password — always 200, whether or not the address exists. */
      forgotPassword(email: string): Promise<MessageResponse> {
        return request<MessageResponse>("POST", "/auth/forgot-password", {
          body: { email },
          noRefresh: true,
        });
      },
      /** POST /auth/reset-password — consume a reset token and set a new password. */
      resetPassword(token: string, password: string): Promise<MessageResponse> {
        return request<MessageResponse>("POST", "/auth/reset-password", {
          body: { token, password },
          noRefresh: true,
        });
      },
      /** POST /me/change-password — change password while logged in. */
      changePassword(currentPassword: string, newPassword: string): Promise<MessageResponse> {
        return request<MessageResponse>("POST", "/me/change-password", {
          body: { current_password: currentPassword, new_password: newPassword },
        });
      },
    },

    // -----------------------------------------------------------------------
    videos: {
      /** GET /videos — public videos; pass mine=true (authed) for your own. */
      list(params: ListVideosParams = {}): Promise<Page<Video>> {
        return requestPage<Video>("GET", "/videos", {
          query: {
            page: params.page,
            limit: params.limit,
            search: params.search,
            status: params.status,
            mine: params.mine ? "true" : undefined,
          },
        });
      },
      /** GET /videos/:id — private videos 404 for non-owners. */
      get(id: string): Promise<Video> {
        return request<Video>("GET", `/videos/${id}`);
      },
      /** GET /videos/:id/status — transcoding progress; poll while status is "processing". */
      status(id: string): Promise<VideoStatusInfo> {
        return request<VideoStatusInfo>("GET", `/videos/${id}/status`);
      },
      /**
       * POST /videos/upload — multipart upload, queued for transcoding.
       * onProgress fires only in the browser (see the comment on xhrUpload).
       */
      upload,
      /** DELETE /videos/:id — owner only (or delete_any_video). */
      delete(id: string): Promise<MessageResponse> {
        return request<MessageResponse>("DELETE", `/videos/${id}`);
      },
    },

    // -----------------------------------------------------------------------
    social: {
      /** PUT /videos/:id/like — isLike=false records a dislike. */
      like(videoId: string, isLike: boolean): Promise<Like> {
        return request<Like>("PUT", `/videos/${videoId}/like`, { body: { is_like: isLike } });
      },
      /** GET /videos/:id/like — the caller's current rating (404 if not rated). */
      getLike(videoId: string): Promise<Like> {
        return request<Like>("GET", `/videos/${videoId}/like`);
      },
      /** DELETE /videos/:id/like — clear the caller's rating. */
      removeLike(videoId: string): Promise<MessageResponse & { video_id: string }> {
        return request("DELETE", `/videos/${videoId}/like`);
      },

      /** GET /videos/:id/comments — top-level comments, pinned first. */
      comments(videoId: string, params: PageParams = {}): Promise<Page<Comment>> {
        return requestPage<Comment>("GET", `/videos/${videoId}/comments`, { query: { ...params } });
      },
      /** POST /videos/:id/comments — comment, or reply when parent_id is set. */
      comment(videoId: string, body: CreateCommentRequest): Promise<Comment> {
        return request<Comment>("POST", `/videos/${videoId}/comments`, { body });
      },
      /** GET /comments/:id/replies — a comment's replies, oldest first. */
      replies(commentId: string, params: PageParams = {}): Promise<Page<Comment>> {
        return requestPage<Comment>("GET", `/comments/${commentId}/replies`, { query: { ...params } });
      },
      /** PATCH /comments/:id — edit your own comment. */
      updateComment(commentId: string, content: string): Promise<Comment> {
        return request<Comment>("PATCH", `/comments/${commentId}`, { body: { content } });
      },
      /** DELETE /comments/:id — author, video owner, or moderator. */
      deleteComment(commentId: string): Promise<MessageResponse & { comment_id: string }> {
        return request("DELETE", `/comments/${commentId}`);
      },

      /** POST /users/:id/subscribe — idempotent; self-subscribe is a 400. */
      subscribe(creatorId: string): Promise<MessageResponse & { creator_id: string }> {
        return request("POST", `/users/${creatorId}/subscribe`);
      },
      /** DELETE /users/:id/subscribe. */
      unsubscribe(creatorId: string): Promise<MessageResponse & { creator_id: string }> {
        return request("DELETE", `/users/${creatorId}/subscribe`);
      },
      /** GET /users/:id/subscribers. */
      subscribers(creatorId: string, params: PageParams = {}): Promise<Page<SubscriptionEntry>> {
        return requestPage<SubscriptionEntry>("GET", `/users/${creatorId}/subscribers`, {
          query: { ...params },
        });
      },
      /** GET /me/subscriptions — creators the caller follows. */
      subscriptions(params: PageParams = {}): Promise<Page<SubscriptionEntry>> {
        return requestPage<SubscriptionEntry>("GET", "/me/subscriptions", { query: { ...params } });
      },

      playlists: {
        /** POST /playlists. */
        create(body: CreatePlaylistRequest): Promise<Playlist> {
          return request<Playlist>("POST", "/playlists", { body });
        },
        /** GET /playlists/:id — private playlists 404 for non-owners. */
        get(id: string): Promise<Playlist> {
          return request<Playlist>("GET", `/playlists/${id}`);
        },
        /** PATCH /playlists/:id — at least one field required. Owner only. */
        update(id: string, body: UpdatePlaylistRequest): Promise<Playlist> {
          return request<Playlist>("PATCH", `/playlists/${id}`, { body });
        },
        /** DELETE /playlists/:id — owner only. */
        delete(id: string): Promise<MessageResponse & { playlist_id: string }> {
          return request("DELETE", `/playlists/${id}`);
        },
        /** POST /playlists/:id/videos — append a video (409 if already present). */
        addVideo(playlistId: string, videoId: string): Promise<PlaylistVideo> {
          return request<PlaylistVideo>("POST", `/playlists/${playlistId}/videos`, {
            body: { video_id: videoId },
          });
        },
        /** DELETE /playlists/:id/videos/:videoId. */
        removeVideo(playlistId: string, videoId: string): Promise<MessageResponse & { video_id: string }> {
          return request("DELETE", `/playlists/${playlistId}/videos/${videoId}`);
        },
        /** GET /playlists/:id/videos — items in position order. */
        videos(playlistId: string, params: PageParams = {}): Promise<Page<PlaylistItem>> {
          return requestPage<PlaylistItem>("GET", `/playlists/${playlistId}/videos`, {
            query: { ...params },
          });
        },
        /** GET /me/playlists — the caller's playlists, private included. */
        mine(params: PageParams = {}): Promise<Page<Playlist>> {
          return requestPage<Playlist>("GET", "/me/playlists", { query: { ...params } });
        },
      },

      watchLater: {
        /** PUT /videos/:id/watch-later — idempotent save. */
        add(videoId: string): Promise<MessageResponse & { video_id: string }> {
          return request("PUT", `/videos/${videoId}/watch-later`);
        },
        /** DELETE /videos/:id/watch-later. */
        remove(videoId: string): Promise<MessageResponse & { video_id: string }> {
          return request("DELETE", `/videos/${videoId}/watch-later`);
        },
        /** GET /me/watch-later — most recently saved first. */
        list(params: PageParams = {}): Promise<Page<WatchLaterItem>> {
          return requestPage<WatchLaterItem>("GET", "/me/watch-later", { query: { ...params } });
        },
      },

      notifications: {
        /** GET /me/notifications — newest first; unreadOnly narrows to unread. */
        list(params: PageParams & { unreadOnly?: boolean } = {}): Promise<Page<Notification>> {
          return requestPage<Notification>("GET", "/me/notifications", {
            query: {
              page: params.page,
              limit: params.limit,
              unread: params.unreadOnly ? "true" : undefined,
            },
          });
        },
        /** GET /me/notifications/unread-count — for badge rendering. */
        unreadCount(): Promise<{ unread_count: number }> {
          return request("GET", "/me/notifications/unread-count");
        },
        /** POST /me/notifications/read-all. */
        readAll(): Promise<MessageResponse & { marked: number }> {
          return request("POST", "/me/notifications/read-all");
        },
        /** POST /me/notifications/:id/read. */
        markRead(id: string): Promise<MessageResponse & { notification_id: string }> {
          return request("POST", `/me/notifications/${id}/read`);
        },
      },

      /** POST /reports — report a video, user, or comment (409 on duplicate). */
      report(body: CreateReportRequest): Promise<ContentReport> {
        return request<ContentReport>("POST", "/reports", { body });
      },
    },

    // -----------------------------------------------------------------------
    discovery: {
      /** GET /search — full-text search over public, ready videos. q is required. */
      search(params: SearchParams): Promise<Page<VideoSearchItem>> {
        return requestPage<VideoSearchItem>("GET", "/search", { query: { ...params } });
      },
      /** GET /search/suggest — up to ten title suggestions for autocomplete. */
      suggest(q: string): Promise<string[]> {
        return request<string[]>("GET", "/search/suggest", { query: { q } });
      },
      /** GET /categories — distinct categories with video counts. */
      categories(): Promise<CategoryCount[]> {
        return request<CategoryCount[]>("GET", "/categories");
      },
      /** GET /videos/trending — NOT paginated; returns a plain array. */
      trending(window?: TrendingWindow, limit?: number): Promise<VideoSearchItem[]> {
        return request<VideoSearchItem[]>("GET", "/videos/trending", { query: { window, limit } });
      },
      /** GET /videos/:id/related — NOT paginated; returns a plain array. */
      related(videoId: string, limit?: number): Promise<VideoSearchItem[]> {
        return request<VideoSearchItem[]>("GET", `/videos/${videoId}/related`, { query: { limit } });
      },
      /** GET /me/feed — new videos from subscribed creators. Requires auth. */
      feed(params: PageParams = {}): Promise<Page<VideoSearchItem>> {
        return requestPage<VideoSearchItem>("GET", "/me/feed", { query: { ...params } });
      },
    },

    // -----------------------------------------------------------------------
    engagement: {
      /**
       * POST /videos/:id/view — record one view (deduped server-side).
       * Playback does NOT auto-count; call this when playback starts.
       * Anonymous callers MUST pass session_id.
       * Resolves { counted: true } when counted (201), { counted: false } on a
       * deduplicated repeat (200).
       */
      recordView(videoId: string, body: RecordViewRequest = {}): Promise<RecordViewResult> {
        return request<RecordViewResult>("POST", `/videos/${videoId}/view`, { body });
      },
      /** POST /videos/:id/progress — upsert the resume position (seconds). */
      saveProgress(videoId: string, body: SaveProgressRequest): Promise<MessageResponse> {
        return request<MessageResponse>("POST", `/videos/${videoId}/progress`, { body });
      },
      history: {
        /** GET /me/history — most recently watched first. */
        list(params: PageParams = {}): Promise<Page<WatchHistory>> {
          return requestPage<WatchHistory>("GET", "/me/history", { query: { ...params } });
        },
        /** DELETE /me/history — clear everything. */
        clear(): Promise<MessageResponse> {
          return request<MessageResponse>("DELETE", "/me/history");
        },
        /** DELETE /me/history/:videoId — remove one entry. */
        remove(videoId: string): Promise<MessageResponse> {
          return request<MessageResponse>("DELETE", `/me/history/${videoId}`);
        },
      },
    },

    // -----------------------------------------------------------------------
    media: {
      /**
       * Resolve an API-relative media path (video.hls_url / video.thumbnail_url
       * are absolute PATHS like "/api/v1/videos/{id}/hls/master.m3u8", not full
       * URLs) against the client's origin, producing a URL you can hand to
       * hls.js, a <video> tag, or an <img> tag.
       */
      resolve(path: string): string {
        return path.startsWith("http") ? path : origin + path;
      },
      /** Full URL of a video's HLS master playlist (for hls.js / Safari native HLS). */
      hlsUrl(video: Video | string): string {
        if (typeof video !== "string" && video.hls_url) return origin + video.hls_url;
        const id = typeof video === "string" ? video : video.id;
        return `${apiBase}/videos/${id}/hls/master.m3u8`;
      },
      /** Full URL of one quality's HLS media playlist. */
      hlsQualityUrl(videoId: string, quality: VideoQuality): string {
        return `${apiBase}/videos/${videoId}/hls/${quality}/playlist.m3u8`;
      },
      /** Full URL of the progressive MP4 fallback (supports Range → 206). */
      streamUrl(videoId: string, quality: VideoQuality): string {
        return `${apiBase}/videos/${videoId}/stream/${quality}`;
      },
      /** Full URL of the poster image. */
      thumbnailUrl(video: Video | string): string {
        if (typeof video !== "string" && video.thumbnail_url) return origin + video.thumbnail_url;
        const id = typeof video === "string" ? video : video.id;
        return `${apiBase}/videos/${id}/thumbnail`;
      },
    },

    // -----------------------------------------------------------------------
    admin: {
      /** POST /admin/videos/:id/retry — re-queue a FAILED video for transcoding. */
      retryVideo(videoId: string): Promise<MessageResponse & { video_id: string }> {
        return request("POST", `/admin/videos/${videoId}/retry`);
      },
      /** GET /admin/queue/stats. */
      queueStats(): Promise<QueueStats> {
        return request<QueueStats>("GET", "/admin/queue/stats");
      },
      /** GET /admin/workers. */
      workers(): Promise<WorkersResponse> {
        return request<WorkersResponse>("GET", "/admin/workers");
      },
      /** DELETE /admin/videos/:id/cache — flush cached HLS playlists. */
      clearVideoCache(videoId: string): Promise<MessageResponse & { video_id: string }> {
        return request("DELETE", `/admin/videos/${videoId}/cache`);
      },
      /** GET /admin/reports/pending. */
      pendingReports(params: PageParams = {}): Promise<Page<ContentReport>> {
        return requestPage<ContentReport>("GET", "/admin/reports/pending", { query: { ...params } });
      },
      /** POST /admin/reports/:id/review — action=ban_user additionally needs manage_users. */
      reviewReport(
        reportId: string,
        body: ReviewReportRequest,
      ): Promise<MessageResponse & { report_id: string; action: string }> {
        return request("POST", `/admin/reports/${reportId}/review`, { body });
      },
      /** POST /admin/users/:id/ban — omit duration for a permanent ban. */
      banUser(userId: string, body: BanUserRequest): Promise<MessageResponse & { user_id: string }> {
        return request("POST", `/admin/users/${userId}/ban`, { body });
      },
      /** POST /admin/users/:id/unban. */
      unbanUser(userId: string): Promise<MessageResponse & { user_id: string }> {
        return request("POST", `/admin/users/${userId}/unban`);
      },

      analytics: {
        /** GET /admin/analytics/dashboard. */
        dashboard(): Promise<DashboardStats> {
          return request<DashboardStats>("GET", "/admin/analytics/dashboard");
        },
        /** GET /admin/analytics/realtime — always uncached. */
        realtime(): Promise<RealtimeMetrics> {
          return request<RealtimeMetrics>("GET", "/admin/analytics/realtime");
        },
        /** GET /admin/analytics/top-videos — plain array; limit 1-50, default 10. */
        topVideos(limit?: number): Promise<VideoAnalytics[]> {
          return request<VideoAnalytics[]>("GET", "/admin/analytics/top-videos", {
            query: { limit },
          });
        },
        /** GET /admin/analytics/videos/:id. */
        video(videoId: string): Promise<VideoAnalytics> {
          return request<VideoAnalytics>("GET", `/admin/analytics/videos/${videoId}`);
        },
        /** GET /admin/analytics/videos/:id/views — view count time series. */
        videoViews(videoId: string, interval?: TimeSeriesInterval): Promise<TimeSeriesData> {
          return request<TimeSeriesData>("GET", `/admin/analytics/videos/${videoId}/views`, {
            query: { interval },
          });
        },
      },

      monitoring: {
        /** GET /admin/monitoring/metrics — everything in one payload. */
        all(): Promise<AllMetrics> {
          return request<AllMetrics>("GET", "/admin/monitoring/metrics");
        },
        /** GET /admin/monitoring/system. */
        system(): Promise<SystemMetrics> {
          return request<SystemMetrics>("GET", "/admin/monitoring/system");
        },
        /** GET /admin/monitoring/queue. */
        queue(): Promise<QueueMetrics> {
          return request<QueueMetrics>("GET", "/admin/monitoring/queue");
        },
        /** GET /admin/monitoring/database. */
        database(): Promise<DatabaseMetrics> {
          return request<DatabaseMetrics>("GET", "/admin/monitoring/database");
        },
        /** GET /admin/monitoring/redis. */
        redis(): Promise<RedisMetrics> {
          return request<RedisMetrics>("GET", "/admin/monitoring/redis");
        },
      },
    },
  };

  return client;
}

/** The type of the object returned by createClient. */
export type VideoStreamingClient = ReturnType<typeof createClient>;
