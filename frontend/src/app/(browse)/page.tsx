import { Flame, Upload, Video as VideoIcon } from "lucide-react";
import Link from "next/link";
import { Suspense } from "react";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { routes } from "@/config/routes";
import { site } from "@/config/site";
import { CategoryChips } from "@/features/search/components/category-chips";
import { getCategories, getFeed, getTrending, listVideos } from "@/features/videos/api";
import { searchItemToVideoCard, toVideoCard } from "@/features/videos/card-data";
import { VideoGridSkeleton } from "@/features/videos/components/video-card-skeleton";
import { VideoGrid } from "@/features/videos/components/video-grid";
import { VideoRail, VideoRailSkeleton } from "@/features/videos/components/video-rail";
import type { VideoCardData } from "@/features/videos/types";
import { isApiError } from "@/lib/api-error";

/**
 * Home. Four independent sections, each its own async component behind its own
 * Suspense boundary: they all start fetching at once, and each paints the
 * moment its own data lands. A slow trending query therefore cannot hold the
 * main grid hostage, and a failing one cannot blank the page — the furniture
 * (feed, trending, categories) quietly renders nothing on error, while the
 * grid, which IS the page, says so out loud.
 */
export default function HomePage() {
  return (
    <div className="mx-auto flex w-full max-w-[1600px] flex-1 flex-col gap-10 px-4 py-6 sm:px-6">
      {/*
       * The document has to start at h1. Home has no visible page title — the
       * shelves ARE the page, and a big "Home" above them is a word nobody needs
       * — but heading navigation is a primary way a screen-reader user finds
       * their way around, and this page was starting them at h2 with level 1
       * skipped entirely. So: a real h1, addressed to the people who actually
       * use the outline.
       */}
      <h1 className="sr-only">{site.name} — browse videos</h1>

      {/* Signed in, and subscribed to someone who has posted? That is the most
          relevant thing on the page, so it leads. Otherwise it does not exist —
          an empty "from your subscriptions" shelf is worse than no shelf. */}
      <Suspense fallback={<RailSectionSkeleton />}>
        <FeedSection />
      </Suspense>

      <Suspense fallback={<RailSectionSkeleton />}>
        <TrendingSection />
      </Suspense>

      <Suspense fallback={<Skeleton className="h-8 w-full max-w-2xl rounded-full" />}>
        <CategorySection />
      </Suspense>

      <section aria-labelledby="home-latest" className="flex flex-1 flex-col">
        <SectionHeader id="home-latest" title="Latest videos" href={routes.videos} linkLabel="Browse all" />
        <Suspense fallback={<VideoGridSkeleton className="mt-4" count={12} />}>
          <LatestSection />
        </Suspense>
      </section>
    </div>
  );
}

/* -------------------------------------------------------------------------- */

async function FeedSection() {
  // An anonymous visitor's feed is a 401 by design. That is "signed out", not a
  // failure — and either way the answer is the same: no shelf.
  const feed = await getFeed({ page: 1, limit: 8 }).catch(() => null);
  if (!feed || feed.items.length === 0) return null;

  return (
    <section aria-labelledby="home-feed">
      <SectionHeader id="home-feed" title="From your subscriptions" href={routes.subscriptions} />
      <VideoRail className="mt-4" videos={feed.items.map(searchItemToVideoCard)} />
    </section>
  );
}

async function TrendingSection() {
  const trending = await getTrending("24h", 12).catch(() => []);
  if (trending.length === 0) return null;

  return (
    <section aria-labelledby="home-trending">
      <SectionHeader
        id="home-trending"
        title="Trending today"
        icon={<Flame aria-hidden className="size-[1.1em] text-brand-500" />}
        href={routes.trending}
      />
      <VideoRail className="mt-4" videos={trending.map(searchItemToVideoCard)} />
    </section>
  );
}

async function CategorySection() {
  const categories = await getCategories().catch(() => []);
  if (categories.length === 0) return null;

  return (
    <section aria-label="Browse by category">
      <CategoryChips categories={categories} />
    </section>
  );
}

/**
 * Fetch first, render second. The try/catch has to stay clear of JSX: React
 * does not render a component at the moment its element is constructed, so a
 * `catch` wrapped around JSX would never actually catch a render error — it
 * only lulls you into thinking it would. So the failure is turned into data
 * here, and the JSX for it is chosen outside.
 */
type LatestResult =
  | { ok: true; cards: VideoCardData[] }
  | { ok: false; reason: "rate-limited" | "failed" };

async function loadLatest(): Promise<LatestResult> {
  try {
    const latest = await listVideos({ page: 1, limit: 24 });
    return { ok: true, cards: latest.items.map(toVideoCard) };
  } catch (error) {
    if (isApiError(error) && error.isRateLimited) {
      return { ok: false, reason: "rate-limited" };
    }
    return { ok: false, reason: "failed" };
  }
}

async function LatestSection() {
  const result = await loadLatest();

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
        title="No videos yet"
        description="This place is brand new. Be the one who breaks the silence."
        action={
          <Button asChild>
            <Link href={routes.upload}>
              <Upload aria-hidden />
              Upload the first one
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
