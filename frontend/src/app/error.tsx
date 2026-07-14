"use client";

import { RotateCw } from "lucide-react";
import Link from "next/link";
import { useEffect } from "react";

import { ErrorState } from "@/components/common/error-state";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";

/**
 * The app's root error boundary.
 *
 * Without this, an unhandled throw in ANY page or layout that does not ship its
 * own error.tsx produces Next's bare 500 — no chrome, no copy, no way out. That
 * was not hypothetical: a rate-limited `GET /auth/me` escaping `getCurrentUser`
 * did exactly that on every route in the app. That specific bug is fixed at
 * source (see `features/auth/current-user.ts`), but the boundary is what makes
 * the NEXT one survivable.
 *
 * The copy is deliberately generic, and that is a constraint rather than a
 * choice. React strips a server error's message in production and hands this
 * boundary only a `digest`, so there is genuinely no way to tell a rate limit
 * from a crash from a bad gateway HERE. Anything more specific would be a guess
 * dressed up as a diagnosis — and guessing wrong is how a user ends up waiting
 * out a "slow down" that was really an outage. The surfaces that CAN tell the
 * difference — the ones holding the ApiError, on the server — say "slow down"
 * precisely, in place, and never reach this boundary at all.
 *
 * `reset()` re-renders the segment. For the transient failures that dominate
 * here (a timeout, a rate limit, a blip) that genuinely fixes it, which is why
 * it is the primary action.
 */
export default function AppError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // The server already logged the real error with its stack. This is the
    // client's view of it — the digest is the string that ties the two together.
    console.error("Unhandled application error", error.digest ?? error);
  }, [error]);

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-1 items-center px-4 py-16 sm:px-6">
      <ErrorState
        title="Something went wrong"
        description="This page didn't load. It's usually temporary — try again, and if it keeps happening, give it a few minutes."
        className="w-full"
        action={
          <div className="flex flex-col items-center gap-4">
            <div className="flex flex-wrap items-center justify-center gap-2">
              <Button onClick={reset}>
                <RotateCw aria-hidden />
                Try again
              </Button>
              <Button variant="ghost" asChild>
                <Link href={routes.home}>Back to home</Link>
              </Button>
            </div>
            {error.digest ? (
              <p className="font-mono text-xs text-muted-foreground/70">
                Reference: <span className="tabular-nums">{error.digest}</span>
              </p>
            ) : null}
          </div>
        }
      />
    </div>
  );
}
