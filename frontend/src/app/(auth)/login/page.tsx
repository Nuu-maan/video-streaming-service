import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";

import { routes } from "@/config/routes";
import { AuthCard } from "@/features/auth/components/auth-card";
import { LoginForm } from "@/features/auth/components/login-form";
import { getCurrentUser } from "@/features/auth/current-user";

export const metadata: Metadata = {
  title: "Sign in",
  description: "Sign in to your account.",
};

/** `?next=` may arrive repeated (`?next=/a&next=/b`); take the first, never the array. */
function firstValue(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

export default async function LoginPage(props: PageProps<"/login">) {
  const { next } = await props.searchParams;
  const target = firstValue(next);

  /* Somebody who is already signed in has no business on a sign-in form; send
     them where they were going. The action re-validates `next` before it
     redirects — this path does not, because it never leaves the origin. */
  const user = await getCurrentUser();
  if (user) {
    redirect(target?.startsWith("/") && !target.startsWith("//") ? target : routes.home);
  }

  return (
    <AuthCard
      title="Welcome back"
      description="Sign in to keep watching, comment, and upload."
      footer={
        <>
          New here?{" "}
          <Link
            href={target ? `${routes.register}?next=${encodeURIComponent(target)}` : routes.register}
            className="rounded-sm font-medium text-foreground underline-offset-4 outline-none hover:underline focus-visible:ring-3 focus-visible:ring-ring/50"
          >
            Create an account
          </Link>
        </>
      }
    >
      <LoginForm next={target} />
    </AuthCard>
  );
}
