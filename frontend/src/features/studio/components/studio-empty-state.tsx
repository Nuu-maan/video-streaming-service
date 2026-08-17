import { Clapperboard, Upload } from "lucide-react";
import Link from "next/link";

import { EmptyState } from "@/components/common/empty-state";
import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";

/**
 * A creator with nothing published yet. The only useful thing this screen can
 * do is get them to the one action that changes it — so the CTA is the point,
 * and everything else is there to make pressing it feel obvious.
 */
export function StudioEmptyState() {
  return (
    <EmptyState
      icon={Clapperboard}
      title="Your studio is empty"
      description="Upload your first video and we'll transcode it into every quality your viewers need — 360p through 1080p, streamed adaptively."
      action={
        <Button asChild size="lg">
          <Link href={routes.upload}>
            <Upload aria-hidden data-icon="inline-start" />
            Upload a video
          </Link>
        </Button>
      }
    />
  );
}
