"use client";

import {
  AtSign,
  Bell,
  MessageSquare,
  Reply,
  ThumbsUp,
  UserPlus,
  Video,
  type LucideIcon,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useOptimistic, useState, useTransition } from "react";

import { markNotificationRead } from "@/features/notifications/actions";
import { formatDate, formatRelativeTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { Notification } from "@/types/common";

const iconFor: Record<Notification["type"], LucideIcon> = {
  new_video: Video,
  comment: MessageSquare,
  reply: Reply,
  like: ThumbsUp,
  subscriber: UserPlus,
  mention: AtSign,
};

/**
 * A notification is a link that also has a side effect, so it is a button, not
 * an anchor: it marks itself read and then navigates. The read flip is
 * optimistic — the dot goes out under the finger, and the navigation follows
 * without waiting for the round trip.
 *
 * `action_url` is a path the API supplies. Only internal paths are followed; a
 * value that does not start with a single "/" is ignored rather than trusted.
 */
function isInternalPath(path: string | undefined): path is string {
  return Boolean(path && path.startsWith("/") && !path.startsWith("//"));
}

export function NotificationItem({ notification }: { notification: Notification }) {
  const router = useRouter();
  const [read, setRead] = useState(notification.read);
  const [optimisticRead, setOptimisticRead] = useOptimistic(read);
  const [, startTransition] = useTransition();
  const Icon = iconFor[notification.type] ?? Bell;
  const href = isInternalPath(notification.action_url) ? notification.action_url : null;

  function handle() {
    startTransition(async () => {
      if (!read) {
        setOptimisticRead(true);
        const result = await markNotificationRead(notification.id);
        if (result.ok) setRead(true);
      }
      if (href) router.push(href);
    });
  }

  return (
    <li>
      <button
        type="button"
        onClick={handle}
        aria-label={optimisticRead ? notification.title : `${notification.title} (unread)`}
        className={cn(
          "flex w-full items-start gap-3 rounded-xl px-3 py-3 text-left outline-none transition-colors duration-(--motion-fast) hover:bg-muted/60 focus-visible:ring-3 focus-visible:ring-ring/50",
          !optimisticRead && "bg-brand-500/[0.06]",
        )}
      >
        <span
          className={cn(
            "mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-full ring-1 ring-inset",
            optimisticRead
              ? "bg-muted text-muted-foreground ring-border/60"
              : // brand-400 on the light chip was 2.0:1 — under even the 3:1
                // floor a non-text icon has to clear. Scoped: 5.5:1 / 6.3:1.
                "bg-brand-500/12 text-brand-700 ring-brand-500/25 dark:text-brand-400",
          )}
        >
          <Icon aria-hidden className="size-4" />
        </span>

        <span className="min-w-0 flex-1">
          <span className="flex items-center gap-2">
            <span
              className={cn(
                "min-w-0 flex-1 truncate text-sm",
                optimisticRead ? "font-medium" : "font-semibold",
              )}
            >
              {notification.title}
            </span>
            <time
              dateTime={notification.created_at}
              title={formatDate(notification.created_at)}
              suppressHydrationWarning
              className="shrink-0 text-xs text-muted-foreground"
            >
              {formatRelativeTime(notification.created_at)}
            </time>
          </span>

          <span className="mt-0.5 line-clamp-2 block text-sm text-pretty text-muted-foreground">
            {notification.message}
          </span>
        </span>

        {optimisticRead ? null : (
          <span
            aria-hidden
            className="mt-3 size-2 shrink-0 rounded-full bg-brand-500"
            title="Unread"
          />
        )}
      </button>
    </li>
  );
}
