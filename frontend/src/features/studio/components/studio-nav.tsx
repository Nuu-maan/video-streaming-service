"use client";

import { Upload, Video } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { routes } from "@/config/routes";
import { cn } from "@/lib/utils";

const TABS = [
  { href: routes.studio, label: "Videos", icon: Video },
  { href: routes.upload, label: "Upload", icon: Upload },
] as const;

/**
 * The studio's own sub-navigation, sitting under the app header in place of
 * the browse sidebar. A client component only because it needs the current
 * path to mark the active tab.
 *
 * The active tab is marked with `aria-current`, not colour alone: an
 * underline a screen reader cannot see is not navigation, it is decoration.
 */
export function StudioNav() {
  const pathname = usePathname();

  return (
    <nav aria-label="Studio" className="border-b border-border">
      <ul className="mx-auto flex w-full max-w-6xl items-center gap-1 px-4 sm:px-6">
        {TABS.map((tab) => {
          const active = pathname === tab.href;
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
