import { Skeleton } from "@/components/ui/skeleton";

interface StudioVideoTableSkeletonProps {
  rows?: number;
}

/**
 * The table's shape while it loads: the same 16:9 thumbnail, the same two
 * lines of title, the same badge-sized blocks. A skeleton that matches the
 * real content means nothing moves when the data lands.
 */
export function StudioVideoTableSkeleton({ rows = 6 }: StudioVideoTableSkeletonProps) {
  return (
    <div aria-hidden className="overflow-hidden rounded-xl bg-card shadow-border">
      {Array.from({ length: rows }, (_, index) => (
        <div
          key={index}
          className="flex items-center gap-3 border-b border-border/60 px-4 py-3 last:border-b-0"
        >
          <Skeleton className="aspect-video w-28 shrink-0 rounded-lg" />
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <Skeleton className="h-4 w-3/5 rounded-md" />
            <Skeleton className="h-4 w-2/5 rounded-md" />
          </div>
          <Skeleton className="hidden h-5 w-20 rounded-full md:block" />
          <Skeleton className="hidden h-5 w-20 rounded-full md:block" />
          <Skeleton className="hidden h-4 w-24 rounded-md lg:block" />
        </div>
      ))}
    </div>
  );
}
