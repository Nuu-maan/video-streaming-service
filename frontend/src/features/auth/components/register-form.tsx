"use client";

import { useActionState } from "react";

import { register } from "@/features/auth/actions";
import { FormAlert } from "@/features/auth/components/form-alert";
import { PasswordField } from "@/features/auth/components/password-field";
import { SubmitButton } from "@/features/auth/components/submit-button";
import { TextField } from "@/features/auth/components/text-field";
import { useInvalidFieldFocus } from "@/features/auth/hooks/use-invalid-field-focus";
import { idleAuthFormState } from "@/features/auth/schemas";

export function RegisterForm({ next }: { next?: string }) {
  const [state, formAction, pending] = useActionState(register, idleAuthFormState);
  const formRef = useInvalidFieldFocus(state);

  return (
    <form ref={formRef} action={formAction} className="grid gap-4" noValidate>
      {state.error ? <FormAlert tone="error">{state.error}</FormAlert> : null}

      {next ? <input type="hidden" name="next" value={next} /> : null}

      <TextField
        name="username"
        label="Username"
        autoComplete="username"
        placeholder="filmmaker_92"
        hint="Letters, numbers and underscores. 3–30 characters."
        autoFocus
        error={state.fieldErrors?.username}
      />

      <TextField
        name="email"
        label="Email"
        type="email"
        autoComplete="email"
        placeholder="you@example.com"
        error={state.fieldErrors?.email}
      />

      <PasswordField
        name="password"
        label="Password"
        /* `new-password` is what tells a password manager to *offer to generate*
           one, rather than autofilling the password for some other account. */
        autoComplete="new-password"
        error={state.fieldErrors?.password}
      />

      <SubmitButton pending={pending}>Create account</SubmitButton>
    </form>
  );
}
