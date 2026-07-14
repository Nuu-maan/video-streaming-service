"use client";

import { Eye, EyeOff } from "lucide-react";
import { useId, useState } from "react";

import { IconSwap } from "@/components/common/icon-swap";
import { FieldError } from "@/features/auth/components/field-error";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface PasswordFieldProps {
  name: string;
  label: string;
  /**
   * `current-password` when signing in, `new-password` when registering or
   * resetting. Getting this wrong is what makes a password manager offer to
   * fill an old password into a "choose a new one" box — or offer to save a
   * password that was only being confirmed.
   */
  autoComplete: "current-password" | "new-password";
  autoFocus?: boolean;
  error?: string;
  /** Rendered opposite the label — where "Forgot password?" goes on the login form. */
  action?: React.ReactNode;
  hint?: string;
}

/**
 * A password input with a reveal toggle.
 *
 * The toggle is a real <button> inside the field, not an overlay: it is
 * reachable by keyboard, it announces its state, and it never steals the
 * click target from the input. Toggling does not blur the input or lose the
 * caret, so someone can reveal what they typed and keep typing.
 */
export function PasswordField({
  name,
  label,
  autoComplete,
  autoFocus,
  error,
  action,
  hint,
}: PasswordFieldProps) {
  const [visible, setVisible] = useState(false);
  const id = useId();
  const errorId = `${id}-error`;
  const hintId = `${id}-hint`;

  return (
    <div className="grid gap-1.5">
      <div className="flex items-baseline justify-between gap-2">
        <Label htmlFor={id}>{label}</Label>
        {action}
      </div>

      <div className="relative">
        <Input
          id={id}
          name={name}
          type={visible ? "text" : "password"}
          autoComplete={autoComplete}
          autoFocus={autoFocus}
          required
          aria-invalid={error ? true : undefined}
          aria-describedby={[error && errorId, hint && hintId].filter(Boolean).join(" ") || undefined}
          className="h-10 pr-10"
        />

        {/* In the natural tab order, deliberately. It used to carry tabIndex={-1}
            to save the sighted mouse user one stop on the way to Submit — which
            is a trade made against exactly the person who needs this control
            most: someone who cannot see what they typed and is not using a
            mouse. One extra tab stop is a small price for the only control that
            makes the field auditable. */}
        <button
          type="button"
          onClick={() => setVisible((shown) => !shown)}
          aria-label={visible ? "Hide password" : "Show password"}
          aria-pressed={visible}
          className="absolute inset-y-0 right-0 flex w-10 items-center justify-center rounded-r-lg text-muted-foreground outline-none transition-[color,scale] duration-(--motion-fast) ease-out-quart hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50 active:scale-[0.96]"
        >
          <IconSwap
            active={visible}
            from={<Eye aria-hidden className="size-4" />}
            to={<EyeOff aria-hidden className="size-4" />}
          />
        </button>
      </div>

      {hint && !error ? (
        <p id={hintId} className="text-[0.8rem] leading-snug text-muted-foreground">
          {hint}
        </p>
      ) : null}
      <FieldError id={errorId} message={error} />
    </div>
  );
}
