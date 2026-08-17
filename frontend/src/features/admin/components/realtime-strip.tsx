import { Eye, Radio, Upload } from "lucide-react";

import { Meter } from "@/features/admin/components/meter";
import { Panel } from "@/features/admin/components/panel";
import type { RealtimeMetrics } from "@/features/admin/types";
import { formatCompact } from "@/lib/format";

interface RealtimeStripProps {
  metrics: RealtimeMetrics;
}

/**
 * The "right now" panel: who is watching, what is uploading, how hard the boxes
 * are breathing.
 *
 * The API never caches this endpoint and neither does the page — a stale live
 * counter is worse than no live counter, because it is confidently wrong. The
 * timestamp is rendered as a `<time>` so it is machine-readable, and it is the
 * *server's* timestamp, not the browser's clock: the two disagree, and the one
 * that matters is when the API measured this.
 */
export function RealtimeStrip({ metrics }: RealtimeStripProps) {
  const counters = [
    { label: "Watching now", value: metrics.active_viewers, icon: Radio },
    { label: "Views, last hour", value: metrics.views_last_hour, icon: Eye },
    { label: "Uploads, last hour", value: metrics.uploads_last_hour, icon: Upload },
  ];

  return (
    <Panel
      title="Right now"
      description="Live counters, measured server-side and never cached."
      aside={
        <time
          dateTime={metrics.timestamp}
          className="text-xs text-muted-foreground"
          title={metrics.timestamp}
        >
          {new Date(metrics.timestamp).toLocaleTimeString("en", {
            hour: "2-digit",
            minute: "2-digit",
            second: "2-digit",
          })}
        </time>
      }
    >
      <div className="grid gap-x-8 gap-y-6 sm:grid-cols-2 lg:grid-cols-[repeat(3,minmax(0,1fr))_1px_repeat(2,minmax(0,12rem))]">
        {counters.map(({ label, value, icon: Icon }) => (
          <div key={label}>
            <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
              <Icon aria-hidden className="size-3.5" />
              {label}
            </div>
            <p className="mt-1 text-2xl font-semibold tabular-nums">{formatCompact(value)}</p>
          </div>
        ))}

        {/* A hairline between "what users are doing" and "what the hardware is
            doing" — two different questions that happen to share a row. */}
        <div aria-hidden className="hidden bg-border lg:block" />

        <Meter label="CPU" percent={metrics.current_cpu} />
        <Meter label="Memory" percent={metrics.current_memory} />
      </div>
    </Panel>
  );
}
