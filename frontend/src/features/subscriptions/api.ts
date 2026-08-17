import "server-only";

import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";
import type { Page, PageParams, SubscriptionEntry } from "@/types/common";

/** Creators the caller follows. */
export async function listMySubscriptions(params: PageParams = {}): Promise<Page<SubscriptionEntry>> {
  return api.page<SubscriptionEntry>("/me/subscriptions", {
    query: { page: params.page, limit: params.limit ?? 24 },
  });
}

/** A creator's subscribers. Public — no session required. */
export async function listSubscribers(
  userId: string,
  params: PageParams = {},
): Promise<Page<SubscriptionEntry>> {
  return api.page<SubscriptionEntry>(`/users/${userId}/subscribers`, {
    query: { page: params.page, limit: params.limit ?? 24 },
  });
}

/**
 * Whether the caller subscribes to a creator.
 *
 * The API has no single-subscription lookup, so this walks `/me/subscriptions`.
 * The scan is bounded: subscribe/unsubscribe are both idempotent, so the worst
 * case for someone following more than 500 creators is a button that opens on
 * "Subscribe" and quietly re-subscribes them. Cheap to be wrong, expensive to
 * be exhaustive.
 *
 * The pages after the first are fetched TOGETHER. They were a strictly
 * sequential `for` loop — five round trips stacked end to end, on a component
 * that was itself already sitting behind two other serial awaits. The first page
 * has to come back before we know how many others there are, but the others do
 * not depend on each other at all, so they cost one round trip between them
 * rather than four. Anybody following 100 creators or fewer — which is very
 * nearly everybody — pays for exactly one request.
 */
const SCAN_LIMIT = 100;
const SCAN_MAX_PAGES = 5;

export async function isSubscribedTo(userId: string): Promise<boolean> {
  const holds = (page: Page<SubscriptionEntry>) =>
    page.items.some((entry) => entry.user_id === userId);

  try {
    const first = await listMySubscriptions({ page: 1, limit: SCAN_LIMIT });
    if (holds(first)) return true;

    const lastPage = Math.min(first.pagination.total_pages, SCAN_MAX_PAGES);
    if (lastPage <= 1) return false;

    const rest = await Promise.all(
      Array.from({ length: lastPage - 1 }, (_, index) =>
        listMySubscriptions({ page: index + 2, limit: SCAN_LIMIT }),
      ),
    );
    return rest.some(holds);
  } catch (error) {
    if (isApiError(error) && error.isUnauthorized) return false;
    throw error;
  }
}
