"use client";

import { useCallback, useEffect, useRef, useState } from "react";

interface UseIntersectionObserverOptions {
  root?: Element | null;
  rootMargin?: string;
  threshold?: number | number[];
  /** Stop observing after the first intersection (reveal-once patterns). */
  once?: boolean;
}

interface UseIntersectionObserverResult<T extends Element> {
  /** Attach to the element to observe: `<div ref={ref} />`. */
  ref: (node: T | null) => void;
  isIntersecting: boolean;
}

/**
 * Observes an element's viewport intersection. The infinite-scroll recipe:
 * put the ref on a sentinel after the list, fetch the next page when
 * `isIntersecting` flips true (use `rootMargin: "400px"` to fetch early).
 */
export function useIntersectionObserver<T extends Element>({
  root = null,
  rootMargin = "0px",
  threshold = 0,
  once = false,
}: UseIntersectionObserverOptions = {}): UseIntersectionObserverResult<T> {
  const [isIntersecting, setIsIntersecting] = useState(false);
  const [node, setNode] = useState<T | null>(null);
  const frozen = useRef(false);

  const ref = useCallback((next: T | null) => {
    frozen.current = false;
    setNode(next);
  }, []);

  useEffect(() => {
    if (!node || frozen.current) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        setIsIntersecting(entry.isIntersecting);
        if (entry.isIntersecting && once) {
          frozen.current = true;
          observer.disconnect();
        }
      },
      { root, rootMargin, threshold },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [node, root, rootMargin, threshold, once]);

  return { ref, isIntersecting };
}
