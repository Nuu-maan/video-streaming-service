/**
 * A validation message under a field.
 *
 * `role="alert"` so it is announced when it appears — a red line of text a
 * screen reader never mentions is not an error message, it is decoration. It
 * renders nothing at all when there is no error, rather than an empty element
 * that reserves space and makes the form twitch as messages come and go.
 */
export function FieldError({ message }: { message?: string }) {
  if (!message) return null;

  return (
    <p role="alert" className="text-sm text-pretty text-destructive">
      {message}
    </p>
  );
}
