import { redirect } from "next/navigation";

import { SiteHeader } from "@/components/layout/site-header";
import { SiteSidebar } from "@/components/layout/site-sidebar";
import { routes } from "@/config/routes";
import { signOut } from "@/features/auth/actions";
import { getCurrentUser } from "@/features/auth/current-user";
import { NotificationBell } from "@/features/notifications/components/notification-bell";
import { HeaderSearch } from "@/features/search/components/header-search";

/**
 * Everything under (account) is the signed-in library: playlists, watch later,
 * subscriptions, notifications, settings.
 *
 * The session is re-checked here, on the server, on every request. `proxy.ts`
 * already bounces anonymous visitors away from these paths, but that is a
 * courtesy — it exists so nobody watches an empty shell render before being
 * told to sign in. It is not the control. Next's own docs are explicit that a
 * proxy matcher must not be treated as authorization, and the actions behind
 * these pages check again anyway.
 */
export default async function AccountLayout({ children }: { children: React.ReactNode }) {
  const user = await getCurrentUser();
  if (!user) redirect(routes.login);

  return (
    <div className="flex flex-1 flex-col">
      <SiteHeader
        user={user}
        signOutAction={signOut}
        search={<HeaderSearch />}
        notifications={<NotificationBell />}
      />
      <div className="flex flex-1">
        <SiteSidebar />
        <main id="main-content" className="mx-auto flex w-full min-w-0 max-w-6xl flex-1 flex-col gap-8 px-4 py-8 sm:px-6">
          {children}
        </main>
      </div>
    </div>
  );
}
