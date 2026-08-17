import { ErrorState } from "@/components/common/error-state";
import { Skeleton } from "@/components/ui/skeleton";
import { getRealtime } from "@/features/admin/api";
import { Panel } from "@/features/admin/components/panel";
import { RealtimeStrip } from "@/features/admin/components/realtime-strip";
import { errorCopy } from "@/features/admin/error-copy";

/** Fetches the live counters; the strip renders them. Suspended by the page. */
export async function RealtimeSection() {
  let metrics;
  try {
    metrics = await getRealtime();
  } catch (error) {
    return <ErrorState {...errorCopy(error, "live metrics", "view_analytics")} />;
  }

  return <RealtimeStrip metrics={metrics} />;
}

export function RealtimeSectionSkeleton() {
  return (
    <Panel title="Right now" description="Live counters, measured server-side and never cached.">
      <div className="grid gap-x-8 gap-y-6 sm:grid-cols-2 lg:grid-cols-5">
        {Array.from({ length: 5 }, (_, index) => (
          <div key={index} className="space-y-2">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-7 w-14" />
          </div>
        ))}
      </div>
    </Panel>
  );
}
