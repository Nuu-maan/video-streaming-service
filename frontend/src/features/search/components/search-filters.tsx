"use client";

import { X } from "lucide-react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { DURATION_OPTIONS } from "@/features/search/types";
import { cn } from "@/lib/utils";
import type { CategoryCount } from "@/types/common";

const SORT_OPTIONS = [
  { value: "relevance", label: "Relevance" },
  { value: "newest", label: "Newest" },
  { value: "views", label: "Most viewed" },
  { value: "likes", label: "Most liked" },
] as const;

const ALL_CATEGORIES = "__all__";

interface SearchFiltersProps {
  categories: CategoryCount[];
  className?: string;
}

/**
 * Category, sort and duration narrowing. All state lives in the URL — a
 * filtered search is a link someone can share, and the back button undoes a
 * filter the way a user expects. Defaults are *deleted* from the URL, not
 * written into it, so the canonical form of a plain search stays `?q=`.
 */
export function SearchFilters({ categories, className }: SearchFiltersProps) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const category = searchParams.get("category") ?? ALL_CATEGORIES;
  const sort = searchParams.get("sort") ?? "relevance";
  const duration =
    DURATION_OPTIONS.find(
      (option) =>
        option.min === searchParams.get("min_duration") &&
        option.max === searchParams.get("max_duration"),
    )?.value ?? "any";

  const hasActiveFilters =
    category !== ALL_CATEGORIES ||
    sort !== "relevance" ||
    searchParams.has("min_duration") ||
    searchParams.has("max_duration");

  /** Set or delete (`null`) params; any change resets pagination. */
  function apply(updates: Record<string, string | null>) {
    const params = new URLSearchParams(searchParams);
    for (const [key, value] of Object.entries(updates)) {
      if (value === null) params.delete(key);
      else params.set(key, value);
    }
    params.delete("page");
    const query = params.toString();
    router.push(query ? `${pathname}?${query}` : pathname);
  }

  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      {categories.length > 0 ? (
        <Select
          value={category}
          onValueChange={(value) => apply({ category: value === ALL_CATEGORIES ? null : value })}
        >
          <SelectTrigger size="sm" aria-label="Category" className="w-auto min-w-36 rounded-full">
            <SelectValue placeholder="Category" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_CATEGORIES}>All categories</SelectItem>
            {categories.map((entry) => (
              <SelectItem key={entry.category} value={entry.category}>
                {entry.category}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : null}

      <Select value={sort} onValueChange={(value) => apply({ sort: value === "relevance" ? null : value })}>
        <SelectTrigger size="sm" aria-label="Sort by" className="w-auto min-w-32 rounded-full">
          <SelectValue placeholder="Sort by" />
        </SelectTrigger>
        <SelectContent>
          {SORT_OPTIONS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select
        value={duration}
        onValueChange={(value) => {
          const option = DURATION_OPTIONS.find((entry) => entry.value === value);
          apply({ min_duration: option?.min ?? null, max_duration: option?.max ?? null });
        }}
      >
        <SelectTrigger size="sm" aria-label="Duration" className="w-auto min-w-36 rounded-full">
          <SelectValue placeholder="Duration" />
        </SelectTrigger>
        <SelectContent>
          {DURATION_OPTIONS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {hasActiveFilters ? (
        <Button
          variant="ghost"
          size="sm"
          className="rounded-full text-muted-foreground"
          onClick={() =>
            apply({ category: null, sort: null, min_duration: null, max_duration: null })
          }
        >
          <X aria-hidden />
          Clear filters
        </Button>
      ) : null}
    </div>
  );
}
