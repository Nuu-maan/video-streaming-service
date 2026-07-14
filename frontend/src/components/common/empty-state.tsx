import type { LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";

interface EmptyStateProps {
  icon?: LucideIcon;
  title: string;
  description?: string;
  /** Usually a Button or a Link styled as one. */
  action?: React.ReactNode;
  className?: string;
}

/**
 * A designed empty state: icon in a soft well, a headline, a short
 * explanation, and — whenever there is a sensible next step — an action.
 * Server-compatible; no hooks.
 */
export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex min-h-64 flex-col items-center justify-center rounded-xl border border-dashed border-border/70 px-6 py-12 text-center",
        className,
      )}
    >
      {Icon ? (
        <div className="mb-4 flex size-14 items-center justify-center rounded-2xl bg-muted text-muted-foreground ring-1 ring-border/60 ring-inset">
          <Icon aria-hidden className="size-6" />
        </div>
      ) : null}
      <h2 className="text-heading text-balance">{title}</h2>
      {description ? (
        <p className="mt-1.5 max-w-sm text-sm text-pretty text-muted-foreground">{description}</p>
      ) : null}
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  );
}
