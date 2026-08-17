import { Skeleton } from "@/components/ui/skeleton";

export default function SubscriptionsLoading() {
  return (
    <>
      <div className="flex flex-col gap-2">
        <Skeleton className="h-8 w-48 rounded-md" />
        <Skeleton className="h-4 w-80 max-w-full rounded-md" />
      </div>

      <div className="flex flex-col gap-1">
        {Array.from({ length: 6 }, (_, index) => (
          <div key={index} className="flex items-center gap-4 px-3 py-3">
            <Skeleton className="size-11 shrink-0 rounded-full" />
            <div className="flex min-w-0 flex-1 flex-col gap-2">
              <Skeleton className="h-4 w-40 rounded-md" />
              <Skeleton className="h-3 w-56 max-w-full rounded-md" />
            </div>
            <Skeleton className="h-8 w-28 shrink-0 rounded-full" />
          </div>
        ))}
      </div>
    </>
  );
}
