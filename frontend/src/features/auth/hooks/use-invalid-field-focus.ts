"use client";

import { useEffect, useRef } from "react";

import type { AuthFormState } from "@/features/auth/schemas";

/**
 * After a failed submit, put the caret where the problem is.
 *
 * Returns a ref for the <form>. On every settled action result that carries
 * field errors, it focuses the first control the server marked `aria-invalid` —
 * which, because the fields wire `aria-describedby` to their error message,
 * makes a screen reader read the label AND the reason in one breath.
 *
 * Without this, the flow was: press "Sign in", the page re-renders with
 * "Username is required" under a field, and focus stays parked on the submit
 * button. Nothing is announced, nothing has moved, and a person who cannot see
 * the red text has no way to know the form even rejected them.
 *
 * Querying `[aria-invalid="true"]` rather than threading ids around is
 * deliberate: the fields already generate their own ids with `useId`, and the
 * DOM order of the invalid controls IS the visual order, so "first match" is
 * genuinely the first error on the form.
 *
 * The effect keys on the whole `state` object, not on its contents:
 * `useActionState` hands back a fresh object on every submit, so submitting
 * twice with the same error still re-runs — which is the case that matters.
 */
export function useInvalidFieldFocus(state: AuthFormState) {
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    if (!state.fieldErrors) return;
    const firstInvalid = formRef.current?.querySelector<HTMLElement>('[aria-invalid="true"]');
    firstInvalid?.focus();
  }, [state]);

  return formRef;
}
