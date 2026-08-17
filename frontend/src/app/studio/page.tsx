import { Upload } from "lucide-react";
import Link from "next/link";
import { Suspense } from "react";

import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { Pagination } from "@/components/common/pagination";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { getMyVideos } from "@/features/studio/api";
import { StudioEmptyState } from "@/features/studio/components/studio-empty-state";
import { StudioVideoTable } from "@/features/studio/components/studio-video-table";
import { toStudioRow } from "@/features/studio/row-data";
import { isApiError } from "@/lib/api-error";
import { formatCount } from "@/lib/format";
import type { Page, Video } from "@/types/common";

/** `?page=` is attacker-writable. Anything that is not a positive integer is page 1. */
function pageNumber(value: string | string[] | undefined): number {
  const raw = Array.isArray(value) ? value[0] : value;
  const parsed = Number(raw);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 1;
}

export default async function StudioPage(props: PageProps<"/studio">) {
  const searchParams = await props.searchParams;
  const page = pageNumber(searchParams.page);

  let result: Page<Video>;
  try {
    result = await getMyVideos({ page });
  } catch (error) {
    // A creator staring at a red box wants to know whether to retry now or in
    // a minute, so the two cases the API actually distinguishes get their own
    // copy and everything else gets an honest shrug.
    const rateLimited = isApiError(error) && error.isRateLimited;
    return (
      <>
        <StudioHeader />
        <ErrorState
          title={rateLimited ? "Slow down a moment" : "Couldn't load your videos"}
          description={
            rateLimited
              ? "You're making requests faster than the API allows. Wait a minute, then reload."
              : "Something went wrong fetching your library. Reload the page to try again."
          }
        />
      </>
    );
  }

  const { items, pagination } = result;

  return (
    <>
      <StudioHeader total={pagination.total} />

      {items.length === 0 ? (
        <StudioEmptyState />
      ) : (
        <>
          <StudioVideoTable videos={items.map(toStudioRow)} />
          {/* Pagination reads useSearchParams; this page is dynamic, but the
              Suspense boundary keeps it safe if that ever changes. */}
          <Suspense>
            <Pagination pagination={pagination} />
          </Suspense>
        </>
      )}
    </>
  );
}

function StudioHeader({ total }: { total?: number }) {
  return (
    <PageHeader
      title="Your videos"
      description={
        total === undefined || total === 0
          ? "Everything you've uploaded — public, unlisted and private."
          : `${formatCount(total, "video")} — public, unlisted and private.`
      }
      actions={
        <Button asChild>
          <Link href={routes.upload}>
            <Upload aria-hidden data-icon="inline-start" />
            Upload
          </Link>
        </Button>
      }
    />
  );
}
