"use client";

import { useEffect, useState } from "react";

/**
 * Returns `value` after it has held still for `delayMs`. The staple for
 * search-as-you-type: debounce the query, fire the request on the debounced
 * value.
 */
export function useDebounce<T>(value: T, delayMs = 300): T {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delayMs);
    return () => window.clearTimeout(timer);
  }, [value, delayMs]);

  return debounced;
}
