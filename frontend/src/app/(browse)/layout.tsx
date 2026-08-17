import { SiteHeader } from "@/components/layout/site-header";
import { SiteSidebar } from "@/components/layout/site-sidebar";
import { signOut } from "@/features/auth/actions";
import { getCurrentUser } from "@/features/auth/current-user";
import { NotificationBell } from "@/features/notifications/components/notification-bell";
import { HeaderSearch } from "@/features/search/components/header-search";

/**
 * The browsing shell: sticky header, desktop rail, content.
 *
 * The layout — not the header — fetches the session, and the feature widgets
 * (sign-out, search, notifications) are threaded in as props because
 * `components/layout` is forbidden from importing features. The boundary lint
 * enforces that, and this is the seam it forces: the shell owns the geometry,
 * the layout owns the product dependencies.
 *
 * The bell is rendered only for a signed-in viewer — there is nothing to notify
 * an anonymous one about — and it takes no initial count on purpose: it fetches
 * its own on mount through a Server Action, so an unread badge costs the shell
 * zero server latency on every page load.
 */
export default async function BrowseLayout({ children }: { children: React.ReactNode }) {
  const user = await getCurrentUser();

  return (
    <div className="flex flex-1 flex-col">
      <SiteHeader
        user={user}
        signOutAction={signOut}
        search={<HeaderSearch />}
        notifications={user ? <NotificationBell /> : null}
      />
      <div className="flex flex-1">
        <SiteSidebar />
        <main id="main-content" className="flex min-w-0 flex-1 flex-col">{children}</main>
      </div>
    </div>
  );
}
