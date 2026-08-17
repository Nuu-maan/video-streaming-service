"use client";

import { BadgeCheck, LoaderCircle, MailWarning } from "lucide-react";
import { useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { sendVerificationEmail } from "@/features/auth/actions";

interface EmailVerificationCardProps {
  email: string;
  verified: boolean;
}

export function EmailVerificationCard({ email, verified }: EmailVerificationCardProps) {
  const [pending, startTransition] = useTransition();

  function handleResend() {
    startTransition(async () => {
      const result = await sendVerificationEmail();
      if (result.status === "success") {
        toast.success(result.message ?? "Verification email sent.");
        return;
      }
      toast.error(result.error ?? "We couldn't send that email. Try again.");
    });
  }

  if (verified) {
    return (
      <p className="flex items-center gap-2 text-sm text-muted-foreground">
        <BadgeCheck aria-hidden className="size-4 text-brand-500" />
        <span>
          <span className="font-medium text-foreground">{email}</span> is verified.
        </span>
      </p>
    );
  }

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl bg-muted/50 px-4 py-3 ring-1 ring-border/60 ring-inset">
      <p className="flex items-center gap-2 text-sm text-pretty text-muted-foreground">
        <MailWarning aria-hidden className="size-4 shrink-0 text-amber-500" />
        <span>
          <span className="font-medium text-foreground">{email}</span> is not verified yet.
        </span>
      </p>
      <Button variant="secondary" size="sm" disabled={pending} onClick={handleResend}>
        {pending ? <LoaderCircle aria-hidden className="animate-spin" /> : null}
        Send verification email
      </Button>
    </div>
  );
}
