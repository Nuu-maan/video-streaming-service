import type { Metadata } from "next";
import Link from "next/link";

import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { AuthCard } from "@/features/auth/components/auth-card";
import { ResetPasswordForm } from "@/features/auth/components/reset-password-form";

export const metadata: Metadata = {
  title: "Set a new password",
  description: "Choose a new password for your account.",
  /* A reset link must never be indexed, previewed, or archived with its token. */
  robots: { index: false, follow: false },
};

function firstValue(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

export default async function ResetPasswordPage(props: PageProps<"/reset-password">) {
  const { token } = await props.searchParams;
  const value = firstValue(token);

  /* No token means the link was mangled in transit (mail clients do this), not
     that the person did anything wrong. Say what to do next rather than
     rendering a form whose submission cannot possibly succeed. */
  if (!value) {
    return (
      <AuthCard
        title="This link isn't complete"
        description="The reset link is missing its token — it may have been broken by your email client. Request a fresh one and open it directly."
      >
        <Button asChild size="lg" className="h-10 w-full">
          <Link href={routes.forgotPassword}>Request a new link</Link>
        </Button>
      </AuthCard>
    );
  }

  return (
    <AuthCard title="Set a new password" description="Choose a password you haven't used here before.">
      <ResetPasswordForm token={value} />
    </AuthCard>
  );
}
