import { UsersRound } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { Pagination } from "@/components/common/pagination";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { getCurrentUser } from "@/features/auth/current-user";
import { listMySubscriptions } from "@/features/subscriptions/api";
import { SubscriptionCard } from "@/features/subscriptions/components/subscription-card";
import { isApiError } from "@/lib/api-error";

export const metadata: Metadata = { title: "Subscriptions" };

function toPage(value: string | string[] | undefined): number {
  const parsed = Number(Array.isArray(value) ? value[0] : value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 1;
}

export default async function SubscriptionsPage(props: PageProps<"/subscriptions">) {
  const searchParams = await props.searchParams;
  const page = toPage(searchParams.page);

  const [user, result] = await Promise.all([
    getCurrentUser(),
    listMySubscriptions({ page, limit: 24 }).catch((error: unknown) => {
      if (isApiError(error) && error.isRateLimited) return "rate-limited" as const;
      return "failed" as const;
    }),
  ]);

  // The layout already redirected an anonymous visitor; this satisfies the type
  // and covers the sliver where the session expires between the two calls.
  if (!user) redirect(routes.login);

  return (
    <>
      <PageHeader
        title="Subscriptions"
        description="Creators you follow. New uploads from them show up on your home feed."
      />

      {result === "rate-limited" ? (
        <ErrorState
          title="Slow down a moment"
          description="You're loading pages faster than we can serve them. Try again shortly."
        />
      ) : result === "failed" ? (
        <ErrorState
          title="Your subscriptions didn't load"
          description="Refresh the page to try again."
        />
      ) : result.items.length === 0 ? (
        <EmptyState
          icon={UsersRound}
          title="You aren't following anyone yet"
          description="Subscribe to a creator and their new videos will find you instead of the other way round."
          action={
            <Button asChild size="sm" variant="secondary">
              <Link href={routes.home}>Find creators</Link>
            </Button>
          }
        />
      ) : (
        <>
          <ul className="flex flex-col gap-1">
            {result.items.map((entry) => (
              <SubscriptionCard key={entry.user_id} entry={entry} viewerId={user.id} />
            ))}
          </ul>
          <Pagination pagination={result.pagination} />
        </>
      )}
    </>
  );
}
