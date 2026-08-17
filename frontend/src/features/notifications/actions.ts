"use server";

import { revalidatePath } from "next/cache";

import { routes } from "@/config/routes";
import { getUnreadCount } from "@/features/notifications/api";
import type {
  ActionFailure,
  MarkAllResult,
  NotificationResult,
} from "@/features/notifications/types";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";

function fail(error: unknown): ActionFailure {
  if (isApiError(error)) {
    if (error.isUnauthorized) {
      return { ok: false, code: "UNAUTHORIZED", message: "Sign in to see your notifications." };
    }
    if (error.isRateLimited) {
      return { ok: false, code: "RATE_LIMITED", message: "Slow down a moment, then try again." };
    }
    return { ok: false, code: error.code, message: error.message };
  }
  return { ok: false, code: "UNKNOWN", message: "Something went wrong. Please try again." };
}

/**
 * The bell's poll. It is a client component with no bearer token, so the count
 * comes back through here. Returns 0 rather than throwing on any failure: a
 * background poll that can take down the header is a bad trade for a badge.
 */
export async function fetchUnreadCount(): Promise<number> {
  try {
    return await getUnreadCount();
  } catch {
    return 0;
  }
}

export async function markNotificationRead(notificationId: string): Promise<NotificationResult> {
  try {
    await api.post(`/me/notifications/${notificationId}/read`);
  } catch (error) {
    // Already read is the state we wanted.
    if (!(isApiError(error) && error.isNotFound)) return fail(error);
  }
  revalidatePath(routes.notifications);
  return { ok: true };
}

export async function markAllNotificationsRead(): Promise<MarkAllResult> {
  try {
    const result = await api.post<{ marked?: number }>("/me/notifications/read-all");
    revalidatePath(routes.notifications);
    return { ok: true, marked: result?.marked ?? 0 };
  } catch (error) {
    return fail(error);
  }
}
