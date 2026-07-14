"use client";

import { MailCheck } from "lucide-react";
import Link from "next/link";
import { useActionState } from "react";

import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { forgotPassword } from "@/features/auth/actions";
import { FormAlert } from "@/features/auth/components/form-alert";
import { SubmitButton } from "@/features/auth/components/submit-button";
import { TextField } from "@/features/auth/components/text-field";
import { idleAuthFormState } from "@/features/auth/schemas";

export function ForgotPasswordForm() {
  const [state, formAction, pending] = useActionState(forgotPassword, idleAuthFormState);

  /**
   * On success the form is replaced, not merely annotated. Leaving a filled-in
   * email field under a "check your inbox" banner invites a second submission
   * against a 10/min budget, and gives no sense that the flow moved on.
   *
   * The copy never confirms that the address has an account — the API is
   * deliberately unenumerable, and a UI that says "sent!" for a real address and
   * something else for a fake one would undo that in one afternoon.
   */
  if (state.status === "success") {
    return (
      <div className="grid animate-in gap-4 fade-in-0 duration-(--motion-medium) ease-out-quart">
        <div className="flex size-11 items-center justify-center rounded-full bg-muted">
          <MailCheck aria-hidden className="size-5 text-foreground" />
        </div>
        <p className="text-pretty text-muted-foreground">{state.message}</p>
        <Button asChild variant="outline" size="lg" className="mt-1 h-10 w-full">
          <Link href={routes.login}>Back to sign in</Link>
        </Button>
      </div>
    );
  }

  return (
    <form action={formAction} className="grid gap-4" noValidate>
      {state.error ? <FormAlert tone="error">{state.error}</FormAlert> : null}

      <TextField
        name="email"
        label="Email"
        type="email"
        autoComplete="email"
        placeholder="you@example.com"
        autoFocus
        error={state.fieldErrors?.email}
      />

      <SubmitButton pending={pending}>Send reset link</SubmitButton>
    </form>
  );
}
