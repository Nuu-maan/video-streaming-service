import { cn } from "@/lib/utils";
import { formatCompact } from "@/lib/format";
import type { DataPoint } from "@/types/common";

interface ViewsSparklineProps {
  points: DataPoint[];
  /** What one bucket covers, for the caption under the chart. */
  intervalLabel: string;
  className?: string;
}

/** The viewBox the path is drawn in. Nothing here is in pixels — see below. */
const W = 600;
const H = 140;
const PAD = 4;

/**
 * Views over time, as an area chart.
 *
 * Inline SVG with no charting library: this is one series of at most a few dozen
 * points, and pulling in a runtime to draw a polyline would cost more than the
 * page it sits on. It renders on the server, so there is no client bundle and no
 * empty box waiting for hydration.
 *
 * `preserveAspectRatio="none"` with a fixed viewBox is what lets the chart
 * stretch to any container width while the geometry below stays in one tidy
 * coordinate space. The stroke is vector-effect'd so it does not stretch with
 * it — without that, a wide container renders a hairline top edge and a fat left
 * one.
 */
export function ViewsSparkline({ points, intervalLabel, className }: ViewsSparklineProps) {
  if (points.length === 0) {
    return (
      <p className={cn("py-10 text-center text-sm text-muted-foreground", className)}>
        No views recorded yet.
      </p>
    );
  }

  const values = points.map((point) => point.value);
  const peak = Math.max(...values, 1);
  const total = values.reduce((sum, value) => sum + value, 0);

  /*
   * A single point has no line to draw, so it is widened into a flat two-point
   * series. Without this the path is a degenerate "M x y" and renders nothing at
   * all — a brand-new video would show an empty chart rather than its one view.
   */
  const plotted = points.length === 1 ? [points[0], points[0]] : points;
  const step = (W - PAD * 2) / (plotted.length - 1);

  const coords = plotted.map((point, index) => {
    const x = PAD + index * step;
    const y = H - PAD - (point.value / peak) * (H - PAD * 2);
    return `${x.toFixed(2)},${y.toFixed(2)}`;
  });

  const line = `M${coords.join("L")}`;
  const area = `${line}L${(W - PAD).toFixed(2)},${H - PAD}L${PAD},${H - PAD}Z`;

  return (
    <figure className={cn("flex flex-col gap-2", className)}>
      <svg
        viewBox={`0 0 ${W} ${H}`}
        preserveAspectRatio="none"
        role="img"
        aria-label={`${formatCompact(total)} views across ${points.length} ${intervalLabel} buckets, peaking at ${formatCompact(peak)}.`}
        className="h-32 w-full"
      >
        <defs>
          <linearGradient id="views-fill" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="var(--color-brand-500)" stopOpacity="0.28" />
            <stop offset="100%" stopColor="var(--color-brand-500)" stopOpacity="0" />
          </linearGradient>
        </defs>

        <path d={area} fill="url(#views-fill)" />
        <path
          d={line}
          fill="none"
          stroke="var(--color-brand-500)"
          strokeWidth="2"
          strokeLinejoin="round"
          strokeLinecap="round"
          vectorEffect="non-scaling-stroke"
        />
      </svg>

      <figcaption className="flex items-baseline justify-between text-xs text-muted-foreground">
        <span>
          Peak <span className="font-medium text-foreground tabular-nums">{formatCompact(peak)}</span> per{" "}
          {intervalLabel}
        </span>
        <span className="tabular-nums">
          {points.length} {intervalLabel}
          {points.length === 1 ? "" : "s"}
        </span>
      </figcaption>
    </figure>
  );
}
