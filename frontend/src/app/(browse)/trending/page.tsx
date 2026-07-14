import { Flame } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { getTrending } from "@/features/videos/api";
import { searchItemToVideoCard } from "@/features/videos/card-data";
import { VideoGrid } from "@/features/videos/components/video-grid";
import type { TrendingWindow } from "@/features/videos/types";
import { isApiError } from "@/lib/api-error";
import { cn } from "@/lib/utils";
import type { VideoSearchItem } from "@/types/common";

export const metadata: Metadata = {
  title: "Trending",
  description: "The most-watched videos right now.",
};

const WINDOWS: Array<{ value: TrendingWindow; label: string; blurb: string }> = [
  { value: "24h", label: "Today", blurb: "the last 24 hours" },
  { value: "7d", label: "This week", blurb: "the last 7 days" },
  { value: "30d", label: "This month", blurb: "the last 30 days" },
];

/** Anything that is not one of the three windows the API accepts becomes the default. */
function toWindow(value: string | string[] | undefined): TrendingWindow {
  const raw = Array.isArray(value) ? value[0] : value;
  return WINDOWS.some((w) => w.value === raw) ? (raw as TrendingWindow) : "24h";
}

/**
 * The window switcher.
 *
 * Links, not buttons — each window is a distinct URL, so it is shareable,
 * back/forward works, and the page stays a Server Component with no client
 * JavaScript at all.
 *
 * And a <nav> with aria-current, NOT role="tablist"/role="tab": these navigate.
 * The tab roles would promise a screen reader a tabpanel that swaps in place
 * without leaving the page, which is not what happens — announcing "tab" for
 * something that performs a navigation is a lie about how the control behaves.
 * The default window carries no `?window=` at all, so the canonical URL of
 * /trending stays clean.
 */
function WindowSwitcher({ active }: { active: TrendingWindow }) {
  return (
    <nav
      aria-label="Trending window"
      className="inline-flex items-center gap-1 rounded-full bg-muted/60 p-1"
    >
      {WINDOWS.map((option) => {
        const selected = option.value === active;
        return (
          <Link
            key={option.value}
            href={option.value === "24h" ? routes.trending : `${routes.trending}?window=${option.value}`}
            aria-current={selected ? "page" : undefined}
            className={cn(
              "rounded-full px-3.5 py-1.5 text-sm outline-none transition-colors duration-(--motion-fast)",
              "focus-visible:ring-3 focus-visible:ring-ring/50",
              selected
                ? "bg-background font-medium text-foreground shadow-border"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            {option.label}
          </Link>
        );
      })}
    </nav>
  );
}

/**
 * `GET /videos/trending` is not paginated — it returns a ranked list and that is
 * the whole of it. So this page has no <Pagination>: there is no page two, and
 * pretending otherwise would mean inventing a control that leads nowhere.
 *
 * The result is a discriminated union rather than JSX built inside the `catch`.
 * A try/catch around JSX does not catch a render error — React does not render a
 * component when its element is constructed — so it only reads as though it
 * does. The markup is chosen outside the catch.
 */
type Result =
  | { ok: true; items: VideoSearchItem[] }
  | { ok: false; reason: "rate-limited" | "failed" };

async function loadTrending(window: TrendingWindow): Promise<Result> {
  try {
    return { ok: true, items: await getTrending(window, 48) };
  } catch (error) {
    if (isApiError(error) && error.isRateLimited) return { ok: false, reason: "rate-limited" };
    return { ok: false, reason: "failed" };
  }
}

export default async function TrendingPage(props: PageProps<"/trending">) {
  const searchParams = await props.searchParams;
  const window = toWindow(searchParams.window);
  const active = WINDOWS.find((option) => option.value === window) ?? WINDOWS[0];

  const result = await loadTrending(window);

  return (
    <div className="mx-auto flex w-full max-w-7xl flex-1 flex-col gap-6 px-4 py-8 sm:px-6">
      <PageHeader
        title="Trending"
        description={`The most-watched videos of ${active.blurb}.`}
        actions={<WindowSwitcher active={window} />}
      />

      {!result.ok ? (
        result.reason === "rate-limited" ? (
          <ErrorState
            title="Slow down a moment"
            description="You're loading pages faster than we can serve them. Try again in a few seconds."
          />
        ) : (
          <ErrorState
            title="Trending didn't load"
            description="Something went wrong on our side. Refresh the page to try again."
          />
        )
      ) : result.items.length === 0 ? (
        <EmptyState
          icon={Flame}
          title="Nothing is trending yet"
          description={`No video has picked up enough views in ${active.blurb}. Try a longer window, or go and browse everything.`}
          action={
            <Button asChild size="sm" variant="secondary">
              <Link href={routes.videos}>Browse all videos</Link>
            </Button>
          }
        />
      ) : (
        <VideoGrid videos={result.items.map(searchItemToVideoCard)} />
      )}
    </div>
  );
}
