import { Globe, Link2, ListVideo, Lock } from "lucide-react";
import NextLink from "next/link";

import { routes } from "@/config/routes";
import { formatCount, formatDate } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { Playlist } from "@/types/common";

const visibilityMeta = {
  public: { icon: Globe, label: "Public" },
  unlisted: { icon: Link2, label: "Unlisted" },
  private: { icon: Lock, label: "Private" },
} as const;

/**
 * A playlist has no thumbnail of its own in this API, so the card leans on a
 * stacked-card motif instead of a broken image: it reads as "a pile of videos"
 * at a glance, and it never shifts layout while loading because there is
 * nothing to load.
 */
export function PlaylistCard({ playlist, className }: { playlist: Playlist; className?: string }) {
  const { icon: VisibilityIcon, label } = visibilityMeta[playlist.visibility];

  return (
    <NextLink
      href={routes.playlist(playlist.id)}
      className={cn(
        "group flex flex-col gap-3 rounded-xl p-3 outline-none transition-shadow duration-(--motion-fast) shadow-border hover:shadow-border-hover focus-visible:ring-3 focus-visible:ring-ring/50",
        className,
      )}
    >
      <div className="relative">
        {/* The stack: two offset plates behind the face, purely decorative. */}
        <div
          aria-hidden
          className="absolute inset-x-3 -top-1.5 h-3 rounded-t-lg bg-muted/50 ring-1 ring-border/40 ring-inset"
        />
        <div className="relative flex aspect-video items-center justify-center overflow-hidden rounded-lg bg-muted ring-1 ring-border/50 ring-inset">
          <ListVideo
            aria-hidden
            className="size-8 text-muted-foreground/60 transition-transform duration-(--motion-medium) ease-out-quart hover-ok:group-hover:scale-110"
          />
          <span className="absolute right-1.5 bottom-1.5 rounded-md bg-black/75 px-1.5 py-0.5 text-xs font-medium text-white tabular-nums">
            {playlist.video_count}
          </span>
        </div>
      </div>

      <div className="min-w-0">
        <h3 className="line-clamp-2 text-sm leading-snug font-medium text-pretty">{playlist.title}</h3>
        <p className="mt-1 flex items-center gap-1.5 text-xs text-muted-foreground">
          <VisibilityIcon aria-hidden className="size-3" />
          <span>{label}</span>
          <span aria-hidden>·</span>
          <span className="tabular-nums">{formatCount(playlist.video_count, "video")}</span>
        </p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          Updated <time dateTime={playlist.updated_at}>{formatDate(playlist.updated_at)}</time>
        </p>
      </div>
    </NextLink>
  );
}
