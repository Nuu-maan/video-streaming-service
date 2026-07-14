import { ServerOff } from "lucide-react";

import { EmptyState } from "@/components/common/empty-state";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Panel } from "@/features/admin/components/panel";
import type { WorkerInfo } from "@/features/admin/types";
import { formatRelativeTime } from "@/lib/format";

/**
 * The transcoding workers currently checked in.
 *
 * Zero workers with a non-empty queue is the single most useful fact this whole
 * admin area can tell you — it is the difference between "transcoding is slow"
 * and "transcoding is not happening" — so the empty state says so out loud
 * rather than shrugging with "no workers".
 *
 * Numbers are right-aligned and tabular so the columns actually line up; a
 * concurrency of 8 above a concurrency of 12 should not shift the digits.
 */
export function WorkerTable({ workers }: { workers: WorkerInfo[] }) {
  return (
    <Panel
      title="Workers"
      description="The machines pulling jobs off the queue."
      aside={
        <span className="text-sm tabular-nums text-muted-foreground">
          {workers.length} active
        </span>
      }
    >
      {workers.length === 0 ? (
        <EmptyState
          icon={ServerOff}
          title="No workers are running"
          description="Nothing is checked in to pull jobs off the queue, so no video will finish transcoding until a worker comes back."
          className="min-h-48 border-0"
        />
      ) : (
        <div className="-mx-1">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="min-w-40">Host</TableHead>
                <TableHead className="w-20 text-right">PID</TableHead>
                <TableHead className="w-28 text-right">Concurrency</TableHead>
                <TableHead className="w-24 text-right">Active</TableHead>
                <TableHead className="w-32">Started</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {workers.map((worker) => (
                <TableRow key={worker.server_id ?? `${worker.host}:${worker.pid}`}>
                  <TableCell className="font-mono text-xs">
                    {worker.host ?? "unknown"}
                    {worker.queues ? (
                      <span className="ml-2 text-muted-foreground">
                        {Object.keys(worker.queues).join(", ")}
                      </span>
                    ) : null}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">{worker.pid ?? "—"}</TableCell>
                  <TableCell className="text-right tabular-nums">
                    {worker.concurrency ?? "—"}
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {worker.active_tasks ?? 0}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {worker.started ? (
                      <time dateTime={worker.started}>{formatRelativeTime(worker.started)}</time>
                    ) : (
                      "—"
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </Panel>
  );
}
