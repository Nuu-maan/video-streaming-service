import { BadgeCheck, User as UserIcon } from "lucide-react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Skeleton } from "@/components/ui/skeleton";
import { getUploader } from "@/features/player/api";
import { isSubscribedTo } from "@/features/subscriptions/api";
import { SubscribeButton } from "@/features/subscriptions/components/subscribe-button";
import { formatCount } from "@/lib/format";
import type { User, Video } from "@/types/common";

interface ChannelRowProps {
  video: Video;
  viewer: User | null;
}

/**
 * Who made this.
 *
 * Async, and deliberately rendered inside its own <Suspense>: it costs extra API
 * calls (see `getUploader` — the API has no user-profile endpoint) and not one
 * frame of the player should wait on them.
 *
 * The two reads run TOGETHER, and the reason they can is worth saying out loud.
 * The subscription check used to be chained behind `getUploader` purely to learn
 * `channel.id` — but `channel.id` IS `video.user_id`, which is sitting right
 * there in the props. Nothing ever forced the serial chain; it was an accident of
 * how the code was written, and it cost the slowest component on the watch page
 * an entire extra round trip (on top of the two `getUploader` itself was
 * serialising, and the multi-page scan inside `isSubscribedTo`).
 *
 * When the uploader cannot be identified it says so plainly. An invented name,
 * or a UUID dressed up as one, is worse than an honest blank.
 */
export async function ChannelRow({ video, viewer }: ChannelRowProps) {
  const uploaderId = video.user_id;

  // Only ask whether the viewer subscribes when there is a viewer, and someone
  // other than themselves to subscribe to.
  const wantsSubscriptionCheck = Boolean(viewer && uploaderId && viewer.id !== uploaderId);

  const [channel, subscribed] = await Promise.all([
    getUploader(video, viewer),
    wantsSubscriptionCheck && uploaderId
      ? isSubscribedTo(uploaderId).catch(() => false)
      : Promise.resolve(false),
  ]);

  if (!channel) {
    return (
      <div className="flex items-center gap-3">
        <Avatar className="size-10">
          <AvatarFallback>
            <UserIcon aria-hidden className="size-4 text-muted-foreground" />
          </AvatarFallback>
        </Avatar>
        <p className="text-sm text-muted-foreground">Uploader unavailable</p>
      </div>
    );
  }

  const initial = channel.username.slice(0, 1).toUpperCase();

  return (
    <div className="flex min-w-0 items-center gap-3">
      <Avatar className="size-10 shrink-0">
        {channel.avatarUrl ? <AvatarImage src={channel.avatarUrl} alt="" /> : null}
        <AvatarFallback className="text-sm font-medium">{initial}</AvatarFallback>
      </Avatar>

      <div className="min-w-0">
        <p className="flex items-center gap-1 text-sm font-medium">
          <span className="truncate">{channel.username}</span>
          {channel.verified ? (
            <BadgeCheck aria-label="Verified" className="size-3.5 shrink-0 text-muted-foreground" />
          ) : null}
        </p>
        <p className="text-xs text-muted-foreground">
          {channel.subscriberCount === null ? (
            channel.isViewer ? (
              "Your upload"
            ) : (
              "Channel"
            )
          ) : (
            <span className="tabular-nums">{formatCount(channel.subscriberCount, "subscriber")}</span>
          )}
        </p>
      </div>

      <SubscribeButton
        channelId={channel.id}
        channelName={channel.username}
        initialSubscribed={subscribed}
        isAuthenticated={Boolean(viewer)}
        isSelf={channel.isViewer}
        className="ml-2 shrink-0"
      />
    </div>
  );
}

/** The Suspense fallback. Same metrics as the row it stands in for, so nothing shifts. */
export function ChannelRowSkeleton() {
  return (
    <div className="flex items-center gap-3">
      <Skeleton className="size-10 shrink-0 rounded-full" />
      <div className="flex flex-col gap-1.5">
        <Skeleton className="h-4 w-32 rounded-md" />
        <Skeleton className="h-3 w-20 rounded-md" />
      </div>
    </div>
  );
}
