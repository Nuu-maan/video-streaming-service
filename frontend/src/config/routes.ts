/**
 * Every internal path, in one place. A route that exists as a string literal
 * scattered across forty files is a route nobody can safely rename.
 */
export const routes = {
  home: "/",
  videos: "/videos",
  video: (id: string) => `/videos/${id}`,
  search: "/search",
  category: (name: string) => `/search?category=${encodeURIComponent(name)}`,

  login: "/login",
  register: "/register",
  forgotPassword: "/forgot-password",
  resetPassword: "/reset-password",
  verifyEmail: "/verify-email",

  studio: "/studio",
  upload: "/studio/upload",

  history: "/history",
  watchLater: "/watch-later",
  playlists: "/playlists",
  playlist: (id: string) => `/playlists/${id}`,
  subscriptions: "/subscriptions",
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
  "/watch-later",
  "/playlists",
  "/subscriptions",
  "/notifications",
  "/settings",
  "/admin",
] as const;

export function isProtectedPath(pathname: string): boolean {
  return protectedPaths.some((path) => pathname === path || pathname.startsWith(`${path}/`));
}
