"use client";

import { useId } from "react";

import { FieldError } from "@/features/auth/components/field-error";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface TextFieldProps {
  name: string;
  label: string;
  type?: "text" | "email";
  /**
   * `username` on the login identifier and the register username, `email` on an
   * address. These are the strings a password manager matches on; a wrong value
   * here is the difference between one tap to sign in and typing it all out.
   */
  autoComplete: "username" | "email";
  autoFocus?: boolean;
  placeholder?: string;
  hint?: string;
  error?: string;
  defaultValue?: string;
}

export function TextField({
  name,
  label,
  type = "text",
  autoComplete,
  autoFocus,
  placeholder,
  hint,
  error,
  defaultValue,
}: TextFieldProps) {
  const id = useId();
  const errorId = `${id}-error`;
  const hintId = `${id}-hint`;

  return (
    <div className="grid gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        name={name}
        type={type}
        autoComplete={autoComplete}
        autoFocus={autoFocus}
        placeholder={placeholder}
        defaultValue={defaultValue}
        required
        /* Nothing in an auth form benefits from the OS capitalising or
           autocorrecting it — a "corrected" username is just a wrong one. */
        autoCapitalize="none"
        autoCorrect="off"
        spellCheck={false}
        aria-invalid={error ? true : undefined}
        aria-describedby={[error && errorId, hint && hintId].filter(Boolean).join(" ") || undefined}
        className="h-10"
      />
      {hint && !error ? (
        <p id={hintId} className="text-[0.8rem] leading-snug text-muted-foreground">
          {hint}
        </p>
      ) : null}
      <FieldError id={errorId} message={error} />
    </div>
  );
}
