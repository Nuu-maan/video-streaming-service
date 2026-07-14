import type { Metadata } from "next";
import Link from "next/link";

import { routes } from "@/config/routes";
import { AuthCard } from "@/features/auth/components/auth-card";
import { ForgotPasswordForm } from "@/features/auth/components/forgot-password-form";

export const metadata: Metadata = {
  title: "Forgot password",
  description: "Send yourself a link to reset your password.",
};

export default function ForgotPasswordPage() {
  return (
    <AuthCard
      title="Reset your password"
      description="Enter the email on your account and we'll send a link to set a new password."
      footer={
        <>
          Remembered it?{" "}
          <Link
            href={routes.login}
            className="rounded-sm font-medium text-foreground underline-offset-4 outline-none hover:underline focus-visible:ring-3 focus-visible:ring-ring/50"
          >
            Sign in
          </Link>
        </>
      }
    >
      <ForgotPasswordForm />
    </AuthCard>
  );
}
