import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Suspense } from "react";

import { ChannelRow, ChannelRowSkeleton } from "./_components/channel-row";
import { ProcessingState } from "./_components/processing-state";
import { VideoDescription } from "./_components/video-description";
import { WatchActions, WatchActionsSkeleton } from "./_components/watch-actions";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { routes } from "@/config/routes";
import { site } from "@/config/site";
import { getCurrentUser } from "@/features/auth/current-user";
import { NotesSection } from "@/features/comments/components/notes-section";
import { getResumePosition, getVideo } from "@/features/player/api";
import { ReviewPlayer } from "@/features/review/components/review-player";
import { ReviewProvider } from "@/features/review/review-context";
import { mediaUrl } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";
import { getAccessToken } from "@/lib/session";
import type { Video } from "@/types/common";

/**
 * The review page.
 *
 * A Server Component, and it stays one: it decides the manifest URL (which needs
 * the video's visibility), reads the viewer's resume position (which needs their
 * token), and hands both to the client player as plain props. The player never
 * calls the API; it cannot — the bearer token is an httpOnly cookie and the
 * browser has no way to read it. That is the whole design.
 *
 * The layout is the product's argument: the player and the notes are on screen
 * together, at full height, because a note anchored to 2:14 only means something
 * next to the frame at 2:14. There is deliberately no related-videos rail — that
 * space is worth more to the people working on this video than to an algorithm
 * suggesting a different one.
 *
 * The two slow-ish reads — the channel row and the notes — each sit behind their
 * own <Suspense>, so the player streams to the viewer immediately and the
 * furniture arrives when it arrives.
 */

/**
 * A video the API will not serve answers 404 — never 403 — whether it does not
 * exist or merely is not yours. That is deliberate: a 403 would confirm the
 * existence of a private video to anyone who guessed its id. So this cannot, and
 * must not, distinguish the two. It says "not found", because that is the only
 * thing it honestly knows.
 */
async function loadVideo(videoId: string): Promise<Video> {
  try {
    return await getVideo(videoId);
  } catch (error) {
    if (isApiError(error) && error.isNotFound) notFound();
    throw error;
  }
}

export async function generateMetadata(props: PageProps<"/videos/[videoId]">): Promise<Metadata> {
  const { videoId } = await props.params;

  let video: Video;
  try {
    video = await getVideo(videoId);
  } catch {
    return { title: "Video not found" };
  }

  const isPublic = video.visibility === "public";
  const description = video.description.trim().slice(0, 200) || `Watch ${video.title} on ${site.name}.`;
  const thumbnail = isPublic ? mediaUrl(video.thumbnail_url) : null;
  const url = `${site.url}${routes.video(video.id)}`;

  return {
    title: video.title,
    description,
    alternates: { canonical: url },
    // An unlisted video is unlisted. Handing it to a crawler undoes the setting.
    robots: isPublic ? undefined : { index: false, follow: false },
    openGraph: {
      type: "video.other",
      title: video.title,
      description,
      url,
      siteName: site.name,
      images: thumbnail ? [{ url: thumbnail, width: 1280, height: 720, alt: video.title }] : undefined,
    },
    twitter: {
      card: thumbnail ? "summary_large_image" : "summary",
      title: video.title,
      description,
      images: thumbnail ? [thumbnail] : undefined,
    },
  };
}

