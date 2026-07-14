import { ListVideo, Plus } from "lucide-react";
import type { Metadata } from "next";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { Pagination } from "@/components/common/pagination";
import { Button } from "@/components/ui/button";
import { listMyPlaylists } from "@/features/playlists/api";
import { PlaylistCard } from "@/features/playlists/components/playlist-card";
import { PlaylistFormDialog } from "@/features/playlists/components/playlist-form-dialog";
import { isApiError } from "@/lib/api-error";

export const metadata: Metadata = { title: "Playlists" };

function toPage(value: string | string[] | undefined): number {
  const parsed = Number(Array.isArray(value) ? value[0] : value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 1;
}

export default async function PlaylistsPage(props: PageProps<"/playlists">) {
  const searchParams = await props.searchParams;
  const page = toPage(searchParams.page);

  const result = await listMyPlaylists({ page, limit: 24 }).catch((error: unknown) => {
    if (isApiError(error) && error.isRateLimited) return "rate-limited" as const;
    return "failed" as const;
  });

  const newPlaylistButton = (
    <PlaylistFormDialog
      trigger={
        <Button size="sm">
          <Plus aria-hidden />
          New playlist
        </Button>
      }
    />
  );

  return (
    <>
      <PageHeader
        title="Playlists"
        description="Collections you have made. Private ones are only ever visible to you."
        actions={newPlaylistButton}
      />

      {result === "rate-limited" ? (
        <ErrorState
          title="Slow down a moment"
          description="You're loading pages faster than we can serve them. Try again shortly."
        />
      ) : result === "failed" ? (
        <ErrorState
          title="Your playlists didn't load"
          description="Refresh the page to try again."
        />
      ) : result.items.length === 0 ? (
        <EmptyState
          icon={ListVideo}
          title="No playlists yet"
          description="A playlist is a queue you keep: talks to catch up on, a series to binge, anything you want to come back to."
          action={newPlaylistButton}
        />
      ) : (
        <>
          <div className="grid grid-cols-[repeat(auto-fill,minmax(15rem,1fr))] gap-4">
            {result.items.map((playlist) => (
              <PlaylistCard key={playlist.id} playlist={playlist} />
            ))}
          </div>
          <Pagination pagination={result.pagination} />
        </>
      )}
    </>
  );
}
