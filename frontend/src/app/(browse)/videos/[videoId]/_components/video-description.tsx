"use client";

import Link from "next/link";
import { useState } from "react";

import { routes } from "@/config/routes";
import { formatCount, formatDate, formatRelativeTime } from "@/lib/format";
import { cn } from "@/lib/utils";

interface VideoDescriptionProps {
  description: string;
  viewCount: number;
  createdAt: string;
  category?: string;
  tags?: string[];
}

/**
 * The description panel: counts, the date, the text, and the tags.
 *
 * Collapsed it shows three lines. Expanding toggles the clamp rather than
 * animating a height — height is a layout property, and animating it on a block
 * of arbitrary text janks and shoves the comments below it around for 300ms.
 * The change is instant, which is also what a "show more" button promises.
 *
 * "Show more" is the ONLY control, and that is a deliberate reduction. The
 * <section> used to carry its own onClick, so any click inside the collapsed
 * panel expanded it — including the click that STARTS A TEXT SELECTION in the
 * description, and the click on the view-count line. Nobody could select a
 * sentence out of a description without the panel jumping under them. It was
 * also a non-interactive landmark with a click handler and no role: invisible to
 * the keyboard and to assistive tech, so the "large, obvious target" it claimed
 * to provide existed for mouse users only.
 *
 * The button was already doing the real work, and it already spans the
 * affordance. One control, reachable by everyone.
 */
export function VideoDescription({ description, viewCount, createdAt, category, tags }: VideoDescriptionProps) {
  const [expanded, setExpanded] = useState(false);
  const text = description.trim();
  const chips = [...(category ? [category] : []), ...(tags ?? [])].slice(0, 12);

  return (
    <section
      aria-label="Video details"
      className="rounded-xl bg-muted/50 px-4 py-3 text-sm"
    >
      <p className="font-medium">
        <span className="tabular-nums">{formatCount(viewCount, "view")}</span>
        <span aria-hidden className="px-1.5 text-muted-foreground">
          ·
        </span>
        <time dateTime={createdAt} title={formatDate(createdAt)} suppressHydrationWarning>
          {formatRelativeTime(createdAt)}
        </time>
      </p>

      {text ? (
        <p className={cn("mt-2 whitespace-pre-wrap text-pretty", !expanded && "line-clamp-3")}>{text}</p>
      ) : (
        <p className="mt-2 text-muted-foreground italic">No description.</p>
      )}

      {chips.length > 0 && expanded ? (
        <ul className="mt-3 flex flex-wrap gap-1.5">
          {chips.map((chip) => (
            <li key={chip}>
              {/* No stopPropagation needed any more — nothing above is listening. */}
              <Link
                href={routes.category(chip)}
                className="inline-flex h-7 items-center rounded-full bg-background px-2.5 text-xs text-muted-foreground outline-none ring-1 ring-border transition-colors duration-(--motion-fast) hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50"
              >
                #{chip}
              </Link>
            </li>
          ))}
        </ul>
      ) : null}

      {text.length > 140 || chips.length > 0 ? (
        <button
          type="button"
          onClick={() => setExpanded((open) => !open)}
          aria-expanded={expanded}
          className="mt-2 rounded-sm font-medium text-muted-foreground outline-none transition-colors duration-(--motion-fast) hover:text-foreground focus-visible:ring-3 focus-visible:ring-ring/50"
        >
          {expanded ? "Show less" : "Show more"}
        </button>
      ) : null}
    </section>
  );
}
