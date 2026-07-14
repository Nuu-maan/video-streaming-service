"use client";

import { CircleCheck } from "lucide-react";
import Link from "next/link";
import { useActionState } from "react";

import { Button } from "@/components/ui/button";
import { routes } from "@/config/routes";
import { resetPassword } from "@/features/auth/actions";
import { FormAlert } from "@/features/auth/components/form-alert";
import { PasswordField } from "@/features/auth/components/password-field";
import { SubmitButton } from "@/features/auth/components/submit-button";
import { useInvalidFieldFocus } from "@/features/auth/hooks/use-invalid-field-focus";
import { idleAuthFormState } from "@/features/auth/schemas";

/**
 * The token arrives in `?token=`, is carried in a hidden input, and is
 * re-validated server-side — the action never trusts that it was the one we
 * rendered. An expired or forged token comes back as a form-level error, not a
 * field error, because the field it belongs to is invisible.
 */
export function ResetPasswordForm({ token }: { token: string }) {
  const [state, formAction, pending] = useActionState(resetPassword, idleAuthFormState);
  const formRef = useInvalidFieldFocus(state);

  if (state.status === "success") {
    return (
      <div className="grid animate-in gap-4 fade-in-0 duration-(--motion-medium) ease-out-quart">
        <div className="flex size-11 items-center justify-center rounded-full bg-muted">
          <CircleCheck aria-hidden className="size-5 text-foreground" />
        </div>
        <p className="text-pretty text-muted-foreground">{state.message}</p>
        <Button asChild size="lg" className="mt-1 h-10 w-full">
          <Link href={routes.login}>Sign in</Link>
        </Button>
      </div>
    );
  }

  return (
    <form ref={formRef} action={formAction} className="grid gap-4" noValidate>
      {state.error ? <FormAlert tone="error">{state.error}</FormAlert> : null}

      <input type="hidden" name="token" value={token} />

      {/* No "confirm password" field: the reveal toggle already lets someone
          check what they typed, and a second box mostly collects the same typo
          twice. The single field is the honest one. */}
      <PasswordField
        name="password"
        label="New password"
        autoComplete="new-password"
        autoFocus
        error={state.fieldErrors?.password}
      />

      <SubmitButton pending={pending}>Reset password</SubmitButton>
    </form>
  );
}
