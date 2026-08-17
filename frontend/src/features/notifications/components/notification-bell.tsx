"use client";

import { Bell } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { fetchUnreadCount } from "@/features/notifications/actions";
import { routes } from "@/config/routes";

interface NotificationBellProps {
  /** The count at render time, so the badge is correct before the first poll. */
  initialCount?: number;
}

/** Once a minute. A notification badge is not a stock ticker. */
const POLL_INTERVAL_MS = 60_000;

/**
 * The header bell.
 *
 * It polls, rather than holding a socket open, because the payload is a single
 * integer and the cost of being sixty seconds stale is zero. It also stops
 * polling entirely while the tab is hidden — a background tab that keeps asking
 * is how a phone battery disappears — and refreshes immediately when the tab
 * comes back, so returning to the tab shows a current count, not a stale one
 * plus a minute's wait.
 *
 * `visibilitychange` is the ONLY listener, and that is the fix rather than an
 * omission. It used to listen for `focus` as well — but returning to a
 * backgrounded tab fires both, so every tab switch cost two round trips instead
 * of one, from the most-mounted client component in the app, against a 60/min
 * budget. visibilitychange already covers the case the `focus` handler was
 * written for.
 */
export function NotificationBell({ initialCount = 0 }: NotificationBellProps) {
  const [count, setCount] = useState(initialCount);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setInterval> | undefined;

    async function refresh() {
      if (document.visibilityState !== "visible") return;
      const next = await fetchUnreadCount();
      if (!cancelled) setCount(next);
    }

    function start() {
      timer ??= setInterval(() => void refresh(), POLL_INTERVAL_MS);
    }

    function stop() {
      if (timer === undefined) return;
      clearInterval(timer);
      timer = undefined;
    }

    function handleVisibility() {
      if (document.visibilityState === "visible") {
        void refresh();
        start();
      } else {
        stop();
      }
    }

    handleVisibility();
    document.addEventListener("visibilitychange", handleVisibility);

    return () => {
      cancelled = true;
      stop();
      document.removeEventListener("visibilitychange", handleVisibility);
    };
  }, []);

  const label =
    count > 0
      ? `Notifications, ${count} unread`
      : "Notifications";

  return (
    <Button asChild variant="ghost" size="icon" className="relative" aria-label={label}>
      <Link href={routes.notifications}>
        <Bell aria-hidden />
        {count > 0 ? (
          <span
            aria-hidden
            className="absolute top-1 right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-brand-500 px-1 text-[0.625rem] leading-none font-semibold text-white tabular-nums ring-2 ring-background"
          >
            {count > 99 ? "99+" : count}
          </span>
        ) : null}
        {/* The count also reaches a screen reader through the button's label,
            which is announced on focus rather than shouted on every poll. */}
      </Link>
    </Button>
  );
}
