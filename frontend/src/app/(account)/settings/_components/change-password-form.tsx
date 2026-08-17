"use client";

import { LoaderCircle } from "lucide-react";
import { useActionState, useEffect, useRef } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { changePassword } from "@/features/auth/actions";
import { idleAuthFormState } from "@/features/auth/schemas";
import { cn } from "@/lib/utils";

/**
 * Password strength is the API's policy, not ours. It answers a weak password
 * with a precise 400, and that message is what a person reads — inventing a
 * second policy here would eventually reject a password the server accepts.
 */
export function ChangePasswordForm() {
  const [state, formAction, pending] = useActionState(changePassword, idleAuthFormState);
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    if (state.status !== "success") return;
    toast.success(state.message ?? "Password changed.");
    formRef.current?.reset();
  }, [state]);

  const currentError = state.fieldErrors?.current_password;
  const newError = state.fieldErrors?.new_password;

  return (
    <form ref={formRef} action={formAction} className="flex max-w-sm flex-col gap-4">
      <div className="flex flex-col gap-2">
        <Label htmlFor="current_password">Current password</Label>
        <Input
          id="current_password"
          name="current_password"
          type="password"
          autoComplete="current-password"
          required
          aria-invalid={Boolean(currentError)}
          aria-describedby={currentError ? "current_password-error" : undefined}
        />
        {currentError ? (
          <p id="current_password-error" className="text-xs text-destructive">
            {currentError}
          </p>
        ) : null}
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor="new_password">New password</Label>
        <Input
          id="new_password"
          name="new_password"
          type="password"
          autoComplete="new-password"
          required
          aria-invalid={Boolean(newError)}
          aria-describedby={newError ? "new_password-error" : undefined}
        />
        {newError ? (
          <p id="new_password-error" className="text-xs text-destructive">
            {newError}
          </p>
        ) : null}
      </div>

      {state.status === "error" && state.error ? (
        <p role="alert" className={cn("text-sm text-destructive")}>
          {state.error}
        </p>
      ) : null}

      <div>
        <Button type="submit" disabled={pending}>
          {pending ? <LoaderCircle aria-hidden className="animate-spin" /> : null}
          Change password
        </Button>
      </div>
    </form>
  );
}
