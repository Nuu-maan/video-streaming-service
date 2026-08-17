import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { SiteHeader } from "@/components/layout/site-header";
import { routes } from "@/config/routes";
import { signOut } from "@/features/auth/actions";
import { getCurrentUser } from "@/features/auth/current-user";
import { NotificationBell } from "@/features/notifications/components/notification-bell";
import { HeaderSearch } from "@/features/search/components/header-search";
import { StudioNav } from "@/features/studio/components/studio-nav";

export const metadata: Metadata = {
  title: { default: "Studio", template: "%s · Studio" },
};

/**
 * The creator's half of the app.
 *
 * Auth is re-checked here even though `proxy.ts` already bounces anonymous
 * visitors away from `/studio`. That is not belt-and-braces paranoia: Next's
 * own docs say the proxy matcher is a redirect, not a security control, and a
 * matcher is exactly the kind of thing a later refactor silently breaks. The
 * layout is where the session is actually required, so the layout checks.
 *
 * (Every Server Action underneath re-checks again, because a Server Action is
 * a public POST endpoint that does not pass through this layout at all.)
 */
export default async function StudioLayout({ children }: LayoutProps<"/studio">) {
  const user = await getCurrentUser();
  if (!user) {
    redirect(`${routes.login}?next=${encodeURIComponent(routes.studio)}`);
  }

  return (
    <div className="flex flex-1 flex-col">
      <SiteHeader
        user={user}
        signOutAction={signOut}
        search={<HeaderSearch />}
        notifications={<NotificationBell />}
      />
      <StudioNav />
      <main id="main-content" className="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 px-4 py-8 sm:px-6">
        {children}
      </main>
    </div>
  );
}
