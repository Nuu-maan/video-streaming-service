import {
  Clock,
  Flame,
  History,
  Home,
  ListVideo,
  UsersRound,
  type LucideIcon,
} from "lucide-react";

import { routes } from "@/config/routes";

export interface NavItem {
  label: string;
  href: string;
  icon: LucideIcon;
}

/**
 * The primary navigation, declared once so the desktop sidebar and the mobile
 * sheet can never drift apart. Split into "explore" (anonymous-friendly) and
 * "library" (session-backed — the proxy bounces signed-out visitors to login).
 */
export const exploreNav: NavItem[] = [
  { label: "Home", href: routes.home, icon: Home },
  { label: "Trending", href: routes.trending, icon: Flame },
];

export const libraryNav: NavItem[] = [
  { label: "Subscriptions", href: routes.subscriptions, icon: UsersRound },
  { label: "History", href: routes.history, icon: History },
  { label: "Watch Later", href: routes.watchLater, icon: Clock },
  { label: "Playlists", href: routes.playlists, icon: ListVideo },
];
