import { FileQuestion, Search } from "lucide-react";
import Link from "next/link";

import { EmptyState } from "@/components/common/empty-state";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";

/**
 * "Video not found" — and nothing more.
 *
 * The API answers a private video with 404, exactly as it answers a video that
 * never existed, so that its existence cannot be probed. This page is where that
 * promise is either kept or broken. It would be easy, and friendly-seeming, to
 * write "you may not have permission to view this" — and it would leak the very
 * thing the 404 was protecting, while being wrong half the time. So: not found.
 */
export default function VideoNotFound() {
  return (
    <div className="mx-auto w-full max-w-2xl px-4 py-16 sm:px-6">
      <EmptyState
        icon={FileQuestion}
        title="Video not found"
        description="This video doesn't exist, or it isn't available. Check the link, or find something else to watch."
        action={
          <div className="flex flex-wrap items-center justify-center gap-2">
            <Button asChild>
              <Link href={routes.home}>Back to home</Link>
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
    </div>
  );
}
