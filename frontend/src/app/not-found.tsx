import { Compass, FileQuestion, Search } from "lucide-react";
import type { Metadata } from "next";
import Link from "next/link";

import { EmptyState } from "@/components/common/empty-state";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";

export const metadata: Metadata = {
  title: "Page not found",
  robots: { index: false, follow: false },
};

/**
 * The app-wide 404: any URL that matches no route at all, plus any `notFound()`
 * thrown outside a segment that ships its own not-found (the watch page has one,
 * and it is deliberately worded so a private video cannot be told apart from a
 * missing one).
 *
 * This renders inside the ROOT layout only — a route group's shell is not
 * applied to it — so there is no header and no sidebar here, and the way out has
 * to be on the page itself. Hence the links: a dead end with no exit is the one
 * thing a 404 must never be.
 */
export default function NotFound() {
  return (
    <main id="main-content" className="mx-auto flex w-full max-w-2xl flex-1 items-center px-4 py-16 sm:px-6">
      <EmptyState
        icon={FileQuestion}
        title="Page not found"
        description="That link doesn't lead anywhere. It may have been moved, or it may never have existed."
        className="w-full"
        action={
          <div className="flex flex-wrap items-center justify-center gap-2">
            <Button asChild>
              <Link href={routes.home}>
                <Compass aria-hidden />
                Back to home
              </Link>
            </Button>
            <Button variant="ghost" asChild>
              <Link href={routes.search}>
                <Search aria-hidden />
                Search videos
              </Link>
            </Button>
          </div>
        }
      />
    </main>
  );
}
