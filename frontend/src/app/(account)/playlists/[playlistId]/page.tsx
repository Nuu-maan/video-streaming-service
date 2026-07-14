import { Globe, Link2, ListVideo, Lock } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { Pagination } from "@/components/common/pagination";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { getCurrentUser } from "@/features/auth/current-user";
import { getPlaylist, listPlaylistVideos, toPlaylistRows } from "@/features/playlists/api";
import { PlaylistActions } from "@/features/playlists/components/playlist-actions";
import { PlaylistVideoList } from "@/features/playlists/components/playlist-video-list";
import { isApiError } from "@/lib/api-error";
import { formatCount } from "@/lib/format";

const PAGE_SIZE = 24;

const visibilityMeta = {
  public: { icon: Globe, label: "Public" },
  unlisted: { icon: Link2, label: "Unlisted" },
  private: { icon: Lock, label: "Private" },
} as const;

function toPage(value: string | string[] | undefined): number {
  const parsed = Number(Array.isArray(value) ? value[0] : value);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : 1;
}

export async function generateMetadata(
  props: PageProps<"/playlists/[playlistId]">,
): Promise<Metadata> {
  const { playlistId } = await props.params;
  const playlist = await getPlaylist(playlistId).catch(() => null);
  return { title: playlist?.title ?? "Playlist" };
}

/**
 * A private playlist that is not yours answers 404, exactly as a deleted one
 * does — the API refuses to confirm that it exists. So this renders "not found"
 * and never "you don't have permission", which would be a claim we cannot make
 * and a leak if we could.
 */
export default async function PlaylistPage(props: PageProps<"/playlists/[playlistId]">) {
  const [{ playlistId }, searchParams] = await Promise.all([props.params, props.searchParams]);
  const page = toPage(searchParams.page);

  /*
   * All three together. The video list depends only on `playlistId` and `page`,
   * both of which are known two lines up — it was sitting behind the Promise.all
   * as a lone `await`, which bought a needless serial round trip on every single
   * playlist view. The notFound() check below is unaffected: it still runs after
   * everything has settled.
   */
  const [user, playlist, videos] = await Promise.all([
    getCurrentUser(),
    getPlaylist(playlistId),
    listPlaylistVideos(playlistId, { page, limit: PAGE_SIZE }).catch((error: unknown) => {
      if (isApiError(error) && error.isRateLimited) return "rate-limited" as const;
      return "failed" as const;
    }),
  ]);

  if (!playlist) notFound();

  const isOwner = user?.id === playlist.user_id;
  const { icon: VisibilityIcon, label } = visibilityMeta[playlist.visibility];

  return (
    <>
      <PageHeader
        title={playlist.title}
        description={playlist.description || undefined}
        actions={isOwner ? <PlaylistActions playlist={playlist} /> : undefined}
      />

      <p className="-mt-6 flex items-center gap-1.5 text-sm text-muted-foreground">
        <VisibilityIcon aria-hidden className="size-3.5" />
        <span>{label}</span>
        <span aria-hidden>·</span>
        <span className="tabular-nums">{formatCount(playlist.video_count, "video")}</span>
      </p>

      {videos === "rate-limited" ? (
        <ErrorState
          title="Slow down a moment"
          description="You're loading pages faster than we can serve them. Try again shortly."
        />
      ) : videos === "failed" ? (
        <ErrorState title="This playlist didn't load" description="Refresh the page to try again." />
      ) : videos.items.length === 0 ? (
        <EmptyState
          icon={ListVideo}
          title="This playlist is empty"
          description={
            isOwner
              ? "Open a video and use Save to add it here."
              : "There is nothing in it yet."
          }
          action={
            isOwner ? (
              <Button asChild size="sm" variant="secondary">
                <Link href={routes.home}>Find something to add</Link>
              </Button>
            ) : undefined
          }
        />
      ) : (
        <>
          <PlaylistVideoList
            playlistId={playlist.id}
            rows={toPlaylistRows(videos.items)}
            canEdit={Boolean(isOwner)}
            offset={(page - 1) * PAGE_SIZE}
          />
          <Pagination pagination={videos.pagination} />
        </>
      )}
    </>
  );
}
