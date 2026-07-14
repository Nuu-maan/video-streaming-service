import { HistoryListSkeleton } from "@/features/history/components/history-list";
import { Skeleton } from "@/components/ui/skeleton";

/** The page's own shape — heading, clear button, six rows — not a spinner. */
export default function HistoryLoading() {
  return (
    <>
      <div className="flex flex-wrap items-end justify-between gap-x-6 gap-y-3">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-9 w-52 rounded-lg" />
          <Skeleton className="h-4 w-80 rounded-md" />
        </div>
        <Skeleton className="h-8 w-32 rounded-md" />
      </div>
      <HistoryListSkeleton rows={6} />
    </>
  );
}
