"use client";

import { LoaderCircle, RefreshCw } from "lucide-react";
import { useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { retryVideo } from "@/features/admin/actions";

/**
 * Re-queue one failed transcode.
 *
 * No confirmation dialog, deliberately. A retry destroys nothing and loses
 * nothing — the worst case is that the video fails again and the row is still
 * here. Putting a dialog in front of it would be exactly the overuse that
 * trains people to click through the dialogs that *do* matter, like the delete
 * two panels up.
 *
 * `useTransition` rather than a local boolean, so the pending state covers the
 * server action *and* the revalidation that follows it. A plain `useState`
 * would flip back to idle the moment the action resolved, leaving a live-looking
 * button sitting above a list that has not refreshed yet.
 */
export function RetryVideoButton({ videoId, title }: { videoId: string; title: string }) {
  const [pending, startTransition] = useTransition();

  function retry() {
    startTransition(async () => {
      const result = await retryVideo(videoId);
      if (result.ok) {
        toast.success(result.message, { description: title });
        return;
      }
      toast.error("Couldn't retry that transcode", { description: result.message });
    });
  }

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={retry}
      disabled={pending}
      aria-label={`Retry transcoding “${title}”`}
    >
      {pending ? (
        <LoaderCircle aria-hidden data-icon="inline-start" className="animate-spin" />
      ) : (
        <RefreshCw aria-hidden data-icon="inline-start" />
      )}
      Retry
    </Button>
  );
}
