/**
 * Every internal path, in one place. A route that exists as a string literal
 * scattered across forty files is a route nobody can safely rename.
 *
 * A note on vocabulary: the UI calls a playlist a "collection" and a comment a
 * "note", and these paths follow the UI. The API keeps its own words — the
 * request still goes to `/playlists` and `/comments` — because the OpenAPI spec
 * is a published contract and renaming a field to suit a frontend is not a
 * frontend's business. The translation happens here and in the display copy;
 * nowhere else.
 */
export const routes = {
  home: "/",
  videos: "/videos",
  video: (id: string) => `/videos/${id}`,
  search: "/search",
  // Search survives the reframe — finding a video you know exists is not the
  // same thing as being fed one you didn't ask for — and category is a filter
  // on it, not a browse destination of its own.
  category: (name: string) => `/search?category=${encodeURIComponent(name)}`,

  login: "/login",
  register: "/register",
  forgotPassword: "/forgot-password",
  resetPassword: "/reset-password",
  verifyEmail: "/verify-email",

  studio: "/studio",
  upload: "/studio/upload",
  videoInsights: (id: string) => `/studio/videos/${id}`,

  history: "/history",
  saved: "/saved",
  collections: "/collections",
  collection: (id: string) => `/collections/${id}`,
  people: "/people",
  notifications: "/notifications",
  settings: "/settings",

  admin: "/admin",
  adminReports: "/admin/reports",
  adminUsers: "/admin/users",
  adminQueue: "/admin/queue",
} as const;

/**
 * Paths that require a session. `proxy.ts` bounces an anonymous visitor away
 * from these before the page renders, purely so they see a login screen instead
 * of an empty shell — it is a redirect, not a security control. The real check
 * happens in the layout and again in every action, because a proxy matcher can
 * be silently bypassed by a refactor and Next's own docs say not to trust it.
 */
export const protectedPaths = [
  "/studio",
  "/history",
  "/saved",
  "/collections",
  "/people",
  "/notifications",
  "/settings",
  "/admin",
] as const;

export function isProtectedPath(pathname: string): boolean {
  return protectedPaths.some((path) => pathname === path || pathname.startsWith(`${path}/`));
}
