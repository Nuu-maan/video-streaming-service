import { History } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { Pagination } from "@/components/common/pagination";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { HISTORY_PAGE_SIZE, listHistory } from "@/features/history/api";
import { ClearHistoryButton } from "@/features/history/components/clear-history-button";
import { HistoryList } from "@/features/history/components/history-list";
import type { HistoryRow } from "@/features/history/types";
import { isApiError } from "@/lib/api-error";
import type { Page } from "@/types/common";

export const metadata: Metadata = {
  title: "Watch history",
  // Someone's viewing history is nobody else's business, least of all a crawler's.
  robots: { index: false, follow: false },
};

function toPage(value: string | string[] | undefined): number {
  const parsed = Number(Array.isArray(value) ? value[0] : value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 1;
}

/**
 * A discriminated result, so no JSX is ever constructed inside the `catch`. React
 * does not render a component when its element is constructed, so a try/catch
 * wrapped around JSX never catches a render error — it only reads as though it
 * does. The markup is selected outside.
 */
type Result = { ok: true; page: Page<HistoryRow> } | { ok: false; reason: "rate-limited" | "failed" };

async function load(page: number): Promise<Result> {
  try {
    return { ok: true, page: await listHistory({ page, limit: HISTORY_PAGE_SIZE }) };
  } catch (error) {
    if (isApiError(error) && error.isRateLimited) return { ok: false, reason: "rate-limited" };
    return { ok: false, reason: "failed" };
  }
}

export default async function HistoryPage(props: PageProps<"/history">) {
  const searchParams = await props.searchParams;
  const page = toPage(searchParams.page);
  const result = await load(page);

  const hasRows = result.ok && result.page.items.length > 0;

  return (
    <>
      <PageHeader
        title="Watch history"
        description="Everything you've watched, most recent first. Pick up where you left off."
        /* Nothing to clear when there is nothing there — and offering a
           destructive action against an empty list is just a trap. */
        actions={hasRows ? <ClearHistoryButton /> : undefined}
      />

      {!result.ok ? (
        result.reason === "rate-limited" ? (
          <ErrorState
            title="Slow down a moment"
            description="You're loading pages faster than we can serve them. Try again in a few seconds."
          />
        ) : (
          <ErrorState
            title="Your history didn't load"
            description="Something went wrong on our side. Refresh the page to try again."
          />
        )
      ) : result.page.items.length === 0 ? (
        page > 1 ? (
          <EmptyState
            icon={History}
            title="Nothing on this page"
            description="You've reached the end of your history."
            action={
              <Button asChild size="sm" variant="secondary">
                <Link href={routes.history}>Back to the first page</Link>
              </Button>
            }
          />
        ) : (
          <EmptyState
            icon={History}
            title="No history yet"
            description="Videos you watch show up here, with a marker for how far you got — so you can pick one back up without hunting for it."
            action={
              <Button asChild size="sm" variant="secondary">
                <Link href={routes.home}>Find something to watch</Link>
              </Button>
            }
          />
        )
      ) : (
        <>
          <HistoryList rows={result.page.items} />
          <Pagination pagination={result.page.pagination} />
        </>
      )}
    </>
  );
}
