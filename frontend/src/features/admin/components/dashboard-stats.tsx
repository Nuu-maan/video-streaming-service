import { Clock, Eye, HardDrive, TriangleAlert, Users, Video } from "lucide-react";

import { ErrorState } from "@/components/common/error-state";
import { Skeleton } from "@/components/ui/skeleton";
import { getDashboard } from "@/features/admin/api";
import { StatCard } from "@/features/admin/components/stat-card";
import { errorCopy } from "@/features/admin/error-copy";
import { formatBytes, formatCompact } from "@/lib/format";

/**
 * The six numbers that describe the platform.
 *
 * Each fetches-and-renders itself rather than being handed data by the page, so
 * the page can wrap it in its own Suspense boundary: the dashboard, the live
 * counters and the top-videos chart are three independent API calls, and one of
 * them being slow should not hold the other two hostage behind a blank screen.
 *
 * The two "something is wrong" tiles are conditional on there actually being
 * something wrong. A `failed_videos` of zero is the *good* outcome, and painting
 * it red every day trains the reader to ignore the colour on the day it matters.
 */
export async function DashboardStats() {
  let stats;
  try {
    stats = await getDashboard();
  } catch (error) {
    return <ErrorState {...errorCopy(error, "the dashboard", "view_analytics")} />;
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <StatCard
        icon={Users}
        label="Users"
        value={formatCompact(stats.total_users)}
        hint={`${formatCompact(stats.new_users_today)} joined today · ${formatCompact(stats.active_users_24h)} active in 24h`}
      />
      <StatCard
        icon={Video}
        label="Videos"
        value={formatCompact(stats.total_videos)}
        hint={`${formatCompact(stats.videos_today)} uploaded today`}
      />
      <StatCard
        icon={Eye}
        label="Views"
        value={formatCompact(stats.total_views)}
        hint={`${formatCompact(stats.views_today)} today`}
      />
      <StatCard
        icon={HardDrive}
        label="Storage"
        value={formatBytes(stats.total_storage_bytes)}
        hint="Across every rendition and thumbnail"
      />
      <StatCard
        icon={Clock}
        label="Processing"
        value={formatCompact(stats.processing_videos)}
        hint={stats.processing_videos > 0 ? "Videos transcoding right now" : "Nothing in flight"}
        tone={stats.processing_videos > 0 ? "warning" : "default"}
      />
      <StatCard
        icon={TriangleAlert}
        label="Failed"
        value={formatCompact(stats.failed_videos)}
        hint={stats.failed_videos > 0 ? "Retry them from the queue" : "No failed transcodes"}
        tone={stats.failed_videos > 0 ? "danger" : "default"}
      />
    </div>
  );
}

/** Matches the grid above, tile for tile, so the layout does not jump on load. */
export function DashboardStatsSkeleton() {
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {Array.from({ length: 6 }, (_, index) => (
        <div key={index} className="flex items-start gap-4 rounded-xl bg-card p-4 shadow-border">
          <Skeleton className="size-9 shrink-0 rounded-lg" />
          <div className="w-full space-y-2">
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-7 w-16" />
            <Skeleton className="h-3 w-28" />
          </div>
        </div>
      ))}
    </div>
  );
}
