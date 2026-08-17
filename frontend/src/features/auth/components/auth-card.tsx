/**
 * The card every auth page sits in. One component so the four screens cannot
 * drift apart in padding, heading size, or where the footer link lands — the
 * flows are cross-linked (sign in → forgot → reset → sign in), and a person
 * walking that path should feel like they are moving inside one surface, not
 * between four pages that were each designed once.
 *
 * `shadow-border` rather than `border`: a 1px ring plus a lift in light mode, a
 * single white ring in dark, where layered depth is invisible anyway.
 */
export function AuthCard({
  title,
  description,
  footer,
  children,
}: {
  title: string;
  description?: React.ReactNode;
  footer?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div className="w-full max-w-sm">
      <div className="rounded-2xl bg-card p-6 shadow-border sm:p-7">
        <div className="mb-6 grid gap-1.5">
          <h1 className="text-heading text-balance text-card-foreground">{title}</h1>
          {description ? <p className="text-pretty text-sm text-muted-foreground">{description}</p> : null}
        </div>

        {children}
      </div>

      {footer ? <p className="mt-5 text-center text-sm text-muted-foreground">{footer}</p> : null}
    </div>
  );
}
