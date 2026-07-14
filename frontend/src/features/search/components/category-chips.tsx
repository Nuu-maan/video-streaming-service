import Link from "next/link";

import { routes } from "@/config/routes";
import { formatCompact } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { CategoryCount } from "@/types/common";

interface CategoryChipsProps {
  categories: CategoryCount[];
  /** The category currently filtered on, if any. Its chip becomes a toggle-off. */
  active?: string;
  /** Current search query, preserved when hopping between categories. */
  q?: string;
  className?: string;
}

function chipHref(category: string | null, q: string | undefined): string {
  const params = new URLSearchParams();
  if (q) params.set("q", q);
  if (category) params.set("category", category);
  const query = params.toString();
  return query ? `${routes.search}?${query}` : routes.search;
}

/**
 * One chip per category from `GET /categories`, name plus how many videos
 * carry it. A horizontally scrollable row of links — URL state, so a chosen
 * category is shareable and the back button undoes it. Server-compatible.
 */
export function CategoryChips({ categories, active, q, className }: CategoryChipsProps) {
  if (categories.length === 0) return null;

  return (
    <nav aria-label="Categories" className={cn("min-w-0", className)}>
      <ul className="no-scrollbar -mx-1 flex gap-2 overflow-x-auto px-1 py-1">
        {[null, ...categories.map((entry) => entry.category)].map((category) => {
          const entry = categories.find((candidate) => candidate.category === category);
          const isActive = category === null ? !active : active === category;
          return (
            <li key={category ?? "__all__"} className="shrink-0">
              <Link
                // An active chip links to itself-removed: clicking it toggles off.
                href={chipHref(isActive ? null : category, q)}
                aria-current={isActive && category !== null ? "true" : undefined}
                className={cn(
                  "flex h-8 items-center gap-1.5 rounded-full border px-3.5 text-sm outline-none",
                  "transition-colors duration-(--motion-fast) focus-visible:ring-3 focus-visible:ring-ring/50",
                  isActive
                    ? "border-transparent bg-primary font-medium text-primary-foreground"
                    : "border-border/70 bg-muted/40 text-muted-foreground hover:bg-muted hover:text-foreground",
                )}
              >
                {category ?? "All"}
                {entry ? (
                  <span
                    className={cn(
                      "text-xs tabular-nums",
                      isActive ? "text-primary-foreground/80" : "text-muted-foreground/70",
                    )}
                  >
                    {formatCompact(entry.video_count)}
                  </span>
                ) : null}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
