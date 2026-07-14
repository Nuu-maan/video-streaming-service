import "server-only";

import { cache } from "react";

import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";
import type { User } from "@/types/common";

/**
 * The signed-in user, or null. This is the app's single answer to "who is
 * signed in": layouts call it to decide what chrome to render, protected
 * layouts redirect on null, and pages call it to personalise.
 *
 * Wrapped in React's `cache()`, so a layout and the page inside it asking the
 * same question during one render produce one `GET /auth/me`, not two. The
 * cache is per-request — it is memoisation, not a data cache, and it never
 * leaks one visitor's user into another's render.
 *
 * Null covers every flavour of "nobody is signed in": no cookie, an expired
 * session the refresh could not save, a revoked token. None of those is
 * exceptional, so none of them throws.
 *
 * A 429 ALSO RETURNS NULL, and that is not laziness — it is the difference
 * between a degraded page and no page at all. Every layout in the app calls this
 * function, and an exception here escapes into the layout, which means it takes
 * down the entire route: header, sidebar, content and all. So when this threw on
 * a rate limit, a rate-limited visitor got a bare 500 on EVERY page of the site,
 * including the public ones that need no session whatsoever. (Anonymous limits
 * are shared per-IP, so one busy network was enough to do it.)
 *
 * Treating it as "we cannot establish a session right now" degrades gracefully
 * instead: a public page renders in full, as anonymous — its own data fetches
 * carry their own 429 handling and say "slow down" in place — and a protected
 * page bounces to /login. Both are recoverable in a way a 500 is not. The limit
 * is per-minute, so it heals itself.
 *
 * A genuine failure (the API is down, a 500) still throws — "signed out" and
 * "broken" must not look the same — and `app/error.tsx` catches it. That
 * boundary is also WHY the 429 has to be resolved here rather than there: React
 * strips a server error's message in production and hands the client boundary
 * only a digest, so `error.tsx` cannot tell a rate limit from a crash. The
 * distinction only exists on this side of the wire, so it has to be made here.
 */
export const getCurrentUser = cache(async (): Promise<User | null> => {
  try {
    return await api.get<User>("/auth/me");
  } catch (error) {
    if (!isApiError(error)) throw error;

    if (error.isUnauthorized || error.isForbidden || error.isNotFound) {
      return null;
    }
    if (error.isRateLimited) {
      return null;
    }
    throw error;
  }
});