export default async function ReviewPage(props: PageProps<"/videos/[videoId]">) {
  const { videoId } = await props.params;

  /*
   * Kicked off before the Promise.all, awaited after it.
   *
   * `getResumePosition` needs nothing but the videoId — which we have on the
   * line above — yet it used to sit as a lone `await` AFTER the video and the
   * viewer had both resolved, so every signed-in viewer ate a full extra round
   * trip (a 100-item scan of /me/history) before the player element could even
   * be constructed. The player is the entire point of this page, and it was
   * queued behind a history lookup it does not need in order to start fetching
   * the manifest.
   *
   * The gate is the session COOKIE, not the user: reading a cookie is free and
   * local, so this starts the lookup in parallel without spending a
   * guaranteed-401 request on every anonymous viewer. (Waiting for
   * `getCurrentUser()` to answer would have put the serial await straight back.)
   */
  const resumePromise = (await getAccessToken())
    ? getResumePosition(videoId)
    : Promise.resolve(null);

  const [video, viewer] = await Promise.all([loadVideo(videoId), getCurrentUser()]);

  const isPublic = video.visibility === "public";
  const isOwner = Boolean(viewer && video.user_id && viewer.id === video.user_id);
  const playable = video.status === "ready" && Boolean(video.hls_url);

  /**
   * Where the media comes from, decided here because only here can it be.
   *
   * A public video is fetched straight off the API origin: cheaper, cacheable,
   * and it keeps the video bytes out of the Next server entirely. A private or
   * unlisted one has to come through this origin's `/api/media` proxy, which
   * attaches the bearer token server-side — hls.js cannot, and giving it the
   * token would defeat the point of the cookie being httpOnly.
   */
  const src = isPublic
    ? mediaUrl(video.hls_url)
    : `/api/media/videos/${video.id}/hls/master.m3u8`;
  const poster = video.thumbnail_url
    ? isPublic
      ? mediaUrl(video.thumbnail_url)
      : `/api/media/videos/${video.id}/thumbnail`
    : null;

  // Only a signed-in viewer has a resume position, and only they can save one.
  // By now this has been in flight since the top of the function.
  const resumeAt = viewer ? await resumePromise : null;

  return (
    /*
     * The provider wraps both columns because they are siblings, and they are
     * siblings so the player can stream while the notes are still being fetched.
     * It costs one client boundary at the top of the page and buys back the
     * whole streaming order.
     */
    <ReviewProvider>
      <div
        className={
          /*
           * Two columns that each own their own scroll on desktop, one stacked
           * column on anything narrower.
           *
           * `h-[calc(100dvh-3.5rem)]` — the viewport minus the sticky header —
           * is what makes "full height" true: without a bounded height the rail
           * cannot scroll independently, and the composer at its foot would sit
           * a thousand pixels down the page. dvh rather than vh because mobile
           * browsers shrink the viewport as their chrome slides away, and vh
           * does not notice.
           */
          "mx-auto flex w-full max-w-[1800px] flex-col gap-6 px-4 py-4 sm:px-6 " +
          "xl:h-[calc(100dvh-3.5rem)] xl:flex-row xl:gap-0 xl:overflow-hidden xl:px-0 xl:py-0"
        }
      >
        <div className="flex min-w-0 flex-1 flex-col gap-4 xl:overflow-y-auto xl:px-6 xl:py-5">
          {playable && src ? (
            <ReviewPlayer
              videoId={video.id}
              src={src}
              poster={poster}
              title={video.title}
              trackProgress={Boolean(viewer)}
              resumeAt={resumeAt}
            />
          ) : (
            <ProcessingState video={video} isOwner={isOwner} />
          )}

          <div className="flex flex-col gap-3">
            <div className="flex items-start gap-3">
              <h1 className="min-w-0 flex-1 text-title text-balance">{video.title}</h1>
              {!isPublic ? (
                <Badge variant="secondary" className="mt-1 shrink-0 capitalize">
                  {video.visibility}
                </Badge>
              ) : null}
            </div>

            <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-3">
              <Suspense fallback={<ChannelRowSkeleton />}>
                <ChannelRow video={video} viewer={viewer} />
              </Suspense>
              <Suspense fallback={<WatchActionsSkeleton />}>
                <WatchActions video={video} signedIn={Boolean(viewer)} />
              </Suspense>
            </div>
          </div>

          <VideoDescription
            description={video.description}
            viewCount={video.view_count}
            createdAt={video.created_at}
            category={video.category}
            tags={video.tags}
          />
        </div>

        {/*
         * The rail. A real <aside> with its own border rather than a floating
         * card: it is a permanent part of this screen, not a panel that happens
         * to be open, and the seam between the two columns should read as
         * structural.
         *
         * `sort` follows what there is to sort. A video with no timeline yet has
         * no meaningful anchor order, so it falls back to newest.
         */}
        <aside className="w-full shrink-0 border-t border-border/60 xl:h-full xl:w-[380px] xl:border-t-0 xl:border-l 2xl:w-[420px]">
          <Suspense fallback={<NotesSkeleton />}>
            <NotesSection
              videoId={video.id}
              videoOwnerId={video.user_id}
              sort={playable ? "timestamp" : "newest"}
            />
          </Suspense>
        </aside>
      </div>
    </ReviewProvider>
  );
}

/** Shaped like the rail it precedes: header bar, a few notes, a composer. */
function NotesSkeleton() {
  return (
    <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)_auto]">
      <div className="flex items-center justify-between border-b border-border/60 px-4 py-3">
        <Skeleton className="h-5 w-20 rounded-md" />
        <Skeleton className="h-7 w-32 rounded-lg" />
      </div>
      <div className="flex flex-col gap-5 px-4 py-4">
        {Array.from({ length: 4 }, (_, index) => (
          <div key={index} className="flex gap-3">
            <Skeleton className="size-9 shrink-0 rounded-full" />
            <div className="flex min-w-0 flex-1 flex-col gap-2">
              <Skeleton className="h-3.5 w-32 rounded-md" />
              <Skeleton className="h-4 w-full rounded-md" />
              <Skeleton className="h-4 w-3/5 rounded-md" />
            </div>
          </div>
        ))}
      </div>
      <div className="border-t border-border/60 px-4 py-3">
        <Skeleton className="h-9 w-full rounded-lg" />
      </div>
    </div>
  );
}
