/**
 * The inline message under a field. Rendering nothing when there is no error
 * (rather than an empty, height-holding element) is deliberate: an auth form is
 * short, and reserving space for errors that may never appear makes the whole
 * card look loose. The shift when one does appear is one line, at the bottom of
 * the field, below the caret — it never moves what the person is looking at.
 *
 * `role="alert"` because the element MOUNTS at the moment the error appears,
 * which is what makes an alert fire. Without it, a Zod validation failure was
 * completely silent for a screen-reader user: the action returns fieldErrors and
 * no form-level error, so <FormAlert> — the only other thing on the page with a
 * role — never rendered, and the person heard nothing at all after pressing
 * "Sign in". The form additionally moves focus to the first invalid field (see
 * `useInvalidFieldFocus`), so the message is both announced and reachable.
 */
export function FieldError({ id, message }: { id: string; message?: string }) {
  if (!message) return null;

  return (
    <p id={id} role="alert" className="text-[0.8rem] leading-snug font-medium text-destructive">
      {message}
    </p>
  );
}
