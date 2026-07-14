import { cn } from "@/lib/utils";

interface PageHeaderProps {
  title: string;
  description?: string;
  /** Right-aligned actions: a primary Button, a filter Select, etc. */
  actions?: React.ReactNode;
  className?: string;
}

/** The h1 block every listing/settings page opens with. */
export function PageHeader({ title, description, actions, className }: PageHeaderProps) {
  return (
    <div className={cn("flex flex-wrap items-end justify-between gap-x-6 gap-y-3", className)}>
      <div className="min-w-0">
        <h1 className="text-title text-balance">{title}</h1>
        {description ? (
          <p className="mt-1 max-w-2xl text-sm text-pretty text-muted-foreground">{description}</p>
        ) : null}
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
    </div>
  );
}
