import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { SubscribeButton } from "@/features/subscriptions/components/subscribe-button";
import { formatCount, formatDate } from "@/lib/format";
import type { SubscriptionEntry } from "@/types/common";

interface SubscriptionCardProps {
  entry: SubscriptionEntry;
  /** The signed-in viewer, so the button knows not to offer a self-subscribe. */
  viewerId: string;
}

/**
 * One row of the subscriptions list.
 *
 * The channel is NOT a link, and that is a deliberate correction rather than an
 * oversight. It used to point at `/users/{id}`, which is a route that cannot
 * exist: the API has no `GET /users/{id}` and `GET /videos` filters only by
 * `mine=true`, so neither a creator's identity nor their videos can be fetched
 * by anyone but themselves. The link was a guaranteed 404. A row that shows who
 * you follow and lets you unfollow them is honest; a link into a dead end is
 * not, and is worse than no link at all.
 *
 * Everything a channel page could have shown that we CAN get — the name, the
 * avatar, the subscriber count — is already right here.
 */
export function SubscriptionCard({ entry, viewerId }: SubscriptionCardProps) {
  const initial = entry.username.slice(0, 1).toUpperCase();

  return (
    <li className="flex items-center gap-4 rounded-xl px-3 py-3 shadow-border transition-shadow duration-(--motion-fast) hover:shadow-border-hover">
      <div className="flex min-w-0 flex-1 items-center gap-4">
        <Avatar className="size-11 shrink-0">
          {entry.avatar_url ? <AvatarImage src={entry.avatar_url} alt="" /> : null}
          <AvatarFallback className="bg-brand-800 text-sm font-medium text-brand-100">
            {initial}
          </AvatarFallback>
        </Avatar>

        <div className="min-w-0">
          <p className="truncate text-sm font-medium">{entry.username}</p>
          <p className="mt-0.5 truncate text-xs text-muted-foreground">
            <span className="tabular-nums">
              {formatCount(entry.subscriber_count, "subscriber")}
            </span>
            <span aria-hidden> · </span>
            <span>Subscribed {formatDate(entry.subscribed_at)}</span>
          </p>
        </div>
      </div>

      <SubscribeButton
        channelId={entry.user_id}
        channelName={entry.username}
        initialSubscribed
        isAuthenticated
        isSelf={entry.user_id === viewerId}
        size="sm"
        className="shrink-0"
      />
    </li>
  );
}
