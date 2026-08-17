import { BellOff } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { Pagination } from "@/components/common/pagination";
import { Button } from "@/components/ui/button";
import { groupByDay, listNotifications } from "@/features/notifications/api";
import { MarkAllReadButton } from "@/features/notifications/components/mark-all-read-button";
import { NotificationList } from "@/features/notifications/components/notification-list";
import { isApiError } from "@/lib/api-error";
import { cn } from "@/lib/utils";

export const metadata: Metadata = { title: "Notifications" };

function toPage(value: string | string[] | undefined): number {
  const parsed = Number(Array.isArray(value) ? value[0] : value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 1;
}

export default async function NotificationsPage(props: PageProps<"/notifications">) {
  const searchParams = await props.searchParams;
  const page = toPage(searchParams.page);
  const unreadOnly = searchParams.unread === "true";

  const result = await listNotifications({ page, limit: 30, unread: unreadOnly }).catch(
    (error: unknown) => {
      if (isApiError(error) && error.isRateLimited) return "rate-limited" as const;
      return "failed" as const;
    },
  );

  const hasItems = typeof result !== "string" && result.items.length > 0;

  return (
    <>
      <PageHeader
        title="Notifications"
        description="Replies, likes, new subscribers, and uploads from creators you follow."
        actions={<MarkAllReadButton disabled={!hasItems} />}
      />

      {/* Two links, not a client-side tab widget: the filter is a URL, so it is
          shareable, back-button-able, and works before any JavaScript loads. */}
      <nav aria-label="Filter notifications" className="-mt-4 flex gap-1">
        {[
          { label: "All", href: "/notifications", active: !unreadOnly },
          { label: "Unread", href: "/notifications?unread=true", active: unreadOnly },
        ].map((tab) => (
          <Link
            key={tab.label}
            href={tab.href}
            aria-current={tab.active ? "page" : undefined}
            className={cn(
              "rounded-full px-3 py-1.5 text-sm font-medium outline-none transition-colors duration-(--motion-fast) focus-visible:ring-3 focus-visible:ring-ring/50",
              tab.active
                ? "bg-secondary text-secondary-foreground"
                : "text-muted-foreground hover:bg-muted/70 hover:text-foreground",
            )}
          >
            {tab.label}
          </Link>
        ))}
      </nav>

      {result === "rate-limited" ? (
        <ErrorState
          title="Slow down a moment"
          description="You're loading pages faster than we can serve them. Try again shortly."
        />
      ) : result === "failed" ? (
        <ErrorState
          title="Notifications didn't load"
          description="Refresh the page to try again."
        />
      ) : result.items.length === 0 ? (
        <EmptyState
          icon={BellOff}
          title={unreadOnly ? "Nothing unread" : "No notifications yet"}
          description={
            unreadOnly
              ? "You're all caught up."
              : "When someone replies to you, subscribes, or a creator you follow uploads, you'll hear about it here."
          }
          action={
            unreadOnly ? (
              <Button asChild size="sm" variant="secondary">
                <Link href="/notifications">See all notifications</Link>
              </Button>
            ) : undefined
          }
        />
      ) : (
        <>
          <NotificationList days={groupByDay(result.items)} />
          <Pagination pagination={result.pagination} />
        </>
      )}
    </>
  );
}
