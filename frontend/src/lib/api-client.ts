import "server-only";

import { API_BASE } from "@/config/env";
import { ApiError } from "@/lib/api-error";
import { getAccessToken, refreshSession } from "@/lib/session";
import type { Page } from "@/types/common";

/**
 * The one place the app talks to the Go API.
 *
 * It is server-only. Tokens live in httpOnly cookies (see lib/session), so the
 * browser cannot attach a bearer header even if it wanted to, and every request
 * therefore originates from a Server Component, a Server Action, or a Route
 * Handler. Features build on this; nothing calls `fetch` against the API
 * directly.
 */

interface RequestOptions {
  /** Query parameters. Undefined and null values are dropped, not stringified. */
  query?: Record<string, string | number | boolean | undefined | null>;
  body?: unknown;
  /**
   * Send the bearer token. Defaults to true. The handful of endpoints that are
   * strictly public still benefit from a token — an authenticated caller can see
   * their own private videos in a listing — so opting out is rare.
   */
  auth?: boolean;
  /**
   * Next's fetch cache. Defaults to "no-store": this is a personalised API and a
   * cached response is a response served to the wrong user. Public, cacheable
   * reads opt in explicitly.
   */
  cache?: RequestCache;
  /** Time-based revalidation for public reads, in seconds. */
  revalidate?: number;
  tags?: string[];
  signal?: AbortSignal;
}

type Method = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

function buildUrl(path: string, query: RequestOptions["query"]): string {
  const url = new URL(`${API_BASE}${path}`);
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value === undefined || value === null || value === "") continue;
      url.searchParams.set(key, String(value));
    }
  }
  return url.toString();
}

/** The API's failure envelope: `{ success: false, error: { code, message } }`. */
async function toApiError(response: Response): Promise<ApiError> {
  const requestId = response.headers.get("X-Request-ID");

  let code = "UNKNOWN";
  let message = response.statusText || "Request failed";

  try {
    const body: unknown = await response.json();
    const error = (body as { error?: { code?: string; message?: string } }).error;
    if (error?.code) code = error.code;
    if (error?.message) message = error.message;
  } catch {
    // A non-JSON body (a proxy's HTML error page, an empty 502) is not worth
    // failing over — the status code is the signal that matters.
  }

  return new ApiError(response.status, code, message, requestId);
}

async function send(method: Method, path: string, options: RequestOptions, token: string | null): Promise<Response> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (options.body !== undefined) headers["Content-Type"] = "application/json";

  const next =
    options.revalidate !== undefined || options.tags
      ? { revalidate: options.revalidate, tags: options.tags }
      : undefined;

  return fetch(buildUrl(path, options.query), {
    method,
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    cache: next ? undefined : (options.cache ?? "no-store"),
    next,
    signal: options.signal,
  });
}

async function request<T>(method: Method, path: string, options: RequestOptions = {}): Promise<T> {
  const useAuth = options.auth ?? true;
  let token = useAuth ? await getAccessToken() : null;

  let response = await send(method, path, options, token);

  /**
   * A 401 on a request that carried a token means the access token expired —
   * they last fifteen minutes, so this is routine, not exceptional. Refresh once
   * and replay. Refreshing only when a token was actually sent is what stops an
   * anonymous visitor's 401 from triggering a pointless refresh attempt, and
   * replaying only once is what stops a genuinely revoked session from looping.
   */
  if (response.status === 401 && useAuth && token) {
    token = await refreshSession();
    if (token) {
      response = await send(method, path, options, token);
    }
  }

  if (!response.ok) {
    throw await toApiError(response);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  const body: unknown = await response.json();
  return (body as { data: T }).data;
}

/** Unwraps the paginated envelope: items under `data`, counts under `pagination`. */
async function requestPage<T>(path: string, options: RequestOptions = {}): Promise<Page<T>> {
  const useAuth = options.auth ?? true;
  let token = useAuth ? await getAccessToken() : null;

  let response = await send("GET", path, options, token);

  if (response.status === 401 && useAuth && token) {
    token = await refreshSession();
    if (token) {
      response = await send("GET", path, options, token);
    }
  }

  if (!response.ok) {
    throw await toApiError(response);
  }

  const body = (await response.json()) as {
    data: T[] | null;
    pagination: Page<T>["pagination"];
  };

  return {
    // The API answers an empty page with `data: null`, not `data: []`. Callers
    // should never have to think about that.
    items: body.data ?? [],
    pagination: body.pagination,
  };
}

export const api = {
  get: <T>(path: string, options?: RequestOptions) => request<T>("GET", path, options),
  post: <T>(path: string, options?: RequestOptions) => request<T>("POST", path, options),
  put: <T>(path: string, options?: RequestOptions) => request<T>("PUT", path, options),
  patch: <T>(path: string, options?: RequestOptions) => request<T>("PATCH", path, options),
  delete: <T>(path: string, options?: RequestOptions) => request<T>("DELETE", path, options),
  page: requestPage,
};

/**
 * Turns a path the API hands back — `hls_url` and `thumbnail_url` arrive as
 * absolute paths like `/api/v1/videos/{id}/thumbnail`, not full URLs — into
 * something the browser can fetch. The API's origin is not the frontend's.
 */
export function mediaUrl(path: string | null | undefined): string | null {
  if (!path) return null;
  if (path.startsWith("http://") || path.startsWith("https://")) return path;
  return `${API_BASE.replace(/\/api\/v1$/, "")}${path}`;
}
