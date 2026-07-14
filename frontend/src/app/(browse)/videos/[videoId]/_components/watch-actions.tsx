import { ShareDialog } from "./share-dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { routes } from "@/config/routes";
import { site } from "@/config/site";
import { getMyRating } from "@/features/likes/api";
import { LikeButton } from "@/features/likes/components/like-button";
import { SaveToPlaylistDialog } from "@/features/playlists/components/save-to-playlist-dialog";
import { ReportDialog } from "@/features/reports/components/report-dialog";
import { isInWatchLater } from "@/features/watch-later/api";
import { WatchLaterButton } from "@/features/watch-later/components/watch-later-button";
import type { Video } from "@/types/common";

interface WatchActionsProps {
  video: Video;
  /** Sharing is open to anyone. Rating, saving and reporting are not. */
  signedIn: boolean;
}

/**
 * The action bar under the player: rate, share, save, report.
 *
 * Async, because three of the four buttons need to open in the right state —
 * a like button that renders empty and then pops into "liked" a beat later is
 * worse than one that arrives correct. Those reads are the reason this sits
 * behind its own <Suspense> in the page: the player must not wait on them.
 *
 * Share is implemented locally (a URL and a clipboard belong to no domain);
 * everything else is the social slice's component, given its initial state here.
 */
export async function WatchActions({ video, signedIn }: WatchActionsProps) {
  // A signed-out viewer has no rating and no watch-later list. Asking the API
  // for either would be two round trips to be told about a session we know is
  // not there.
  const [rating, saved] = signedIn
    ? await Promise.all([
        getMyRating(video.id).catch(() => null),
        isInWatchLater(video.id).catch(() => false),
      ])
    : [null, false];

  // The canonical, absolute link — the one a viewer expects to paste anywhere.
  const url = `${site.url}${routes.video(video.id)}`;

  return (
    <div className="flex flex-wrap items-center gap-2">
      <LikeButton
        videoId={video.id}
        likeCount={video.like_count}
        initialRating={rating}
        isAuthenticated={signedIn}
      />

      <ShareDialog url={url} title={video.title} visibility={video.visibility} />

      <WatchLaterButton videoId={video.id} initialSaved={saved} isAuthenticated={signedIn} />

      {/*
       * Playlists load when the popover opens, not here — most viewers never
       * save a video, and none of them should pay a request for the option. So
       * unlike the two buttons above, this one needs no initial state fetched.
       */}
      <SaveToPlaylistDialog videoId={video.id} isAuthenticated={signedIn} />

      <ReportDialog target={{ kind: "video", id: video.id }} isAuthenticated={signedIn} />
    </div>
  );
}

/**
 * Fallback with the same footprint, so the row does not jump when it resolves:
 * one block per control, in order — rate, share, save for later, save to
 * playlist, report.
 */
export function WatchActionsSkeleton() {
  return (
    <div className="flex items-center gap-2">
      <Skeleton className="h-9 w-28 rounded-full" />
      <Skeleton className="h-9 w-24 rounded-full" />
      <Skeleton className="h-9 w-24 rounded-full" />
      <Skeleton className="h-9 w-20 rounded-full" />
      <Skeleton className="size-9 rounded-full" />
    </div>
  );
}
