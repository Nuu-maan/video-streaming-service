import { cn } from "@/lib/utils";

interface MeterProps {
  label: string;
  /** A percentage, 0–100. Clamped — a host reporting 103% CPU must not overflow the bar. */
  percent: number;
}

/**
 * A single ratio against a limit: CPU, memory. A meter, not a chart — a two-slice
 * pie of "used" and "free" is the classic way to spend a hundred pixels saying
 * one number.
 *
 * The fill carries severity (fine → warning → critical) and the number is always
 * written out beside it, so the colour is a redundant channel rather than the
 * only one. A red bar nobody can see as red still reads as "91%".
 */
export function Meter({ label, percent }: MeterProps) {
  const value = Math.min(Math.max(percent, 0), 100);

  const tone =
    value >= 90
      ? "bg-destructive"
      : value >= 75
        ? "bg-amber-500"
        : "bg-brand-500";

  return (
    <div>
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-sm text-muted-foreground">{label}</span>
        <span className="text-sm font-medium tabular-nums">{Math.round(value)}%</span>
      </div>
      <div
        role="meter"
        aria-label={label}
        aria-valuenow={Math.round(value)}
        aria-valuemin={0}
        aria-valuemax={100}
        className="mt-1.5 h-2 overflow-hidden rounded-full bg-muted"
      >
        {/* The track is the neutral well and the fill is the value. Width is set
            inline because it is data, not design — there is no Tailwind class for
            "37.4%", and this is a static render, not an animation. */}
        <div className={cn("h-full rounded-full", tone)} style={{ width: `${value}%` }} />
      </div>
    </div>
  );
}
