import Link from "next/link";

import { routes } from "@/config/routes";
import { formatCompact } from "@/lib/format";
import type { VideoAnalytics } from "@/types/common";

interface TopVideosChartProps {
  videos: VideoAnalytics[];
}

/**
 * The most-watched videos of the past week, as a horizontal bar chart.
 *
 * This is the one honest chart on the dashboard. The six headline stats are a
 * KPI row, because plotting "total users" beside "total views" puts two
 * quantities with no shared unit on one axis and invites a comparison that
 * means nothing. *This* data has a single unit and a single job — compare
 * magnitude across entities — so it gets bars, and bars in one hue: the series
 * is not an identity, it is a quantity, and a rainbow would imply otherwise.
 *
 * Hand-rolled SVG, no charting library. Six-to-ten bars do not justify shipping
 * a runtime to a page that already knows every value at render time.
 *
 * Geometry note: the `<svg>` has no `viewBox` on purpose. Without one, user
 * units are CSS pixels and nothing is scaled, so the bar's `rx` corners stay
 * circular at any container width — a `viewBox` plus
 * `preserveAspectRatio="none"` would stretch them into ellipses. The bar's own
 * width is a percentage, which SVG resolves against the element, so the chart
 * is fluid without a single line of layout maths.
 */
export function TopVideosChart({ videos }: TopVideosChartProps) {
  // The scale is anchored to the largest bar, not to zero-to-total: the reader
  // is comparing these ten against each other, and a `max` of 1 keeps a set of
  // all-zero views from dividing by zero.
  const max = Math.max(...videos.map((video) => video.total_views), 1);

  return (
    <ol className="flex flex-col gap-3">
      {videos.map((video) => {
        const percent = (video.total_views / max) * 100;

        return (
          <li key={video.video_id} className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-1">
            <Link
              href={routes.video(video.video_id)}
              className="min-w-0 truncate text-sm font-medium outline-none hover:underline focus-visible:ring-3 focus-visible:ring-ring/50 rounded-sm"
              title={video.title}
            >
              {video.title}
            </Link>

            {/* The value is written out, so the bar is never the only channel
                carrying the number — colour and length are the redundant,
                scannable encoding, not the sole one. */}
            <span className="text-sm font-medium tabular-nums text-muted-foreground">
              {formatCompact(video.total_views)}
            </span>

            <svg
              width="100%"
              height="8"
              className="col-span-2 overflow-visible"
              role="img"
              aria-label={`${video.total_views.toLocaleString("en")} views`}
            >
              {/* The track is a lighter step of the bar's own hue, so an empty
                  bar still reads as "this metric", not as a stray divider. */}
              <rect x="0" y="0" width="100%" height="8" rx="4" className="fill-brand-500/12" />
              <rect
                x="0"
                y="0"
                width={`${percent}%`}
                height="8"
                rx="4"
                className="fill-brand-500"
              />
            </svg>
          </li>
        );
      })}
    </ol>
  );
}
