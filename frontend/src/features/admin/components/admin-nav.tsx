"use client";

import { ChartNoAxesColumn, Flag, ListChecks, Users } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { routes } from "@/config/routes";
import { cn } from "@/lib/utils";

const TABS = [
  { href: routes.admin, label: "Overview", icon: ChartNoAxesColumn },
  { href: routes.adminReports, label: "Reports", icon: Flag },
  { href: routes.adminUsers, label: "Users", icon: Users },
  { href: routes.adminQueue, label: "Queue", icon: ListChecks },
] as const;

/**
 * The admin area's sub-navigation. A client component only because it needs the
 * current path to mark the active tab.
 *
 * "Overview" matches exactly; the rest match their subtree, so a future
 * `/admin/reports/{id}` still lights up "Reports". Without the exact check on
 * the index route, `/admin` would be a prefix of all four and every tab would
 * claim to be active at once.
 */
export function AdminNav() {
  const pathname = usePathname();

  return (
    <nav aria-label="Admin" className="border-b border-border">
      <ul className="mx-auto flex w-full max-w-6xl items-center gap-1 px-4 sm:px-6">
        {TABS.map((tab) => {
          const active =
            tab.href === routes.admin ? pathname === tab.href : pathname.startsWith(tab.href);
          const Icon = tab.icon;

          return (
            <li key={tab.href}>
              <Link
                href={tab.href}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "-mb-px flex items-center gap-2 border-b-2 px-3 py-3 text-sm font-medium outline-none",
                  "transition-colors duration-(--motion-fast) ease-(--ease-out-quart)",
                  "focus-visible:ring-3 focus-visible:ring-ring/50",
                  active
                    ? "border-brand-500 text-foreground"
                    : "border-transparent text-muted-foreground hover:border-border hover:text-foreground",
                )}
              >
                <Icon aria-hidden className="size-4" />
                {tab.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
