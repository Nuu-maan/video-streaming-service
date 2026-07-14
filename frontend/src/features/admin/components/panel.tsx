import { cn } from "@/lib/utils";

interface PanelProps {
  title: string;
  description?: string;
  /** Right-aligned: a "last updated" timestamp, a link, a small control. */
  aside?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}

/**
 * A titled section of the admin surface.
 *
 * `shadow-border` rather than a `border`: the design system's layered
 * ring-plus-lift reads as a raised surface in light mode and a single hairline
 * ring in dark, where a drop shadow would be invisible anyway. Panels sit on
 * the page background, so they need to lift off it.
 *
 * The heading is an `h2` because the page already owns the `h1` — an admin
 * screen with four `h1`s is four documents as far as a screen reader's outline
 * is concerned.
 */
export function Panel({ title, description, aside, children, className }: PanelProps) {
  return (
    <section className={cn("rounded-xl bg-card shadow-border", className)}>
      <header className="flex flex-wrap items-start justify-between gap-x-6 gap-y-2 px-5 pt-5 pb-4">
        <div className="min-w-0">
          <h2 className="text-heading text-balance">{title}</h2>
          {description ? (
            <p className="mt-1 text-sm text-pretty text-muted-foreground">{description}</p>
          ) : null}
        </div>
        {aside ? <div className="shrink-0">{aside}</div> : null}
      </header>
      <div className="px-5 pb-5">{children}</div>
    </section>
  );
}
