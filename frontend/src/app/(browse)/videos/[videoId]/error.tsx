"use client";

import { RotateCcw } from "lucide-react";
import Link from "next/link";
import { useEffect } from "react";

import { ErrorState } from "@/components/common/error-state";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";

/**
 * The watch page fell over.
 *
 * Not a 404 — `notFound()` routes to not-found.tsx, and a private video is a 404
 * by design. This is the genuinely unexpected: the API is down, the network
 * blinked, a 500 came back. So it offers the two things that actually help —
 * retry the render, or leave — and never speculates about permissions.
 *
 * A rate limit gets its own words. "Something went wrong" is a lie when the
 * server told us exactly what was wrong: too fast.
 */
export default function WatchError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => {
    // The digest is the only handle on the server-side stack, which was
    // deliberately stripped before it reached the browser.
    console.error("watch page failed", error);
  }, [error]);

  const rateLimited = /rate limit|too many|slow down/i.test(error.message);

  return (
    <div className="mx-auto w-full max-w-2xl px-4 py-16 sm:px-6">
      <ErrorState
        title={rateLimited ? "Slow down a moment" : "This video wouldn't load"}
        description={
          rateLimited
            ? "You're making requests faster than the server allows. Give it a few seconds and try again."
            : "Something went wrong on our side while loading this page. It wasn't anything you did."
        }
        action={
          <div className="flex flex-wrap items-center justify-center gap-2">
            <Button onClick={reset}>
              <RotateCcw aria-hidden />
              Try again
            </Button>
            <Button variant="ghost" asChild>
              <Link href={routes.home}>Back to home</Link>
            </Button>
          </div>
        }
      />
    </div>
  );
}
