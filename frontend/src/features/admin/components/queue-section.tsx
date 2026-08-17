import { ErrorState } from "@/components/common/error-state";
import { Skeleton } from "@/components/ui/skeleton";
import { getFailedVideos, getQueueStats, getWorkers } from "@/features/admin/api";
import { FailedVideosPanel } from "@/features/admin/components/failed-videos-panel";
import { Panel } from "@/features/admin/components/panel";
import { QueueStatsGrid } from "@/features/admin/components/queue-stats-grid";
import { WorkerTable } from "@/features/admin/components/worker-table";
import { errorCopy } from "@/features/admin/error-copy";

/**
 * Queue depth and the workers draining it.
 *
 * These two are fetched together — `Promise.all`, not sequential awaits —
 * because they are read together: "eight hundred pending" means one thing with
 * six workers up and something very different with none. Awaiting them one after
 * the other would double the latency of a panel that is useless half-rendered.
 */
export async function QueueSection() {
  let stats;
  let workers;
  try {
    [stats, workers] = await Promise.all([getQueueStats(), getWorkers()]);
  } catch (error) {
    return <ErrorState {...errorCopy(error, "the queue", "moderate_content")} />;
  }

  return (
    <>
      <QueueStatsGrid stats={stats} />
      <WorkerTable workers={workers.workers} />
    </>
  );
}

export function QueueSectionSkeleton() {
  return (
    <>
      <Panel title="Queue" description="The transcoding queue, task by task.">
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
          {Array.from({ length: 6 }, (_, index) => (
            <div key={index} className="space-y-2">
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-7 w-10" />
            </div>
          ))}
        </div>
      </Panel>
      <Panel title="Workers" description="The machines pulling jobs off the queue.">
        <div className="space-y-3">
          {Array.from({ length: 3 }, (_, index) => (
            <Skeleton key={index} className="h-8 w-full" />
          ))}
        </div>
      </Panel>
    </>
  );
}

/** Separate boundary from the queue itself: it is a different endpoint, and a slower one. */
export async function FailedVideosSection() {
  let failed;
  try {
    failed = await getFailedVideos({ limit: 20 });
  } catch (error) {
    return <ErrorState {...errorCopy(error, "failed transcodes")} />;
  }

  return <FailedVideosPanel videos={failed.items} />;
}

export function FailedVideosSectionSkeleton() {
  return (
    <Panel
      title="Failed transcodes"
      description="Re-queue a video whose transcode failed. Only a video in status failed can be retried."
    >
      <div className="space-y-3">
        {Array.from({ length: 3 }, (_, index) => (
          <Skeleton key={index} className="h-10 w-full" />
        ))}
      </div>
    </Panel>
  );
}
