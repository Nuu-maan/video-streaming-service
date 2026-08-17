import { Suspense } from "react";

import { PageHeader } from "@/components/common/page-header";
import {
  FailedVideosSection,
  FailedVideosSectionSkeleton,
  QueueSection,
  QueueSectionSkeleton,
} from "@/features/admin/components/queue-section";

export const metadata = { title: "Queue" };

/**
 * The transcoding pipeline: what is queued, what is draining it, and what broke.
 *
 * Two Suspense boundaries rather than one — the queue/worker pair and the failed
 * list are independent reads, and the failed list goes through the ordinary
 * video listing, which is the slower of the two. Splitting them means the number
 * an on-call admin actually opened this page for (is anything processing?)
 * paints first.
 */
export default function AdminQueuePage() {
  return (
    <>
      <PageHeader
        title="Queue"
        description="What's transcoding, what's waiting, and what needs pushing again."
      />

      <Suspense fallback={<QueueSectionSkeleton />}>
        <QueueSection />
      </Suspense>

      <Suspense fallback={<FailedVideosSectionSkeleton />}>
        <FailedVideosSection />
      </Suspense>
    </>
  );
}
