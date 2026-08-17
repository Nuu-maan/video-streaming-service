import { Clapperboard, Search, Upload } from "lucide-react";
import Link from "next/link";

import { MobileNav } from "@/components/layout/mobile-nav";
import { ThemeToggle } from "@/components/layout/theme-toggle";
import { UserMenu } from "@/components/layout/user-menu";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { site } from "@/config/site";
import type { User } from "@/types/common";

interface SiteHeaderProps {
  /** The signed-in user, or null. The header never fetches — the layout that renders it does. */
  user: User | null;
  /** Sign-out Server Action from `@/features/auth/actions`, threaded through to the user menu. */
  signOutAction: () => Promise<void>;
  /**
   * The search combobox — `<SearchInput />` from `@/features/search`.
   *
   * It arrives as a slot rather than an import for the same reason
   * `signOutAction` does: `components/` may not import `@/features/*` (the
   * boundary lint enforces it), and a header that reached into the search
   * feature would make the shell depend on a product domain. The layout owns
   * that dependency; the header owns the geometry. Omitting it degrades to a
   * plain link to /search rather than leaving a hole.
   */
  search?: React.ReactNode;
  /**
   * The notification bell — `<NotificationBell />` from `@/features/notifications`.
   * Same slot reasoning. Layouts pass it only when there is a signed-in user;
   * there is nothing to notify an anonymous visitor about.
   */
  notifications?: React.ReactNode;
}

/**
 * Sticky translucent app chrome. Server Component — the interactive pieces
 * (mobile nav, theme toggle, user menu, and whatever fills the slots) are
 * client leaves it composes.
 *
 * `data-translucent` lets globals.css solidify it under
 * `prefers-reduced-transparency`.
 */
export function SiteHeader({ user, signOutAction, search, notifications }: SiteHeaderProps) {
  return (
    <header
      data-translucent
      className="sticky top-0 z-40 flex h-14 items-center gap-2 border-b border-border/40 bg-background/75 px-3 backdrop-blur-lg backdrop-saturate-150 sm:px-4"
    >
      <MobileNav />

      <Link
        href={routes.home}
        className="flex items-center gap-2 rounded-md text-base font-semibold tracking-tight outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
      >
        <Clapperboard aria-hidden className="size-5 text-brand-500" />
        <span className="max-sm:sr-only">{site.name}</span>
      </Link>

      {/* The combobox needs room to drop its suggestion list, so it owns the
          centre column. Below `sm` it is hidden entirely and the icon button on
          the right takes over: a full autocomplete in a 320px header is a worse
          experience than a tap that opens the page built for it. */}
      <div className="flex flex-1 justify-center px-2">
        {search ? (
          <div className="hidden w-full max-w-xl sm:block">{search}</div>
        ) : (
          <Link
            href={routes.search}
            className="hidden h-9 w-full max-w-md items-center gap-2.5 rounded-full border border-input bg-muted/40 pr-4 pl-3.5 text-sm text-muted-foreground outline-none transition-colors duration-(--motion-fast) hover:bg-muted/70 hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 sm:flex"
          >
            <Search aria-hidden className="size-4 shrink-0" />
            Search videos
          </Link>
        )}
      </div>

      {/* gap-2 / gap-3: an icon button's hit area now extends 4px (fine pointer)
          or 6px (coarse) past its 32px glyph on every side, and two hit areas
          must never overlap — so adjacent icon buttons need at least twice that
          between them. This row is five buttons wide on a phone and still fits
          a 360px viewport. */}
      <div className="flex items-center gap-2 pointer-coarse:gap-3">
        <Button asChild variant="ghost" size="icon" className="sm:hidden" aria-label="Search">
          <Link href={routes.search}>
            <Search aria-hidden />
          </Link>
        </Button>

        <Button asChild variant="ghost" size="icon" aria-label="Upload a video">
          <Link href={routes.upload}>
            <Upload aria-hidden />
          </Link>
        </Button>

        {notifications}

        <ThemeToggle />

        {user ? (
          <UserMenu user={user} signOutAction={signOutAction} />
        ) : (
          <Button asChild size="sm" className="ml-1">
            <Link href={routes.login}>Sign in</Link>
          </Button>
        )}
      </div>
    </header>
  );
}
