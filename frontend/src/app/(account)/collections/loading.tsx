import { CardSkeleton } from "@/components/common/card-skeleton";
import { Skeleton } from "@/components/ui/skeleton";

export default function PlaylistsLoading() {
  return (
    <>
      <div className="flex items-end justify-between gap-6">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-8 w-40 rounded-md" />
          <Skeleton className="h-4 w-80 max-w-full rounded-md" />
        </div>
        <Skeleton className="h-8 w-32 rounded-md" />
      </div>

      <div className="grid grid-cols-[repeat(auto-fill,minmax(15rem,1fr))] gap-4">
        {Array.from({ length: 8 }, (_, index) => (
          <CardSkeleton key={index} lines={2} className="p-3" />
        ))}
      </div>
    </>
  );
}
