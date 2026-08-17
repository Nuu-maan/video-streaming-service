import { Suspense } from "react";

import { PageHeader } from "@/components/common/page-header";
import {
  DashboardStats,
  DashboardStatsSkeleton,
} from "@/features/admin/components/dashboard-stats";
import {
  RealtimeSection,
  RealtimeSectionSkeleton,
} from "@/features/admin/components/realtime-section";
import {
  TopVideosSection,
  TopVideosSectionSkeleton,
} from "@/features/admin/components/top-videos-section";

export const metadata = { title: "Overview" };

/**
 * The dashboard.
 *
 * Three independent API calls, three Suspense boundaries. They are deliberately
 * *not* awaited together in the page body: doing so would make the whole screen
 * wait for the slowest of them, and the realtime endpoint (which the API never
 * caches) is reliably the slowest. As separate boundaries they stream in as they
 * land, each behind a skeleton shaped like the thing it is replacing.
 *
 * Each section owns its own failure, too. A missing `view_analytics` permission
 * takes out the numbers but not the page, and the moderation queue in the next
 * tab still works.
 */
export default function AdminDashboardPage() {
  return (
    <>
      <PageHeader
        title="Overview"
        description="The platform at a glance — what's on it, who's watching, and what's stuck."
      />

      <Suspense fallback={<DashboardStatsSkeleton />}>
        <DashboardStats />
      </Suspense>

      <Suspense fallback={<RealtimeSectionSkeleton />}>
        <RealtimeSection />
      </Suspense>

      <Suspense fallback={<TopVideosSectionSkeleton />}>
        <TopVideosSection />
      </Suspense>
    </>
  );
}
