"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";

import { Button } from "@/components/ui/button";
import type { PaginationMeta } from "@/types/common";
import { cn } from "@/lib/utils";

interface PaginationProps {
  pagination: PaginationMeta;
  className?: string;
}

/**
 * Which page numbers to show: always 1 and the last, the current page ±1,
 * `null` marking a gap. Small totals render in full.
 */
function pageWindow(current: number, total: number): (number | null)[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
  const pages = new Set<number>([1, total, current - 1, current, current + 1]);
  const sorted = [...pages].filter((p) => p >= 1 && p <= total).sort((a, b) => a - b);
  const result: (number | null)[] = [];
  let previous = 0;
  for (const page of sorted) {
    if (page - previous === 2) result.push(previous + 1);
    else if (page - previous > 2) result.push(null);
    result.push(page);
    previous = page;
  }
  return result;
}

/**
 * Driven by the API's PaginationMeta; navigates by updating `?page=` and
 * preserving every other search param. Uses `useSearchParams`, so pages that
 * render it statically need a Suspense boundary around it — the data-fetching
 * pages it belongs on are dynamic anyway.
 */
export function Pagination({ pagination, className }: PaginationProps) {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { page: current, total_pages: total, has_next, has_previous } = pagination;

  if (total <= 1) return null;

  const hrefFor = (page: number) => {
    const params = new URLSearchParams(searchParams);
    if (page === 1) params.delete("page");
    else params.set("page", String(page));
    const query = params.toString();
    return query ? `${pathname}?${query}` : pathname;
  };

  return (
    /*
     * gap-2 / gap-3: every icon button's hit area now reaches 40px (fine) / 44px
     * (coarse) via a pseudo-element, and two hit areas must never overlap — so
     * adjacent icon buttons need at least twice the overhang between them.
     */
    <nav
      aria-label="Pagination"
      className={cn("flex items-center justify-center gap-2 pointer-coarse:gap-3", className)}
    >
      <Button
        asChild={has_previous}
        variant="ghost"
        size="icon"
        disabled={!has_previous}
        aria-label="Previous page"
      >
        {has_previous ? (
          <Link href={hrefFor(current - 1)} rel="prev">
            <ChevronLeft aria-hidden />
          </Link>
        ) : (
          <ChevronLeft aria-hidden />
        )}
      </Button>

      {/*
       * Seven touch-sized page numbers plus their gaps do not fit a 360px phone —
       * something has to give, and it is not going to be the hit area. Below `sm`
       * the numeric window gives way to a plain "Page 3 of 12", which is a real
       * pagination pattern and leaves prev/next big enough to actually hit.
       */}
      <p className="px-1 text-sm text-muted-foreground tabular-nums sm:hidden">
        Page {current} of {total}
      </p>

      <div className="hidden items-center gap-2 pointer-coarse:gap-3 sm:flex">
        {pageWindow(current, total).map((page, index) =>
          page === null ? (
            <span
              key={`gap-${index}`}
              aria-hidden
              className="flex size-8 items-end justify-center pb-1.5 text-xs text-muted-foreground select-none"
            >
              …
            </span>
          ) : (
            <Button
              key={page}
              asChild
              variant={page === current ? "secondary" : "ghost"}
              size="icon"
              className="tabular-nums"
            >
              <Link
                href={hrefFor(page)}
                aria-label={`Page ${page}`}
                aria-current={page === current ? "page" : undefined}
              >
                {page}
              </Link>
            </Button>
          ),
        )}
      </div>

      <Button
        asChild={has_next}
        variant="ghost"
        size="icon"
        disabled={!has_next}
        aria-label="Next page"
      >
        {has_next ? (
          <Link href={hrefFor(current + 1)} rel="next">
            <ChevronRight aria-hidden />
          </Link>
        ) : (
          <ChevronRight aria-hidden />
        )}
      </Button>
    </nav>
  );
}
