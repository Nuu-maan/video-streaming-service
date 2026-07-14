import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

import { isProtectedPath, routes } from "@/config/routes";

/**
 * Next 16 renamed `middleware.ts` to `proxy.ts`. It runs on the Node runtime and
 * the `runtime` option is not configurable — setting one throws.
 *
 * This does exactly one thing: send an anonymous visitor to the login page
 * instead of rendering an empty account shell they cannot populate. It is a
 * user-experience redirect, NOT an access control.
 *
 * The distinction matters. Next's own docs warn that a Server Function is a POST
 * to the route it is used from, so a matcher change — or simply moving an action
 * to another route — silently removes proxy coverage from it. Authorization is
 * therefore re-checked in each protected layout and again inside every action.
 * Nothing here is load-bearing for security, and nothing should ever become so.
 */
export function proxy(request: NextRequest): NextResponse {
  const { pathname } = request.nextUrl;

  if (!isProtectedPath(pathname)) {
    return NextResponse.next();
  }

  /**
   * Presence of the refresh cookie, not the access cookie: an access token lives
   * fifteen minutes, so a returning visitor almost always arrives with it
   * expired and only the refresh cookie intact. Redirecting on a missing access
   * cookie would sign people out every fifteen minutes.
   *
   * This only asks whether a cookie is *there*. Whether it is valid is the
   * server's business, and it will find out when it tries to use it.
   */
  const hasSession = request.cookies.has("vs_refresh");
  if (hasSession) {
    return NextResponse.next();
  }

  const login = new URL(routes.login, request.url);
  // Carry the destination so signing in returns the visitor to where they meant
  // to go, rather than dumping them on the home page.
  login.searchParams.set("next", pathname);
  return NextResponse.redirect(login);
}

export const config = {
  matcher: [
    "/studio/:path*",
    "/history/:path*",
    "/watch-later/:path*",
    "/playlists/:path*",
    "/subscriptions/:path*",
    "/notifications/:path*",
    "/settings/:path*",
    "/admin/:path*",
  ],
};
