import "server-only";

import { cookies } from "next/headers";

import { API_BASE } from "@/config/env";
import type { TokenPair } from "@/types/common";

/**
 * Tokens live in httpOnly cookies, never in JavaScript.
 *
 * The API is a bearer-token service designed to be called from anywhere, which
 * tempts the obvious approach: keep the access token in localStorage and attach
 * it from the browser. That hands the token to any script that ever manages to
 * run on the page. Instead the browser holds an opaque httpOnly cookie, and the
 * Next server — which can read it and the browser cannot — is the only thing
 * that ever sees a bearer token.
 *
 * The cost is that data fetching happens on the server (in Server Components and
 * Server Actions) rather than from the client. That is the direction Next pushes
 * anyway, so it is a cost worth paying.
 */
const ACCESS_COOKIE = "vs_access";
const REFRESH_COOKIE = "vs_refresh";

/**
 * Cookie lifetimes mirror the token lifetimes the API reports: 15 minutes for an
 * access token, 7 days for a refresh token. The refresh cookie outliving the
 * access cookie is the entire point — it is what lets a returning visitor stay
 * signed in.
 */
const ACCESS_MAX_AGE = 15 * 60;
const REFRESH_MAX_AGE = 7 * 24 * 60 * 60;

const cookieOptions = {
  httpOnly: true,
  sameSite: "lax",
  /**
   * Secure is conditional because a cookie marked Secure is silently dropped
   * over plain http, which would make login appear to succeed and then do
   * nothing at all on a local dev server.
   */
  secure: process.env.NODE_ENV === "production",
  path: "/",
} as const;

export async function setSession(tokens: TokenPair): Promise<void> {
  const store = await cookies();

  store.set(ACCESS_COOKIE, tokens.access_token, {
    ...cookieOptions,
    maxAge: tokens.expires_in ?? ACCESS_MAX_AGE,
  });
  store.set(REFRESH_COOKIE, tokens.refresh_token, {
    ...cookieOptions,
    maxAge: tokens.refresh_expires_in ?? REFRESH_MAX_AGE,
  });
}

export async function clearSession(): Promise<void> {
  const store = await cookies();
  store.delete(ACCESS_COOKIE);
  store.delete(REFRESH_COOKIE);
}

export async function getAccessToken(): Promise<string | null> {
  const store = await cookies();
  return store.get(ACCESS_COOKIE)?.value ?? null;
}

export async function getRefreshToken(): Promise<string | null> {
  const store = await cookies();
  return store.get(REFRESH_COOKIE)?.value ?? null;
}

/**
 * Exchanges the refresh token for a fresh access token.
 *
 * Returns null rather than throwing when the refresh token is missing, expired,
 * or revoked: all three mean the same thing to a caller — nobody is signed in —
 * and none of them is an exceptional condition worth a stack trace. The API
 * rejects a revoked refresh token with 401, which is exactly what happens after
 * a logout, so this is the normal path for a returning visitor, not an error.
 */
export async function refreshSession(): Promise<string | null> {
  const refreshToken = await getRefreshToken();
  if (!refreshToken) return null;

  const response = await fetch(`${API_BASE}/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
    cache: "no-store",
  });

  if (!response.ok) {
    await clearSession();
    return null;
  }

  const body: unknown = await response.json();
  const tokens = (body as { data?: TokenPair }).data;
  if (!tokens?.access_token) {
    await clearSession();
    return null;
  }

  await setSession(tokens);
  return tokens.access_token;
}

export const sessionCookies = {
  access: ACCESS_COOKIE,
  refresh: REFRESH_COOKIE,
} as const;
