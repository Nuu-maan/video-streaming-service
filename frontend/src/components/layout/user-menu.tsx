"use client";

import {
  Clapperboard,
  LoaderCircle,
  LogOut,
  Settings,
  ShieldCheck,
} from "lucide-react";
import Link from "next/link";
import { useTransition } from "react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { routes } from "@/config/routes";
import type { User } from "@/types/common";

interface UserMenuProps {
  user: User;
  /**
   * The sign-out Server Action, passed down from the layout that renders the
   * shell. It lives in `@/features/auth/actions` — shared components cannot
   * import features (the boundary lint forbids it), so it arrives as a prop.
   */
  signOutAction: () => Promise<void>;
}

export function UserMenu({ user, signOutAction }: UserMenuProps) {
  const [isSigningOut, startSignOut] = useTransition();
  const initial = user.username.slice(0, 1).toUpperCase();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="rounded-full"
          aria-label="Account menu"
        >
          <Avatar className="size-7">
            {user.avatar_url ? (
              <AvatarImage src={user.avatar_url} alt="" />
            ) : null}
            <AvatarFallback className="bg-brand-800 text-xs font-medium text-brand-100">
              {initial}
            </AvatarFallback>
          </Avatar>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel className="flex flex-col gap-0.5">
          <span className="truncate font-medium">
            {user.full_name || user.username}
          </span>
          <span className="truncate text-xs font-normal text-muted-foreground">
            {user.email}
          </span>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          {/*
           * There is no "Profile" item, and that is deliberate rather than an
           * omission. A channel page needs two things the API does not expose:
           * there is no `GET /users/{id}` (so a creator's identity cannot be
           * fetched by id) and `GET /videos` filters only by `mine=true` (so a
           * creator's videos cannot be listed by anyone else). A `/users/:id`
           * route would therefore be a page with nothing honest to put on it.
           *
           * "Your videos" is what a profile would have been for, and Studio
           * already is exactly that — `mine=true` is the one creator listing
           * that does exist.
           */}
          <DropdownMenuItem asChild>
            <Link href={routes.studio}>
              <Clapperboard aria-hidden />
              Your videos
            </Link>
          </DropdownMenuItem>
          <DropdownMenuItem asChild>
            <Link href={routes.settings}>
              <Settings aria-hidden />
              Settings
            </Link>
          </DropdownMenuItem>
          {user.role === "admin" ? (
            <DropdownMenuItem asChild>
              <Link href={routes.admin}>
                <ShieldCheck aria-hidden />
                Admin
              </Link>
            </DropdownMenuItem>
          ) : null}
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          variant="destructive"
          disabled={isSigningOut}
          onSelect={(event) => {
            /* Keep the menu open while the action runs so the pending state
               is visible instead of the menu vanishing into silence. */
            event.preventDefault();
            startSignOut(async () => {
              await signOutAction();
            });
          }}
        >
          {isSigningOut ? (
            <LoaderCircle aria-hidden className="animate-spin" />
          ) : (
            <LogOut aria-hidden />
          )}
          Sign out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
