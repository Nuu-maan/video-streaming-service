import { Badge } from "@/components/ui/badge";
import { Panel } from "@/features/admin/components/panel";
import type { QueueStats } from "@/features/admin/types";
import { cn } from "@/lib/utils";

/**
 * Asynq's default queue, one cell per state.
 *
 * Every field on QueueStats is optional in the schema, which is not the API
 * being coy — it is Go serialising a zero as an absent key. `?? 0` is therefore
 * the correct reading of a missing field here, not a guess: a queue with no
 * archived tasks and a queue that forgot to mention its archived tasks are the
 * same queue.
 *
 * `retry` and `archived` are the two that mean something is wrong — a task that
 * has been retried is a task that failed at least once, and an archived task is
 * one that gave up entirely — so they colour when they are non-zero and stay
 * quiet when they are not.
 */
export function QueueStatsGrid({ stats }: { stats: QueueStats }) {
  const cells: Array<{ label: string; value: number; tone?: "warning" | "danger"; hint: string }> = [
    { label: "Active", value: stats.active ?? 0, hint: "Being worked on right now" },
    { label: "Pending", value: stats.pending ?? 0, hint: "Waiting for a free worker" },
    { label: "Scheduled", value: stats.scheduled ?? 0, hint: "Queued for a later time" },
    { label: "Retry", value: stats.retry ?? 0, tone: "warning", hint: "Failed once, will try again" },
    { label: "Archived", value: stats.archived ?? 0, tone: "danger", hint: "Gave up after every retry" },
    { label: "Completed", value: stats.completed ?? 0, hint: "Finished and retained" },
  ];

  return (
    <Panel
      title="Queue"
      description="The transcoding queue, task by task."
      aside={
        stats.paused ? (
          <Badge variant="destructive">Paused</Badge>
        ) : (
          <Badge variant="outline" className="border">
            Running
          </Badge>
        )
      }
    >
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
        {cells.map((cell) => {
          const alarming = Boolean(cell.tone) && cell.value > 0;
          return (
            <div key={cell.label} title={cell.hint}>
              <p className="truncate text-sm text-muted-foreground">{cell.label}</p>
              <p
                className={cn(
                  "mt-0.5 text-2xl font-semibold tabular-nums",
                  alarming && cell.tone === "danger" && "text-destructive",
                  alarming && cell.tone === "warning" && "text-amber-600 dark:text-amber-400",
                )}
              >
                {cell.value}
              </p>
            </div>
          );
        })}
      </div>

      {/* Lifetime counters, deliberately smaller: they are context, not a signal
          to act on. Anything here that ticks is a number the reader might watch,
          so the digits are tabular. */}
      <dl className="mt-5 flex flex-wrap gap-x-6 gap-y-1 border-t border-border pt-4 text-sm">
        <div className="flex gap-1.5">
          <dt className="text-muted-foreground">Processed, all time</dt>
          <dd className="font-medium tabular-nums">{(stats.processed ?? 0).toLocaleString("en")}</dd>
        </div>
        <div className="flex gap-1.5">
          <dt className="text-muted-foreground">Failed, all time</dt>
          <dd className="font-medium tabular-nums">{(stats.failed ?? 0).toLocaleString("en")}</dd>
        </div>
      </dl>
    </Panel>
  );
}
