import { Skeleton } from "@/components/ui/skeleton";

export default function NotificationsLoading() {
  return (
    <>
      <div className="flex items-end justify-between gap-6">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-8 w-52 rounded-md" />
          <Skeleton className="h-4 w-80 max-w-full rounded-md" />
        </div>
        <Skeleton className="h-8 w-28 rounded-md" />
      </div>

      <div className="flex flex-col gap-1">
        {Array.from({ length: 6 }, (_, index) => (
          <div key={index} className="flex items-start gap-3 px-3 py-3">
            <Skeleton className="size-9 shrink-0 rounded-full" />
            <div className="flex min-w-0 flex-1 flex-col gap-2">
              <Skeleton className="h-4 w-1/3 rounded-md" />
              <Skeleton className="h-4 w-3/4 rounded-md" />
            </div>
          </div>
        ))}
      </div>
    </>
  );
}
