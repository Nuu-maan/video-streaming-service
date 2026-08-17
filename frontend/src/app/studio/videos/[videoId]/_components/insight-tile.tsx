import type { LucideIcon } from "lucide-react";

/**
 * One number about one video.
 *
 * Tiles rather than a chart, for the same reason the admin dashboard uses them:
 * these quantities share no unit — seconds, percentages, counts — and putting
 * them on one axis would invite a comparison that means nothing. The only chart
 * on this page is the one series that genuinely varies over time.
 *
 * Tabular figures throughout, so a column of tiles keeps its digits aligned.
 */
export function InsightTile({
  label,
  value,
  hint,
  icon: Icon,
}: {
  label: string;
  /** Pre-formatted. The tile lays out; the caller decides what "1.2K" means. */
  value: string;
  hint?: string;
  icon: LucideIcon;
}) {
  return (
    <div className="flex items-start gap-3 rounded-xl bg-card p-4 shadow-border">
      <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground ring-1 ring-border/60 ring-inset">
        <Icon aria-hidden className="size-4.5" />
      </div>
      <div className="min-w-0">
        <p className="truncate text-sm text-muted-foreground">{label}</p>
        <p className="mt-0.5 text-2xl font-semibold tabular-nums">{value}</p>
        {hint ? <p className="mt-0.5 truncate text-xs text-muted-foreground">{hint}</p> : null}
      </div>
    </div>
  );
}
