import type { Metadata } from "next";
import { Upload, Video as VideoIcon } from "lucide-react";
import Link from "next/link";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { Pagination } from "@/components/common/pagination";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { listVideos } from "@/features/videos/api";
import { toVideoCard } from "@/features/videos/card-data";
import { VideoGrid } from "@/features/videos/components/video-grid";
import type { VideoCardData } from "@/features/videos/types";
import { isApiError } from "@/lib/api-error";
import type { PaginationMeta } from "@/types/common";

export const metadata: Metadata = {
  title: "Videos",
  description: "Every video on the platform, newest first.",
};

const PER_PAGE = 24;

/** `?page=` is user input: it arrives as a string and may be junk. */
function parsePage(value: string | string[] | undefined): number {
  const raw = Array.isArray(value) ? value[0] : value;
  const parsed = Number(raw);
  if (!Number.isInteger(parsed) || parsed < 1) return 1;
  return parsed;
}

/**
 * The full catalogue, paginated. `searchParams` is a Promise in Next 16 — the
 * synchronous shim is gone — so it is awaited, not read.
 */
export default async function VideosPage(props: PageProps<"/videos">) {
  const searchParams = await props.searchParams;
  const page = parsePage(searchParams.page);

  return (
    <div className="mx-auto flex w-full max-w-[1600px] flex-1 flex-col gap-6 px-4 py-6 sm:px-6">
      <PageHeader title="Videos" description="Everything on the platform, newest first." />
      <VideoList page={page} />
    </div>
  );
}

/**
 * Fetch first, render second. JSX must not be constructed inside the try/catch:
 * React does not render a component when its element is constructed, so a catch
 * around JSX cannot see a render error — it only looks like it can. The failure
 * becomes data here; the markup for it is chosen below, in the clear.
 */
type ListResult =
  | { ok: true; cards: VideoCardData[]; pagination: PaginationMeta }
  | { ok: false; reason: "rate-limited" | "failed" };

async function loadVideos(page: number): Promise<ListResult> {
  try {
    const { items, pagination } = await listVideos({ page, limit: PER_PAGE });
    return { ok: true, cards: items.map(toVideoCard), pagination };
  } catch (error) {
    if (isApiError(error) && error.isRateLimited) {
      return { ok: false, reason: "rate-limited" };
    }
    return { ok: false, reason: "failed" };
  }
}

async function VideoList({ page }: { page: number }) {
  const result = await loadVideos(page);

  if (!result.ok) {
    return result.reason === "rate-limited" ? (
      <ErrorState
        className="flex-1"
        title="Slow down a little"
        description="You're browsing faster than the server allows. Give it a moment and try again."
      />
    ) : (
      <ErrorState
        className="flex-1"
        title="Couldn't load videos"
        description="Something went wrong talking to the server. Refresh to try again."
      />
    );
  }

  // An empty page 1 means the catalogue is empty — a real, designed beginning.
  // An empty page 12 just means they walked off the end of it. Different
  // situations, different sentences, different buttons.
  if (result.cards.length === 0) {
    return page > 1 ? (
      <EmptyState
        className="flex-1"
        icon={VideoIcon}
        title="Nothing on this page"
        description="You've gone past the last video. Head back to the start."
        action={
          <Button asChild variant="outline">
            <Link href={routes.videos}>Back to page one</Link>
          </Button>
        }
      />
    ) : (
      <EmptyState
        className="flex-1"
        icon={VideoIcon}
        title="No videos yet"
        description="The catalogue is empty. Somebody has to go first — it might as well be you."
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

  return (
    <>
      <VideoGrid videos={result.cards} />
      {result.pagination.total_pages > 1 ? (
        <Pagination pagination={result.pagination} className="mt-4" />
      ) : null}
    </>
  );
}
