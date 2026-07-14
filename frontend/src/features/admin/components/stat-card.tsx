import type { LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";

/**
 * `warning` and `danger` are states, not decoration. They are reserved for the
 * two tiles that describe a platform in trouble — videos stuck processing, and
 * videos that failed outright — and they never fire when the number is zero,
 * because a zero failure count is the good news, not the bad.
 */
type Tone = "default" | "warning" | "danger";

interface StatCardProps {
  label: string;
  /** Pre-formatted. The card lays out; the caller decides what "1.2M" means. */
  value: string;
  /** The unit or period the value is measured in — "all time", "GB", "today". */
  hint?: string;
  icon: LucideIcon;
  tone?: Tone;
}

const TONE: Record<Tone, { icon: string; value: string }> = {
  default: { icon: "bg-muted text-muted-foreground ring-border/60", value: "text-foreground" },
  warning: { icon: "bg-amber-500/10 text-amber-600 ring-amber-500/20 dark:text-amber-400", value: "text-foreground" },
  danger: { icon: "bg-destructive/10 text-destructive ring-destructive/20", value: "text-destructive" },
};

/**
 * One headline number.
 *
 * Six numbers is a KPI row, not a chart — a bar chart of "total users" against
 * "total views" would put two quantities that share no unit on one axis and
 * invite a comparison that means nothing. Tiles state each number plainly and
 * let the reader do the only comparison that is real: this number against what
 * they expected it to be.
 *
 * The digits are tabular because several of these tick (the realtime strip
 * re-renders them), and a proportional `1` is narrower than a `0` — without
 * tabular figures the value visibly reflows every time a counter advances.
 */
export function StatCard({ label, value, hint, icon: Icon, tone = "default" }: StatCardProps) {
  const styles = TONE[tone];

  return (
    <div className="flex items-start gap-4 rounded-xl bg-card p-4 shadow-border">
      <div
        className={cn(
          "flex size-9 shrink-0 items-center justify-center rounded-lg ring-1 ring-inset",
          styles.icon,
        )}
      >
        <Icon aria-hidden className="size-4.5" />
      </div>

      <div className="min-w-0">
        <p className="truncate text-sm text-muted-foreground">{label}</p>
        <p className={cn("mt-0.5 text-2xl font-semibold tabular-nums", styles.value)}>{value}</p>
        {hint ? <p className="mt-0.5 truncate text-xs text-muted-foreground">{hint}</p> : null}
      </div>
    </div>
  );
}
