import { Library, Upload, Video as VideoIcon } from "lucide-react";
import Link from "next/link";
import { Suspense } from "react";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { routes } from "@/config/routes";
import { site } from "@/config/site";
import { listHistory } from "@/features/history/api";
import { listMyPlaylists } from "@/features/playlists/api";
import { PlaylistCard } from "@/features/playlists/components/playlist-card";
import { listVideos } from "@/features/videos/api";
import { toVideoCard } from "@/features/videos/card-data";
import { VideoGridSkeleton } from "@/features/videos/components/video-card-skeleton";
import { VideoGrid } from "@/features/videos/components/video-grid";
import { VideoRail, VideoRailSkeleton } from "@/features/videos/components/video-rail";
import type { VideoCardData } from "@/features/videos/types";
import { isApiError } from "@/lib/api-error";

/**
 * The overview.
 *
 * Three finite shelves and an end to the page. There is no ranked feed here and
 * no "trending", by design: everything on this screen is something a person in
 * this workspace put here, and a shelf of what strangers are watching would be
 * the one section nobody owns. The page is meant to be finishable — you reach
 * the bottom of it.
 *
 * Each section is its own async component behind its own Suspense boundary, so
 * they all start fetching at once and each paints when its own data lands. The
 * two personal shelves return nothing at all when they are empty or when nobody
 * is signed in — an empty "pick up where you left off" is worse than no shelf.
 */
export default function OverviewPage() {
  return (
    <div className="mx-auto flex w-full max-w-[1600px] flex-1 flex-col gap-10 px-4 py-6 sm:px-6">
      {/*
       * The document has to start at h1. The overview has no visible page title
       * — the shelves ARE the page — but heading navigation is a primary way a
       * screen-reader user finds their way around, and this page was starting
       * them at h2 with level 1 skipped entirely.
       */}
      <h1 className="sr-only">{site.name} — workspace overview</h1>

      <Suspense fallback={<RailSectionSkeleton />}>
        <ContinueSection />
      </Suspense>

      <Suspense fallback={<CollectionsSkeleton />}>
        <CollectionsSection />
      </Suspense>

      <section aria-labelledby="overview-recent" className="flex flex-1 flex-col">
        <SectionHeader id="overview-recent" title="Recently added" href={routes.videos} linkLabel="All videos" />
        <Suspense fallback={<VideoGridSkeleton className="mt-4" count={12} />}>
          <RecentSection />
        </Suspense>
      </section>
    </div>
  );
}

/* -------------------------------------------------------------------------- */

/**
 * Started but not finished. `listHistory` costs one request per row (there is no
 * bulk video-by-ids endpoint), so this asks for six and no more — the shelf is a
 * prompt, not an archive, and /history is one click away.
 */
async function ContinueSection() {
  // An anonymous visitor's history is a 401 by design. That is "signed out", not
  // a failure — and either way the answer is the same: no shelf.
  const history = await listHistory({ page: 1, limit: 6 }).catch(() => null);
  if (!history) return null;

  const unfinished = history.items.filter((row) => !row.completed && row.progressPercent > 0);
  if (unfinished.length === 0) return null;

  return (
    <section aria-labelledby="overview-continue">
      <SectionHeader id="overview-continue" title="Pick up where you left off" href={routes.history} />
      <VideoRail className="mt-4" videos={unfinished.map((row) => row.video)} />
    </section>
  );
}

async function CollectionsSection() {
  const collections = await listMyPlaylists({ page: 1, limit: 6 }).catch(() => null);
  if (!collections || collections.items.length === 0) return null;

  return (
    <section aria-labelledby="overview-collections">
      <SectionHeader
        id="overview-collections"
        title="Collections"
        icon={<Library aria-hidden className="size-[1.1em] text-brand-500" />}
        href={routes.collections}
      />
      <div className="mt-4 grid grid-cols-2 gap-x-4 gap-y-6 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
        {collections.items.map((collection) => (
          <PlaylistCard key={collection.id} playlist={collection} />
        ))}
      </div>
    </section>
  );
}

/**
 * Fetch first, render second. The try/catch has to stay clear of JSX: React does
 * not render a component at the moment its element is constructed, so a `catch`
 * wrapped around JSX would never actually catch a render error — it only lulls
 * you into thinking it would. So the failure is turned into data here, and the
 * JSX for it is chosen outside.
 */
type RecentResult =
  | { ok: true; cards: VideoCardData[] }
  | { ok: false; reason: "rate-limited" | "failed" };

async function loadRecent(): Promise<RecentResult> {
  try {
    const recent = await listVideos({ page: 1, limit: 24 });
    return { ok: true, cards: recent.items.map(toVideoCard) };
  } catch (error) {
    if (isApiError(error) && error.isRateLimited) {
      return { ok: false, reason: "rate-limited" };
    }
    return { ok: false, reason: "failed" };
  }
}

async function RecentSection() {
  const result = await loadRecent();

  // A 429 is not "something broke" — it is "you, specifically, are going too
  // fast". That is the only version of this message a user can act on.
  if (!result.ok) {
    return result.reason === "rate-limited" ? (
      <ErrorState
        className="mt-4 flex-1"
        title="Slow down a little"
        description="You're browsing faster than the server allows. Give it a moment and try again."
      />
    ) : (
      <ErrorState
        className="mt-4 flex-1"
        title="Couldn't load videos"
        description="Something went wrong talking to the server. Refresh to try again."
      />
    );
  }

  if (result.cards.length === 0) {
    return (
      <EmptyState
        className="mt-4 flex-1"
        icon={VideoIcon}
        title="Nothing here yet"
        description="Upload something and it will appear here, transcoded and ready to review."
        action={
          <Button asChild>
            <Link href={routes.upload}>
              <Upload aria-hidden />
              Upload a video
            </Link>
          </Button>
        }
      />
    );
  }

  return <VideoGrid videos={result.cards} className="mt-4" />;
}

/* -------------------------------------------------------------------------- */

function SectionHeader({
  id,
  title,
  icon,
  href,
  linkLabel = "See all",
}: {
  id: string;
  title: string;
  icon?: React.ReactNode;
  href: string;
  linkLabel?: string;
}) {
  return (
    <div className="flex items-baseline justify-between gap-4">
      {/* The icon is sized in `em` and centred against the heading's own box,
          so it tracks the fluid heading size instead of drifting off it. */}
      <h2 id={id} className="flex items-center gap-2 text-heading">
        {icon}
        {title}
      </h2>
      <Link
        href={href}
        className="shrink-0 rounded-sm text-sm text-muted-foreground outline-none transition-colors duration-(--motion-fast) hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50"
      >
        {linkLabel}
      </Link>
    </div>
  );
}

/** Fallback for a shelf: heading bar, then a row of cards on the same metrics. */
function RailSectionSkeleton() {
  return (
    <div>
      <Skeleton className="h-6 w-44 rounded-md" />
      <VideoRailSkeleton className="mt-4" />
    </div>
  );
}

function CollectionsSkeleton() {
  return (
    <div>
      <Skeleton className="h-6 w-36 rounded-md" />
      <div className="mt-4 grid grid-cols-2 gap-x-4 gap-y-6 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
        {Array.from({ length: 6 }, (_, index) => (
          <div key={index} className="flex flex-col gap-3 p-3">
            <Skeleton className="aspect-video w-full rounded-lg" />
            <Skeleton className="h-4 w-3/4 rounded-md" />
            <Skeleton className="h-3 w-1/2 rounded-md" />
          </div>
        ))}
      </div>
    </div>
  );
}
