import { Skeleton } from "@/components/ui/skeleton";
import { HistoryRow } from "@/features/history/components/history-row";
import type { HistoryRow as HistoryRowData } from "@/features/history/types";
import { cn } from "@/lib/utils";

export function HistoryList({ rows, className }: { rows: HistoryRowData[]; className?: string }) {
  return (
    <ul className={cn("flex flex-col gap-1", className)}>
      {rows.map((row) => (
        <HistoryRow key={row.entryId} row={row} />
      ))}
    </ul>
  );
}

/** Same metrics as the real rows — thumbnail width included — so nothing shifts. */
export function HistoryListSkeleton({ rows = 6, className }: { rows?: number; className?: string }) {
  return (
    <div className={cn("flex flex-col gap-1", className)} aria-hidden>
      {Array.from({ length: rows }, (_, index) => (
        <div key={index} className="flex items-start gap-3 p-2 sm:gap-4">
          <Skeleton className="aspect-video w-36 shrink-0 rounded-lg sm:w-44" />
          <div className="flex min-w-0 flex-1 flex-col gap-2 py-0.5">
            <Skeleton className="h-4 w-4/5 rounded-md" />
            <Skeleton className="h-3 w-32 rounded-md" />
            <Skeleton className="h-3 w-40 rounded-md" />
          </div>
        </div>
      ))}
    </div>
  );
}
