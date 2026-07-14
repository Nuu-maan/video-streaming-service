"use client";

import { useCallback, useSyncExternalStore } from "react";

/**
 * Tracks a CSS media query. SSR-safe: the server snapshot is `false`, so use
 * it for progressive enhancement, not for anything that changes markup the
 * server already rendered.
 *
 *   const isDesktop = useMediaQuery("(min-width: 768px)");
 */
export function useMediaQuery(query: string): boolean {
  const subscribe = useCallback(
    (onStoreChange: () => void) => {
      const mql = window.matchMedia(query);
      mql.addEventListener("change", onStoreChange);
      return () => mql.removeEventListener("change", onStoreChange);
    },
    [query],
  );

  return useSyncExternalStore(
    subscribe,
    () => window.matchMedia(query).matches,
    () => false,
  );
}
