import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

interface CardSkeletonProps {
  /** Leading 16:9 media block, for card grids with imagery. */
  media?: boolean;
  /** Text placeholder rows below the media. */
  lines?: number;
  className?: string;
}

/**
 * Generic card-shaped loading placeholder. Domain-shaped skeletons (the
 * video-card one with its avatar row) belong to their feature — this is the
 * neutral fallback for everything else.
 */
export function CardSkeleton({ media = true, lines = 2, className }: CardSkeletonProps) {
  return (
    <div className={cn("flex flex-col gap-3", className)}>
      {media ? <Skeleton className="aspect-video w-full rounded-xl" /> : null}
      <div className="flex flex-col gap-2">
        {Array.from({ length: lines }, (_, index) => (
          <Skeleton
            key={index}
            className={cn("h-4 rounded-md", index === lines - 1 ? "w-3/5" : "w-full")}
          />
        ))}
      </div>
    </div>
  );
}
