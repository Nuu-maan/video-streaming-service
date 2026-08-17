import { TriangleAlert, type LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";

interface ErrorStateProps {
  icon?: LucideIcon;
  title?: string;
  description?: string;
  /** A retry button, a "go home" link — whatever recovery the caller can offer. */
  action?: React.ReactNode;
  className?: string;
}

/**
 * The failure twin of EmptyState. Callers own the copy: a 429 should read as
 * "slow down, try again in a moment", not as a generic failure, and a private
 * video is "video not found" — never "no permission".
 */
export function ErrorState({
  icon: Icon = TriangleAlert,
  title = "Something went wrong",
  description,
  action,
  className,
}: ErrorStateProps) {
  return (
    <div
      role="alert"
      className={cn(
        "flex min-h-64 flex-col items-center justify-center rounded-xl border border-dashed border-destructive/30 px-6 py-12 text-center",
        className,
      )}
    >
      <div className="mb-4 flex size-14 items-center justify-center rounded-2xl bg-destructive/10 text-destructive ring-1 ring-destructive/20 ring-inset">
        <Icon aria-hidden className="size-6" />
      </div>
      <h2 className="text-heading text-balance">{title}</h2>
      {description ? (
        <p className="mt-1.5 max-w-sm text-sm text-pretty text-muted-foreground">{description}</p>
      ) : null}
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  );
}
