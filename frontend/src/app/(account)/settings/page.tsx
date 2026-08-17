import type { Metadata } from "next";
import { redirect } from "next/navigation";

import { PageHeader } from "@/components/common/page-header";
import { Separator } from "@/components/ui/separator";
import { routes } from "@/config/routes";
import { getCurrentUser } from "@/features/auth/current-user";

// Route-private components are reached by their sibling path, never through the
// @/ alias: the boundary lint bans the aliased form of an `_components` path
// outright, which is precisely what keeps another route from importing this form.
import { ChangePasswordForm } from "./_components/change-password-form";
import { EmailVerificationCard } from "./_components/email-verification-card";
import { SignOutEverywhere } from "./_components/sign-out-everywhere";

export const metadata: Metadata = { title: "Settings" };

interface SettingsSectionProps {
  title: string;
  description: string;
  children: React.ReactNode;
}

/** Label column on the left, controls on the right — the shape a settings page has had for forty years. */
function SettingsSection({ title, description, children }: SettingsSectionProps) {
  return (
    <section className="grid gap-6 md:grid-cols-[16rem_1fr]">
      <div>
        <h2 className="text-heading">{title}</h2>
        <p className="mt-1 text-sm text-pretty text-muted-foreground">{description}</p>
      </div>
      <div className="min-w-0">{children}</div>
    </section>
  );
}

export default async function SettingsPage() {
  const user = await getCurrentUser();
  if (!user) redirect(routes.login);

  return (
    <>
      <PageHeader title="Settings" description="Your account, your password, your sessions." />

      <div className="flex flex-col gap-10">
        <SettingsSection
          title="Account"
          description="The name and address this account is known by."
        >
          <dl className="flex flex-col gap-3 text-sm">
            <div className="flex gap-3">
              <dt className="w-24 shrink-0 text-muted-foreground">Username</dt>
              <dd className="min-w-0 truncate font-medium">{user.username}</dd>
            </div>
            <div className="flex gap-3">
              <dt className="w-24 shrink-0 text-muted-foreground">Email</dt>
              <dd className="min-w-0 truncate font-medium">{user.email}</dd>
            </div>
          </dl>

          <div className="mt-5">
            <EmailVerificationCard email={user.email} verified={user.email_verified} />
          </div>
        </SettingsSection>

        <Separator />

        <SettingsSection
          title="Password"
          description="Change the password you sign in with. You'll stay signed in here."
        >
          <ChangePasswordForm />
        </SettingsSection>

        <Separator />

        <SettingsSection
          title="Sessions"
          description="Signed in somewhere you shouldn't be? Revoke every token this account holds."
        >
          <SignOutEverywhere />
        </SettingsSection>
      </div>
    </>
  );
}
