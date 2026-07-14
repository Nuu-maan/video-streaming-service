"use client";

import { Clapperboard, Menu } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";

import { exploreNav, libraryNav, type NavItem } from "@/components/layout/nav-items";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { routes } from "@/config/routes";
import { site } from "@/config/site";
import { cn } from "@/lib/utils";

function MobileNavLink({
  item,
  active,
  onNavigate,
}: {
  item: NavItem;
  active: boolean;
  onNavigate: () => void;
}) {
  const Icon = item.icon;
  return (
    <Link
      href={item.href}
      onClick={onNavigate}
      aria-current={active ? "page" : undefined}
      className={cn(
        "flex h-10 items-center gap-3 rounded-lg px-3 text-sm outline-none transition-colors duration-(--motion-fast)",
        "focus-visible:ring-3 focus-visible:ring-ring/50",
        active
          ? "bg-accent font-medium text-accent-foreground"
          : "text-muted-foreground hover:bg-accent/60 hover:text-foreground",
      )}
    >
      <Icon aria-hidden className="size-4.5 shrink-0" />
      {item.label}
    </Link>
  );
}

/** The primary nav behind a hamburger, for viewports where the sidebar is hidden. */
export function MobileNav() {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();
  const close = () => setOpen(false);

  const isActive = (href: string) =>
    href === "/" ? pathname === "/" : pathname === href || pathname.startsWith(`${href}/`);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" className="md:hidden" aria-label="Open navigation">
          <Menu aria-hidden />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="w-64 p-0">
        <SheetHeader className="border-b border-border/40 px-4 py-3">
          <SheetTitle asChild>
            <Link
              href={routes.home}
              onClick={close}
              className="flex items-center gap-2 text-base font-semibold tracking-tight outline-none focus-visible:rounded-md focus-visible:ring-3 focus-visible:ring-ring/50"
            >
              <Clapperboard aria-hidden className="size-5 text-brand-500" />
              {site.name}
            </Link>
          </SheetTitle>
        </SheetHeader>
        <nav aria-label="Primary" className="flex flex-col gap-1 p-3">
          {exploreNav.map((item) => (
            <MobileNavLink key={item.href} item={item} active={isActive(item.href)} onNavigate={close} />
          ))}
          <div role="separator" className="mx-3 my-2 border-t border-border/60" />
          {libraryNav.map((item) => (
            <MobileNavLink key={item.href} item={item} active={isActive(item.href)} onNavigate={close} />
          ))}
        </nav>
      </SheetContent>
    </Sheet>
  );
}
