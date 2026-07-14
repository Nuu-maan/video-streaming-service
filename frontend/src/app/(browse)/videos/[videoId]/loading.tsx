import { Skeleton } from "@/components/ui/skeleton";
import { RelatedVideosSkeleton } from "@/features/search/components/related-videos";

/**
 * The watch page, one beat earlier.
 *
 * Shaped like the real thing — a 16:9 well where the player goes, a two-line
 * title, a channel row, a description block — so the arrival of the real content
 * is a fill, not a reflow. A centred spinner would tell the viewer only that
 * something is happening; this tells them what is about to appear and where.
 */
export default function WatchLoading() {
  return (
    <div className="mx-auto flex w-full max-w-[1600px] flex-col gap-6 px-4 py-4 sm:px-6 lg:py-6 xl:flex-row xl:gap-8">
      <div className="flex min-w-0 flex-1 flex-col gap-4">
        <Skeleton className="aspect-video w-full rounded-xl" />

        <div className="flex flex-col gap-3">
          <div className="space-y-2">
            <Skeleton className="h-7 w-4/5 rounded-md" />
            <Skeleton className="h-7 w-1/3 rounded-md" />
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <Skeleton className="size-10 shrink-0 rounded-full" />
              <div className="flex flex-col gap-1.5">
                <Skeleton className="h-4 w-32 rounded-md" />
                <Skeleton className="h-3 w-20 rounded-md" />
              </div>
            </div>
            <Skeleton className="h-8 w-24 rounded-full" />
          </div>
        </div>

        <div className="flex flex-col gap-2 rounded-xl bg-muted/50 px-4 py-3">
          <Skeleton className="h-4 w-48 rounded-md" />
          <Skeleton className="h-4 w-full rounded-md" />
          <Skeleton className="h-4 w-2/3 rounded-md" />
        </div>
      </div>

      <aside className="w-full shrink-0 xl:w-96 2xl:w-[420px]">
        <RelatedVideosSkeleton />
      </aside>
    </div>
  );
}
