import { Clock } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { Pagination } from "@/components/common/pagination";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { toVideoCard } from "@/features/videos/card-data";
import { listWatchLater } from "@/features/watch-later/api";
import { WatchLaterGrid } from "@/features/watch-later/components/watch-later-grid";
import { isApiError } from "@/lib/api-error";

export const metadata: Metadata = { title: "Watch later" };

function toPage(value: string | string[] | undefined): number {
  const parsed = Number(Array.isArray(value) ? value[0] : value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 1;
}

export default async function WatchLaterPage(props: PageProps<"/watch-later">) {
  const searchParams = await props.searchParams;
  const page = toPage(searchParams.page);

  const result = await listWatchLater({ page, limit: 24 }).catch((error: unknown) => {
    if (isApiError(error) && error.isRateLimited) return "rate-limited" as const;
    return "failed" as const;
  });

  return (
    <>
      <PageHeader
        title="Watch later"
        description="Videos you saved for a quieter moment. Most recently saved first."
      />

      {result === "rate-limited" ? (
        <ErrorState
          title="Slow down a moment"
          description="You're loading pages faster than we can serve them. Try again shortly."
        />
      ) : result === "failed" ? (
        <ErrorState title="Your list didn't load" description="Refresh the page to try again." />
      ) : result.items.length === 0 ? (
        <EmptyState
          icon={Clock}
          title="Nothing saved yet"
          description="Hit Save on any video and it lands here, ready for when you have the time."
          action={
            <Button asChild size="sm" variant="secondary">
              <Link href={routes.home}>Browse videos</Link>
            </Button>
          }
        />
      ) : (
        <>
          <WatchLaterGrid videos={result.items.map((item) => toVideoCard(item.video))} />
          <Pagination pagination={result.pagination} />
        </>
      )}
    </>
  );
}
