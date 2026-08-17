"use client";

import { LoaderCircle, X } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { removeFromHistory } from "@/features/history/actions";

interface RemoveFromHistoryButtonProps {
  videoId: string;
  videoTitle: string;
}

/**
 * Forgetting one video.
 *
 * Pending, not optimistic. The row is server-rendered and removing it is the
 * kind of thing someone does deliberately, having aimed at a small target; a
 * row that vanishes and then reappears because the request failed is worse than
 * one that waits 200ms and goes.
 */
export function RemoveFromHistoryButton({ videoId, videoTitle }: RemoveFromHistoryButtonProps) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();

  function handle() {
    startTransition(async () => {
      const result = await removeFromHistory(videoId);
      if (!result.ok) {
        toast.error(result.message);
        return;
      }
      router.refresh();
      toast.success(`Removed “${videoTitle}” from your history.`);
    });
  }

  return (
    <Button
      variant="ghost"
      size="icon"
      disabled={pending}
      onClick={handle}
      aria-label={`Remove ${videoTitle} from watch history`}
      /* Revealed on hover, but never hidden from a keyboard (focus-visible) or
         from touch (max-md), where there is no hover to reveal it with. */
      className="size-8 shrink-0 rounded-full text-muted-foreground opacity-0 transition-opacity duration-(--motion-fast) group-hover/row:opacity-100 focus-visible:opacity-100 max-md:opacity-100"
    >
      {pending ? (
        <LoaderCircle aria-hidden className="size-4 animate-spin" />
      ) : (
        <X aria-hidden className="size-4" />
      )}
    </Button>
  );
}
