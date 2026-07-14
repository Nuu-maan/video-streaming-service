"use client";

import { CircleCheck, Loader2, TriangleAlert } from "lucide-react";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { verifyEmail } from "@/features/auth/actions";
import type { AuthFormState } from "@/features/auth/schemas";

/**
 * Redeems the token from `?token=` and reports what happened.
 *
 * Why on mount, in the browser, rather than during the page's server render:
 * the token is single-use, and a GET that consumes it would be spent by
 * anything that merely *looks* at the link — a mail client's safe-link scanner,
 * a chat app's preview fetcher, a Next prefetch. By the time the person clicked,
 * their own link would already be "expired". Redeeming from an effect means it is
 * consumed by a real browser that a real person is looking at.
 *
 * The ref guard is what keeps that true under React's development
 * double-invoke: without it the second run would burn the token the first run
 * just spent, and every verification would render as a failure in dev.
 */
export function VerifyEmailStatus({ token }: { token: string | null }) {
  const [state, setState] = useState<AuthFormState>(
    token ? { status: "idle" } : { status: "error", error: "This verification link is missing its token." },
  );
  const redeemed = useRef(false);

  useEffect(() => {
    if (!token || redeemed.current) return;
    redeemed.current = true;

    let active = true;
    void verifyEmail(token).then((result) => {
      if (active) setState(result);
    });

    return () => {
      active = false;
    };
  }, [token]);

  if (state.status === "idle") {
    return (
      <div className="grid justify-items-center gap-4 py-4 text-center">
        <Loader2 aria-hidden className="size-6 animate-spin text-muted-foreground" />
        <p role="status" className="text-sm text-muted-foreground">
          Verifying your email address…
        </p>
      </div>
    );
  }

  const ok = state.status === "success";
  const Icon = ok ? CircleCheck : TriangleAlert;

  return (
    <div className="grid animate-in justify-items-center gap-4 text-center fade-in-0 duration-(--motion-medium) ease-out-quart">
      <div className="flex size-11 items-center justify-center rounded-full bg-muted">
        <Icon aria-hidden className={ok ? "size-5 text-foreground" : "size-5 text-destructive"} />
      </div>

      <p role="status" className="text-pretty text-muted-foreground">
        {ok ? state.message : state.error}
      </p>

      <Button asChild size="lg" className="mt-1 h-10 w-full">
        <Link href={ok ? routes.home : routes.login}>{ok ? "Start watching" : "Back to sign in"}</Link>
      </Button>
    </div>
  );
}
