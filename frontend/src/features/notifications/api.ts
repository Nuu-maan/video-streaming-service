import "server-only";

import type { NotificationDay } from "@/features/notifications/types";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";
import { formatDate } from "@/lib/format";
import type { Notification, Page, PageParams } from "@/types/common";

interface ListParams extends PageParams {
  /** Narrows to unread. The API takes the literal string "true". */
  unread?: boolean;
}

export async function listNotifications(params: ListParams = {}): Promise<Page<Notification>> {
  return api.page<Notification>("/me/notifications", {
    query: {
      unread: params.unread ? "true" : undefined,
      page: params.page,
      limit: params.limit ?? 30,
    },
  });
}

/**
 * The badge count. Signing out is not a failure — it is zero — and neither is a
 * hiccup on a background poll: a bell that throws would take the header with it.
 */
export async function getUnreadCount(): Promise<number> {
  try {
    const result = await api.get<{ unread_count?: number }>("/me/notifications/unread-count");
    return result.unread_count ?? 0;
  } catch (error) {
    if (isApiError(error) && error.isUnauthorized) return 0;
    throw error;
  }
}

/**
 * Groups notifications under day headings.
 *
 * Runs on the server, once, so the labels are computed against one clock — a
 * client that recomputed them at midnight would disagree with the HTML it was
 * hydrating. The API already returns them newest first, so insertion order is
 * the display order and no sort is needed.
 */
export function groupByDay(notifications: Notification[], now: Date = new Date()): NotificationDay[] {
  const dayKey = (date: Date) => date.toISOString().slice(0, 10);
  const today = dayKey(now);
  const yesterday = dayKey(new Date(now.getTime() - 24 * 60 * 60 * 1000));

  const days = new Map<string, Notification[]>();
  for (const notification of notifications) {
    const key = dayKey(new Date(notification.created_at));
    const bucket = days.get(key);
    if (bucket) bucket.push(notification);
    else days.set(key, [notification]);
  }

  return [...days.entries()].map(([key, items]) => ({
    key,
    label: key === today ? "Today" : key === yesterday ? "Yesterday" : formatDate(items[0].created_at),
    items,
  }));
}
