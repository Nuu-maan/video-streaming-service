"use client";

import { PanelLeft } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";

import { exploreNav, libraryNav, type NavItem } from "@/components/layout/nav-items";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

function SidebarLink({
  item,
  active,
  collapsed,
}: {
  item: NavItem;
  active: boolean;
  collapsed: boolean;
}) {
  const Icon = item.icon;
  const link = (
    <Link
      href={item.href}
      aria-current={active ? "page" : undefined}
      className={cn(
        // The transition has to name `transform`, or the press scale below snaps
        // in and snaps back with no easing at all — `transition-colors` does not
        // cover it, so the one tactile interaction in the nav was a hard jump.
        "flex h-10 items-center gap-3 rounded-lg px-3 text-sm outline-none transition-[color,background-color,transform] duration-(--motion-fast) ease-out-quart",
        "focus-visible:ring-3 focus-visible:ring-ring/50",
        "active:scale-[0.96]",
        active
          ? "bg-accent font-medium text-accent-foreground"
          : "text-muted-foreground hover:bg-accent/60 hover:text-foreground",
        collapsed && "justify-center px-0",
      )}
    >
      <Icon aria-hidden className={cn("size-4.5 shrink-0", active && "text-brand-500")} />
      {collapsed ? <span className="sr-only">{item.label}</span> : item.label}
    </Link>
  );

  if (!collapsed) return link;
  return (
    <Tooltip>
      <TooltipTrigger asChild>{link}</TooltipTrigger>
      <TooltipContent side="right">{item.label}</TooltipContent>
    </Tooltip>
  );
}

/**
 * Desktop navigation rail. Collapses to icons; the width change snaps rather
 * than animates — width is a layout property, and animating it repaints the
 * whole page. Hidden on mobile; MobileNav covers those viewports.
 */
export function SiteSidebar({ className }: { className?: string }) {
  const [collapsed, setCollapsed] = useState(false);
  const pathname = usePathname();

  const isActive = (href: string) =>
    href === "/" ? pathname === "/" : pathname === href || pathname.startsWith(`${href}/`);

  return (
    <TooltipProvider delayDuration={200}>
      <aside
        className={cn(
          "sticky top-14 hidden h-[calc(100dvh-3.5rem)] shrink-0 flex-col border-r border-border/40 bg-sidebar md:flex",
          collapsed ? "w-16 px-2.5" : "w-56 px-3",
          className,
        )}
      >
        <nav aria-label="Primary" className="flex flex-1 flex-col gap-1 overflow-y-auto py-3">
          {exploreNav.map((item) => (
            <SidebarLink key={item.href} item={item} active={isActive(item.href)} collapsed={collapsed} />
          ))}
          <div role="separator" className={cn("my-2 border-t border-border/60", collapsed ? "mx-1" : "mx-3")} />
          {libraryNav.map((item) => (
            <SidebarLink key={item.href} item={item} active={isActive(item.href)} collapsed={collapsed} />
          ))}
        </nav>
        <div className={cn("border-t border-border/40 py-2", collapsed && "flex justify-center")}>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
                aria-expanded={!collapsed}
                onClick={() => setCollapsed((value) => !value)}
              >
                <PanelLeft aria-hidden />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="right">
              {collapsed ? "Expand" : "Collapse"}
            </TooltipContent>
          </Tooltip>
        </div>
      </aside>
    </TooltipProvider>
  );
}
