import { ChartNoAxesColumn } from "lucide-react";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { Skeleton } from "@/components/ui/skeleton";
import { getTopVideos } from "@/features/admin/api";
import { Panel } from "@/features/admin/components/panel";
import { TopVideosChart } from "@/features/admin/components/top-videos-chart";
import { errorCopy } from "@/features/admin/error-copy";

export async function TopVideosSection() {
  let videos;
  try {
    videos = await getTopVideos(10);
  } catch (error) {
    return <ErrorState {...errorCopy(error, "top videos", "view_analytics")} />;
  }

  return (
    <Panel title="Most watched" description="The ten most-viewed videos of the past week.">
      {videos.length === 0 ? (
        <EmptyState
          icon={ChartNoAxesColumn}
          title="No views yet this week"
          description="Once people start watching, the week's most-viewed videos show up here."
          className="min-h-48 border-0"
        />
      ) : (
        <TopVideosChart videos={videos} />
      )}
    </Panel>
  );
}

export function TopVideosSectionSkeleton() {
  return (
    <Panel title="Most watched" description="The ten most-viewed videos of the past week.">
      <div className="flex flex-col gap-3">
        {Array.from({ length: 6 }, (_, index) => (
          <div key={index} className="space-y-1.5">
            <Skeleton className="h-4 w-1/3" />
            <Skeleton className="h-2 w-full rounded-full" />
          </div>
        ))}
      </div>
    </Panel>
  );
}
