import {
  Bookmark,
  History,
  Library,
  LayoutGrid,
  UsersRound,
  Video,
  type LucideIcon,
} from "lucide-react";

import { routes } from "@/config/routes";

export interface NavItem {
  label: string;
  href: string;
  icon: LucideIcon;
}

/**
 * The primary navigation, declared once so the desktop rail and the mobile
 * sheet can never drift apart.
 *
 * There is no "Explore" group and no Trending entry. This is a workspace: the
 * things in it are things somebody put there, and every destination below is a
 * finite list with an end. A ranked shelf of what strangers are watching would
 * be the one screen here that nobody owns.
 */
export const workspaceNav: NavItem[] = [
  { label: "Overview", href: routes.home, icon: LayoutGrid },
  { label: "All videos", href: routes.videos, icon: Video },
  { label: "Collections", href: routes.collections, icon: Library },
];

export const libraryNav: NavItem[] = [
  { label: "Saved", href: routes.saved, icon: Bookmark },
  { label: "History", href: routes.history, icon: History },
  { label: "People", href: routes.people, icon: UsersRound },
];
