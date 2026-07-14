"use client";

import { Bell, BellOff } from "lucide-react";
import { useOptimistic, useState, useTransition } from "react";
import { toast } from "sonner";

import { IconSwap } from "@/components/common/icon-swap";
import { Button } from "@/components/ui/button";
import { toggleSubscription } from "@/features/subscriptions/actions";
import { useSignInPrompt } from "@/hooks/use-sign-in-prompt";
import { cn } from "@/lib/utils";

interface SubscribeButtonProps {
  channelId: string;
  channelName?: string;
  initialSubscribed: boolean;
  isAuthenticated: boolean;
  /** True on your own channel — the API 400s a self-subscribe, so we never offer it. */
  isSelf?: boolean;
  size?: "sm" | "default";
  className?: string;
}

/**
 * Optimistic both ways. Unsubscribing is the destructive direction, so it gets
 * the quieter treatment: a subscribed button is a muted pill, an unsubscribed
 * one is the solid primary — the state you are *not* in is the one being
 * offered, which is the whole job of the control.
 */
export function SubscribeButton({
  channelId,
  channelName,
  initialSubscribed,
  isAuthenticated,
  isSelf = false,
  size = "default",
  className,
}: SubscribeButtonProps) {
  const [subscribed, setSubscribed] = useState(initialSubscribed);
  const [optimisticSubscribed, setOptimisticSubscribed] = useOptimistic(subscribed);
  const [, startTransition] = useTransition();
  const promptSignIn = useSignInPrompt();

  // Rendering nothing beats rendering a button that can only ever fail.
  if (isSelf) return null;

  function handle() {
    if (!isAuthenticated) {
      promptSignIn("Sign in to subscribe to channels.");
      return;
    }

    startTransition(async () => {
      setOptimisticSubscribed(!subscribed);
      const result = await toggleSubscription(channelId, subscribed);

      if (result.ok) {
        setSubscribed(result.subscribed);
        if (!result.subscribed && channelName) {
          toast.success(`Unsubscribed from ${channelName}.`);
        }
        return;
      }
      toast.error(result.message);
    });
  }

  return (
    <Button
      type="button"
      size={size}
      variant={optimisticSubscribed ? "secondary" : "default"}
      aria-pressed={optimisticSubscribed}
      onClick={handle}
      className={cn("rounded-full", className)}
    >
      {/* Subscribing is rare, deliberate, and carries a state the person wants
          confirmed — which is precisely the swap that earns the cross-fade. The
          bell used to hard-cut. */}
      <IconSwap
        active={optimisticSubscribed}
        from={<Bell aria-hidden className="size-4" />}
        to={<BellOff aria-hidden className="size-4" />}
      />
      {optimisticSubscribed ? "Subscribed" : "Subscribe"}
    </Button>
  );
}
