/** Sort orders the API's `/search` endpoint accepts. */
export const SEARCH_SORTS = ["relevance", "newest", "views", "likes"] as const;

export type SearchSort = (typeof SEARCH_SORTS)[number];

export function isSearchSort(value: string): value is SearchSort {
  return (SEARCH_SORTS as readonly string[]).includes(value);
}

/** Time windows the API's `/videos/trending` endpoint accepts. */
export type TrendingWindow = "24h" | "7d" | "30d";

/**
 * Everything a search can be narrowed by. Mirrors the query parameters of
 * `GET /search` — camelCased here, translated to snake_case at the API edge.
 */
export interface SearchQuery {
  q: string;
  sort?: SearchSort;
  category?: string;
  language?: string;
  /** Comma-separated tag list, exactly as the API takes it. */
  tags?: string;
  /** Seconds. The API rejects negative values. */
  minDuration?: number;
  maxDuration?: number;
  page?: number;
  limit?: number;
}

/**
 * The duration buckets the filter UI offers. They exist so the URL carries
 * `min_duration`/`max_duration` — real API parameters, shareable — while the
 * user just picks a named range.
 */
export const DURATION_OPTIONS = [
  { value: "any", label: "Any duration", min: null, max: null },
  { value: "short", label: "Under 4 minutes", min: null, max: "240" },
  { value: "medium", label: "4–20 minutes", min: "240", max: "1200" },
  { value: "long", label: "Over 20 minutes", min: "1200", max: null },
] as const;
