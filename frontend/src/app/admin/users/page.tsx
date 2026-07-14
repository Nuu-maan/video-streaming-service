import { Info } from "lucide-react";

import { PageHeader } from "@/components/common/page-header";
import { BanUserForm } from "@/features/admin/components/ban-user-form";
import { Panel } from "@/features/admin/components/panel";
import { UnbanUserForm } from "@/features/admin/components/unban-user-form";

export const metadata = { title: "Users" };

/** `?user=` is attacker-writable and only ever prefills a field the API re-validates. */
function firstParam(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

/**
 * The ban console.
 *
 * This page is keyed by user ID rather than being a browsable directory, and
 * that is a constraint of the API, not a design choice: there is no
 * `GET /admin/users`, and no `GET /users/{id}` either. The only user-shaped
 * endpoints are `ban` and `unban`, both addressed by ID. Inventing a user table
 * here would mean inventing the data to fill it.
 *
 * So the honest design is a console that takes the ID from where the IDs
 * actually come from — the reports queue, which links straight here with
 * `?user=` already filled in — and says plainly that it cannot list accounts.
 * A moderator arriving from a report never types a UUID; one arriving cold is
 * told why they have to.
 */
export default async function AdminUsersPage(props: PageProps<"/admin/users">) {
  const searchParams = await props.searchParams;
  const userId = firstParam(searchParams.user);

  return (
    <>
      <PageHeader
        title="Users"
        description="Ban an account, or lift a ban. Both take effect immediately."
      />

      <div className="flex items-start gap-3 rounded-xl border border-dashed border-border/70 p-4 text-sm text-muted-foreground">
        <Info aria-hidden className="mt-0.5 size-4 shrink-0" />
        <p className="text-pretty">
          The API has no endpoint for listing or looking up accounts, so this console works by user
          ID. The moderation queue links here with the ID already filled in — open a report and
          choose <span className="font-medium text-foreground">Open in ban console</span>.
        </p>
      </div>

      <div className="grid items-start gap-6 lg:grid-cols-2">
        <Panel
          title="Ban an account"
          description="They're signed out of every session and blocked from signing in. Reversible."
        >
          <BanUserForm defaultUserId={userId} />
        </Panel>

        <Panel title="Lift a ban" description="Restores access immediately.">
          <UnbanUserForm defaultUserId={userId} />
        </Panel>
      </div>
    </>
  );
}
