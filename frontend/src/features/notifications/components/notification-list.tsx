import { NotificationItem } from "@/features/notifications/components/notification-item";
import type { NotificationDay } from "@/features/notifications/types";

/**
 * Grouped by day, because "3 hours ago" and "last Tuesday" are answers to
 * different questions. The heading is sticky within its group: scroll a long
 * list and you never lose track of which day you are in.
 */
export function NotificationList({ days }: { days: NotificationDay[] }) {
  return (
    <div className="flex flex-col gap-6">
      {days.map((day) => (
        <section key={day.key} aria-labelledby={`day-${day.key}`}>
          <h2
            id={`day-${day.key}`}
            className="sticky top-14 z-10 bg-background/85 py-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase backdrop-blur-sm"
          >
            {day.label}
          </h2>
          <ul className="mt-1 flex flex-col gap-1">
            {day.items.map((notification) => (
              <NotificationItem key={notification.id} notification={notification} />
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
