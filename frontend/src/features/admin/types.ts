import type { components } from "@/types/api";

/**
 * The admin domain's types.
 *
 * Most of what this feature handles — DashboardStats, ContentReport,
 * VideoAnalytics, User, PaginationMeta — already has a name in
 * `@/types/common`, and that is where those come from.
 *
 * Three do not: QueueStats, WorkerInfo and RealtimeMetrics are admin-only
 * shapes that no other feature has ever needed, so `types/common.ts` never
 * named them. They are pulled straight from the generated schema here, and
 * *only* here — this file is the single place in the feature that touches
 * `@/types/api`, so if those three ever graduate to `types/common.ts` there is
 * exactly one import to delete.
 */
type Schemas = components["schemas"];

export type QueueStats = Schemas["QueueStats"];
export type WorkerInfo = Schemas["WorkerInfo"];
export type RealtimeMetrics = Schemas["RealtimeMetrics"];

/** `GET /admin/workers` answers with the list and its length, not a bare array. */
export interface WorkerList {
  workers: WorkerInfo[];
  count: number;
}

/**
 * What a moderator can do with a pending report.
 *
 * `ban_user` is the odd one out: it needs `manage_users` on top of the
 * `moderate_content` that the other three require, and the API checks that in
 * the handler — so a moderator who can dismiss a report may still be refused a
 * ban with a 403. The UI has to be ready for that (see `reviewReport`).
 */
export type ReviewAction = "delete_video" | "ban_user" | "warn_user" | "dismiss";

/**
 * Every admin mutation answers with this instead of throwing.
 *
 * A moderation queue is a place where things go wrong routinely — a video is
 * already deleted, a ban needs a permission you do not have, the rate limiter
 * trips. None of that deserves to unwind the page into an error boundary and
 * lose the moderator's place in the queue; all of it deserves a toast. So the
 * actions catch, translate, and return.
 */
export type ActionResult = { ok: true; message: string } | { ok: false; message: string };
