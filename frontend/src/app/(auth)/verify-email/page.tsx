import type { Metadata } from "next";

import { AuthCard } from "@/features/auth/components/auth-card";
import { VerifyEmailStatus } from "@/features/auth/components/verify-email-status";

export const metadata: Metadata = {
  title: "Verify email",
  description: "Confirm your email address.",
  /* The URL carries a single-use token. Nothing about it should be indexed. */
  robots: { index: false, follow: false },
};

function firstValue(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

/**
 * The page reads the token and hands it to a client component, which redeems it
 * on mount. The redemption deliberately does NOT happen here, during the server
 * render: a single-use token consumed on GET would be spent by a mail scanner or
 * a link preview long before the person clicked. See verify-email-status.tsx.
 */
export default async function VerifyEmailPage(props: PageProps<"/verify-email">) {
  const { token } = await props.searchParams;

  return (
    <AuthCard title="Verify your email">
      <VerifyEmailStatus token={firstValue(token) ?? null} />
    </AuthCard>
  );
}
