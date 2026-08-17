import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";

import { routes } from "@/config/routes";
import { AuthCard } from "@/features/auth/components/auth-card";
import { RegisterForm } from "@/features/auth/components/register-form";
import { getCurrentUser } from "@/features/auth/current-user";

export const metadata: Metadata = {
  title: "Create an account",
  description: "Create an account to upload and stream video.",
};

function firstValue(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

export default async function RegisterPage(props: PageProps<"/register">) {
  const { next } = await props.searchParams;
  const target = firstValue(next);

  const user = await getCurrentUser();
  if (user) {
    redirect(target?.startsWith("/") && !target.startsWith("//") ? target : routes.home);
  }

  return (
    <AuthCard
      title="Create your account"
      description="Upload, transcode, and stream in a few minutes."
      footer={
        <>
          Already have an account?{" "}
          <Link
            href={target ? `${routes.login}?next=${encodeURIComponent(target)}` : routes.login}
            className="rounded-sm font-medium text-foreground underline-offset-4 outline-none hover:underline focus-visible:ring-3 focus-visible:ring-ring/50"
          >
            Sign in
          </Link>
        </>
      }
    >
      <RegisterForm next={target} />
    </AuthCard>
  );
}
