import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { SiteHeader } from "@/components/layout/site-header";
import { AdminNav } from "@/features/admin/components/admin-nav";
import { signOut } from "@/features/auth/actions";
import { getCurrentUser } from "@/features/auth/current-user";
import { NotificationBell } from "@/features/notifications/components/notification-bell";
import { HeaderSearch } from "@/features/search/components/header-search";

export const metadata: Metadata = {
  title: { default: "Admin", template: "%s · Admin" },
  // An admin console has no business in a search index, and `noindex` is free.
  robots: { index: false, follow: false },
};

/**
 * The admin area.
 *
 * A non-admin gets `notFound()`, not `redirect()`, and the difference matters.
 * A redirect to `/login` — or worse, a "you don't have permission" page —
 * confirms that `/admin` is a real route with something behind it, which is a
 * free reconnaissance win for anyone probing the app. A 404 says only what an
 * unauthenticated stranger already knows: there is nothing here for you. It is
 * the same reasoning the API applies to private videos, which answer 404 rather
 * than 403 so that "not yours" and "not there" are indistinguishable.
 *
 * This is defence in depth, not the lock. The API independently enforces
 * `view_analytics` / `moderate_content` / `manage_users` on every endpoint
 * underneath and 403s without them, and every Server Action re-checks the role
 * because an action is a public POST that never routes through this layout. If
 * this check were the only gate, it would be trivially bypassed by calling the
 * actions directly.
 */
export default async function AdminLayout({ children }: LayoutProps<"/admin">) {
  const user = await getCurrentUser();

  if (user?.role !== "admin") {
    notFound();
  }

  return (
    <div className="flex flex-1 flex-col">
      <SiteHeader
        user={user}
        signOutAction={signOut}
        search={<HeaderSearch />}
        notifications={<NotificationBell />}
      />
      <AdminNav />
      <main id="main-content" className="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 px-4 py-8 sm:px-6">
        {children}
      </main>
    </div>
  );
}
