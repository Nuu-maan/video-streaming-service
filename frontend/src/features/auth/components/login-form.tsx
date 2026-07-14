"use client";

import Link from "next/link";
import { useActionState } from "react";

import { routes } from "@/config/routes";
import { login } from "@/features/auth/actions";
import { FormAlert } from "@/features/auth/components/form-alert";
import { PasswordField } from "@/features/auth/components/password-field";
import { SubmitButton } from "@/features/auth/components/submit-button";
import { TextField } from "@/features/auth/components/text-field";
import { useInvalidFieldFocus } from "@/features/auth/hooks/use-invalid-field-focus";
import { idleAuthFormState } from "@/features/auth/schemas";

/**
 * `next` is the path to return to after signing in — the page reads it from the
 * query string and hands it down, so an anonymous visitor bounced off /studio
 * lands back on /studio. The action validates it before redirecting: it is
 * attacker-writable, and an unchecked value is an open redirect.
 */
export function LoginForm({ next }: { next?: string }) {
  const [state, formAction, pending] = useActionState(login, idleAuthFormState);
  // A failed submit must land the caret on the field that failed — otherwise the
  // error is red text on a screen, and nothing at all to anyone not looking.
  const formRef = useInvalidFieldFocus(state);

  return (
    <form ref={formRef} action={formAction} className="grid gap-4" noValidate>
      {state.error ? <FormAlert tone="error">{state.error}</FormAlert> : null}

      {next ? <input type="hidden" name="next" value={next} /> : null}

      <TextField
        name="identifier"
        label="Username or email"
        /* The API's field is `identifier` and takes either. `autoComplete="username"`
           is still correct — it is the string a password manager stores as the
           account handle, whichever of the two the person typed. */
        autoComplete="username"
        placeholder="you@example.com"
        autoFocus
        error={state.fieldErrors?.identifier}
      />

      <PasswordField
        name="password"
        label="Password"
        autoComplete="current-password"
        error={state.fieldErrors?.password}
        action={
          <Link
            href={routes.forgotPassword}
            className="rounded-sm text-[0.8rem] font-medium text-muted-foreground underline-offset-4 outline-none transition-colors duration-(--motion-fast) hover:text-foreground hover:underline focus-visible:ring-3 focus-visible:ring-ring/50"
          >
            Forgot password?
          </Link>
        }
      />

      <SubmitButton pending={pending}>Sign in</SubmitButton>
    </form>
  );
}
