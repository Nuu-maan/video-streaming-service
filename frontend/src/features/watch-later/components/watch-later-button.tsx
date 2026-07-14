"use client";

import { Check, Clock } from "lucide-react";
import { useOptimistic, useState, useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { toggleWatchLater } from "@/features/watch-later/actions";
import { useSignInPrompt } from "@/hooks/use-sign-in-prompt";
import { cn } from "@/lib/utils";

interface WatchLaterButtonProps {
  videoId: string;
  initialSaved: boolean;
  isAuthenticated: boolean;
  /** "icon" is the compact overlay used on cards; "button" carries a label. */
  variant?: "button" | "icon";
  className?: string;
}

/**
 * Both directions are idempotent server-side (PUT / DELETE), so an optimistic
 * flip is safe: a double click that races itself converges on the same row.
 */
export function WatchLaterButton({
  videoId,
  initialSaved,
  isAuthenticated,
  variant = "button",
  className,
}: WatchLaterButtonProps) {
  const [saved, setSaved] = useState(initialSaved);
  const [optimisticSaved, setOptimisticSaved] = useOptimistic(saved);
  const [, startTransition] = useTransition();
  const promptSignIn = useSignInPrompt();

  function handle() {
    if (!isAuthenticated) {
      promptSignIn("Sign in to save videos for later.");
      return;
    }

    startTransition(async () => {
      setOptimisticSaved(!saved);
      const result = await toggleWatchLater(videoId, saved);

      if (result.ok) {
        setSaved(result.saved);
        toast.success(result.saved ? "Saved to Watch later." : "Removed from Watch later.");
        return;
      }
      toast.error(result.message);
    });
  }

  const label = optimisticSaved ? "Remove from Watch later" : "Save to Watch later";
  const Icon = optimisticSaved ? Check : Clock;

  if (variant === "icon") {
    return (
      <Button
        type="button"
        variant="secondary"
        size="icon"
        aria-label={label}
        title={label}
        onClick={handle}
        className={cn("rounded-full shadow-sm", className)}
      >
        <Icon aria-hidden />
      </Button>
    );
  }

  return (
    <Button
      type="button"
      variant={optimisticSaved ? "secondary" : "outline"}
      size="sm"
      aria-pressed={optimisticSaved}
      onClick={handle}
      className={cn("rounded-full", className)}
    >
      <Icon aria-hidden />
      {optimisticSaved ? "Saved" : "Save"}
    </Button>
  );
}
