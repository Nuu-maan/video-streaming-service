import { ArrowLeft, Clock, Eye, Gauge, Heart, MessageSquareText, Users } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import { InsightTile } from "./_components/insight-tile";
import { ErrorState } from "@/components/common/error-state";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { getMyVideoAnalytics, getMyViewsTimeSeries } from "@/features/studio/api";
import { ViewsSparkline } from "@/features/studio/components/views-sparkline";
import { isApiError } from "@/lib/api-error";
import { formatCompact, formatDuration } from "@/lib/format";
import type { TimeSeriesData, VideoAnalytics } from "@/types/common";

export const metadata: Metadata = { title: "Insights" };

/**
 * A creator's own numbers for one video.
 *
 * This page exists because for most of the project's life it could not: the only
 * per-video analytics route lived under /admin and demanded `view_analytics`, a
 * permission no ordinary account holds, so an uploader had no way to see how
 * their own upload was doing. The API now resolves ownership instead.
 *
 * The 404 is load-bearing and deliberately ambiguous. "Not yours" and "not
 * there" are answered identically by the API so that a stranger cannot probe for
 * the existence of a private video, and this page must not undo that by saying
 * "you don't have permission" — it cannot know that, and claiming it would leak
 * exactly what the API refused to.
 */
async function load(videoId: string): Promise<{ analytics: VideoAnalytics; series: TimeSeriesData | null }> {
  try {
    // The series is the softer of the two: a chart that fails should cost the
    // chart, not the page, so it degrades to null while the tiles still render.
    const [analytics, series] = await Promise.all([
      getMyVideoAnalytics(videoId),
      getMyViewsTimeSeries(videoId, "day").catch(() => null),
    ]);
    return { analytics, series };
  } catch (error) {
    if (isApiError(error) && error.isNotFound) notFound();
    throw error;
  }
}

export default async function VideoInsightsPage(props: PageProps<"/studio/videos/[videoId]">) {
  const { videoId } = await props.params;
  const { analytics, series } = await load(videoId);

  const watchPercent = Math.round(analytics.avg_watch_percent);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-3">
        <Button asChild variant="ghost" size="sm" className="-ml-2 self-start text-muted-foreground">
          <Link href={routes.studio}>
            <ArrowLeft aria-hidden />
            All videos
          </Link>
        </Button>

        <div className="flex flex-wrap items-end justify-between gap-4">
          <div className="min-w-0">
            <h1 className="text-title text-balance">{analytics.title}</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              Your numbers for this video. Only you and an administrator can see this page.
            </p>
          </div>
          <Button asChild variant="secondary" size="sm">
            <Link href={routes.video(analytics.video_id)}>Open the video</Link>
          </Button>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <InsightTile label="Views" value={formatCompact(analytics.total_views)} icon={Eye} hint="All time" />
        <InsightTile
          label="Unique viewers"
          value={formatCompact(analytics.unique_viewers)}
          icon={Users}
          hint="Distinct sessions and accounts"
        />
        <InsightTile
          label="Average watched"
          value={`${watchPercent}%`}
          icon={Gauge}
          hint={`About ${formatDuration(analytics.avg_watch_time)} per view`}
        />
        <InsightTile
          label="Total watch time"
          value={formatDuration(analytics.total_watch_time)}
          icon={Clock}
          hint="Everyone, added together"
        />
        <InsightTile
          label="Likes"
          value={formatCompact(analytics.likes)}
          icon={Heart}
          hint={
            analytics.dislikes > 0 ? `${formatCompact(analytics.dislikes)} dislikes` : "No dislikes"
          }
        />
        <InsightTile label="Notes" value={formatCompact(analytics.comments)} icon={MessageSquareText} />
      </div>

      <section aria-labelledby="insights-views" className="rounded-xl bg-card p-4 shadow-border sm:p-5">
        <h2 id="insights-views" className="text-sm font-semibold tracking-tight">
          Views per day
        </h2>
        {series ? (
          <ViewsSparkline points={series.datapoints ?? []} intervalLabel="day" className="mt-4" />
        ) : (
          <ErrorState
            title="The chart didn't load"
            description="The numbers above are still accurate. Reload to try the chart again."
            className="mt-4 min-h-32"
          />
        )}
      </section>

      {/*
       * Playback quality. This is the one section of the page that is genuinely
       * about the pipeline rather than the audience — which renditions people
       * actually received tells a creator whether the ladder is doing its job,
       * and it is data no consumer video site would ever show you.
       */}
      {Object.keys(analytics.views_by_quality ?? {}).length > 0 ? (
        <section aria-labelledby="insights-quality" className="rounded-xl bg-card p-4 shadow-border sm:p-5">
          <h2 id="insights-quality" className="text-sm font-semibold tracking-tight">
            Delivered at
          </h2>
          <ul className="mt-3 flex flex-col gap-2">
            {Object.entries(analytics.views_by_quality)
              .sort(([a], [b]) => Number.parseInt(b, 10) - Number.parseInt(a, 10))
              .map(([quality, count]) => {
                const share = analytics.total_views > 0 ? (count / analytics.total_views) * 100 : 0;
                return (
                  <li key={quality} className="flex items-center gap-3">
                    <span className="w-14 shrink-0 font-mono text-xs tabular-nums">{quality}</span>
                    <span className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                      <span
                        className="block h-full origin-left rounded-full bg-brand-500"
                        style={{ transform: `scaleX(${share / 100})` }}
                      />
                    </span>
                    <span className="w-16 shrink-0 text-right text-xs text-muted-foreground tabular-nums">
                      {formatCompact(count)}
                    </span>
                  </li>
                );
              })}
          </ul>
        </section>
      ) : null}
    </div>
  );
}
