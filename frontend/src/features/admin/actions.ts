"use server";

import { revalidatePath } from "next/cache";

import { routes } from "@/config/routes";
import { getCurrentUser } from "@/features/auth/current-user";
import type { ActionResult, ReviewAction } from "@/features/admin/types";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";

/**
 * The admin mutations.
 *
 * Every one of these re-checks the role. The `/admin` layout already does, but
 * a Server Action is a public POST endpoint that does not route through the
 * layout at all — anyone who can guess its id can invoke it. The API is the
 * real gate (it enforces `moderate_content` / `manage_users` and 403s), so this
 * is defence in depth rather than the only lock, but a frontend that hands a
 * logged-out visitor's request to the API and lets the API say no is a frontend
 * that leaks the API's error messages to strangers.
 */
async function requireAdmin(): Promise<boolean> {
  const user = await getCurrentUser();
  return user?.role === "admin";
}

const DENIED: ActionResult = { ok: false, message: "You don't have access to that." };

/**
 * Turns a thrown ApiError into the sentence a moderator should read.
 *
 * `manageUsers` exists because the API overloads 403: on a review it can mean
 * "you can't moderate at all" *or* "you can moderate but you can't ban", and
 * the caller knows which one it just attempted. A moderator refused a ban needs
 * to be told it was the ban specifically — otherwise they retry the same click
 * forever.
 */
function explain(error: unknown, options: { forbidden: string; notFound: string }): ActionResult {
  if (!isApiError(error)) {
    return { ok: false, message: "Couldn't reach the server." };
  }
  if (error.isRateLimited) {
    return { ok: false, message: "Slow down — too many requests. Try again in a moment." };
  }
  if (error.isForbidden) {
    return { ok: false, message: options.forbidden };
  }
  if (error.isNotFound) {
    return { ok: false, message: options.notFound };
  }
  // A 400 from these endpoints is specific and useful ("video is not failed",
  // "cannot ban yourself", "invalid duration"). Passing the API's own sentence
  // through beats inventing a vaguer one.
  return { ok: false, message: error.message };
}

/** Refreshes every admin screen whose numbers an action just changed. */
function revalidateAdmin(): void {
  revalidatePath(routes.admin);
  revalidatePath(routes.adminReports);
  revalidatePath(routes.adminQueue);
}

/**
 * Resolves a pending report.
 *
 * `ban_user` needs `manage_users` on top of `moderate_content`, and the API
 * checks that inside the handler — so this is the one action that can 403 for a
 * moderator who is otherwise perfectly entitled to be here.
 */
export async function reviewReport(
  id: string,
  action: ReviewAction,
  notes?: string,
): Promise<ActionResult> {
  if (!(await requireAdmin())) return DENIED;

  try {
    await api.post(`/admin/reports/${id}/review`, {
      body: { action, notes: notes?.trim() || undefined },
    });
  } catch (error) {
    return explain(error, {
      forbidden:
        action === "ban_user"
          ? "You can't ban users — that needs the manage_users permission. Try another action, or ask an admin who has it."
          : "You don't have permission to moderate content.",
      notFound: "That report is gone — someone else may have reviewed it already.",
    });
  }

  revalidateAdmin();
  return { ok: true, message: RESOLVED[action] };
}

const RESOLVED: Record<ReviewAction, string> = {
  dismiss: "Report dismissed. The content is untouched.",
  warn_user: "User warned and the report closed.",
  delete_video: "Video deleted and the report closed.",
  ban_user: "User banned and the report closed.",
};

/**
 * Bans an account. An empty duration is a permanent ban — the API's own
 * convention, not an oversight.
 */
export async function banUser(id: string, reason: string, duration?: string): Promise<ActionResult> {
  if (!(await requireAdmin())) return DENIED;

  try {
    await api.post(`/admin/users/${id}/ban`, {
      body: { reason: reason.trim(), duration: duration?.trim() || undefined },
    });
  } catch (error) {
    return explain(error, {
      forbidden: "You can't ban users — that needs the manage_users permission.",
      notFound: "No user with that ID.",
    });
  }

  revalidateAdmin();
  return {
    ok: true,
    message: duration?.trim() ? `User banned for ${duration.trim()}.` : "User banned permanently.",
  };
}

export async function unbanUser(id: string): Promise<ActionResult> {
  if (!(await requireAdmin())) return DENIED;

  try {
    await api.post(`/admin/users/${id}/unban`);
  } catch (error) {
    return explain(error, {
      forbidden: "You can't lift bans — that needs the manage_users permission.",
      notFound: "No user with that ID.",
    });
  }

  revalidateAdmin();
  return { ok: true, message: "Ban lifted. The account can sign in again." };
}

/**
 * Re-queues a failed transcode. Only a video in status `failed` may be retried;
 * anything else is a 400, and the API's own message says so.
 */
export async function retryVideo(id: string): Promise<ActionResult> {
  if (!(await requireAdmin())) return DENIED;

  try {
    await api.post(`/admin/videos/${id}/retry`);
  } catch (error) {
    return explain(error, {
      forbidden: "You don't have permission to retry transcodes.",
      notFound: "No video with that ID.",
    });
  }

  revalidateAdmin();
  return { ok: true, message: "Transcode re-queued. It'll pick up on the next free worker." };
}
