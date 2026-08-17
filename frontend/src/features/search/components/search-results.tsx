import { BadgeCheck } from "lucide-react";
import Link from "next/link";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Skeleton } from "@/components/ui/skeleton";
import { routes } from "@/config/routes";
import { ResultThumbnail } from "@/features/search/components/result-thumbnail";
import { mediaUrl } from "@/lib/api-client";
import { formatCount, formatRelativeTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { VideoSearchItem } from "@/types/common";

interface SearchResultsProps {
  items: VideoSearchItem[];
  className?: string;
}

/**
 * The search result list: horizontal rows — thumbnail left, title, meta,
 * uploader and the API's match snippet right. Rows, not grid cards: a search
 * is read top to bottom, and the snippet is the whole reason a result earns
 * its place.
 *
 * Server Component; the whole row is one link.
 */
export function SearchResults({ items, className }: SearchResultsProps) {
  return (
    <ul className={className}>
      {items.map((item) => (
        <li key={item.video_id}>
          <Link
            href={routes.video(item.video_id)}
            className="group -mx-3 flex gap-3 rounded-xl p-3 outline-none transition-colors duration-(--motion-fast) hover:bg-muted/50 focus-visible:ring-3 focus-visible:ring-ring/50 sm:gap-4"
          >
            <ResultThumbnail
              src={item.thumbnail_url}
              duration={item.duration}
              className="w-40 sm:w-60 md:w-72"
            />

            <div className="min-w-0 flex-1 py-0.5">
              <h3 className="line-clamp-2 text-sm font-medium text-balance sm:text-base">
                {item.title}
              </h3>

              <p className="mt-1 text-xs text-muted-foreground">
                <span className="tabular-nums">{formatCount(item.views, "view")}</span>
                <span aria-hidden className="px-1.5">
                  ·
                </span>
                <time dateTime={item.created_at}>{formatRelativeTime(item.created_at)}</time>
              </p>

              <p className="mt-2 flex items-center gap-1.5 text-xs text-muted-foreground">
                <Avatar className="size-5">
                  <AvatarImage src={mediaUrl(item.user_avatar_url) ?? undefined} alt="" />
                  <AvatarFallback className="text-[9px] uppercase">
                    {item.username.slice(0, 1)}
                  </AvatarFallback>
                </Avatar>
                <span className="truncate">{item.username}</span>
                {item.user_verified ? (
                  <>
                    <BadgeCheck aria-hidden className="size-3.5 shrink-0 text-muted-foreground/80" />
                    <span className="sr-only">Verified</span>
                  </>
                ) : null}
              </p>

              {item.snippet ? (
                <p className="mt-2 hidden text-xs leading-relaxed text-pretty text-muted-foreground sm:line-clamp-2">
                  {item.snippet}
                </p>
              ) : null}
            </div>
          </Link>
        </li>
      ))}
    </ul>
  );
}

/**
 * The loading state for a page of results, shaped like the rows it stands in
 * for — same thumbnail widths, same padding, same rhythm — so the swap to real
 * content lands without the page jumping.
 */
export function SearchResultsSkeleton({ rows = 6, className }: { rows?: number; className?: string }) {
  return (
    <div className={cn("flex flex-col", className)} aria-hidden>
      {Array.from({ length: rows }, (_, index) => (
        <div key={index} className="-mx-3 flex gap-3 p-3 sm:gap-4">
          <Skeleton className="aspect-video w-40 shrink-0 rounded-lg sm:w-60 md:w-72" />
          <div className="min-w-0 flex-1 py-0.5">
            <Skeleton className="h-4 w-4/5 rounded-md sm:h-5" />
            <Skeleton className="mt-2 h-3 w-40 rounded-md" />
            <div className="mt-2.5 flex items-center gap-1.5">
              <Skeleton className="size-5 shrink-0 rounded-full" />
              <Skeleton className="h-3 w-24 rounded-md" />
            </div>
            <Skeleton className="mt-3 hidden h-3 w-full rounded-md sm:block" />
            <Skeleton className="mt-1.5 hidden h-3 w-2/3 rounded-md sm:block" />
          </div>
        </div>
      ))}
    </div>
  );
}
