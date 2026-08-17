"use client";

import { useRouter, usePathname } from "next/navigation";
import { useCallback } from "react";
import { toast } from "sonner";

import { routes } from "@/config/routes";

/**
 * Every social affordance — liking, commenting, subscribing, saving — is
 * visible to anonymous visitors and only fails at the moment of use. A click
 * that silently does nothing reads as a broken button, so instead we say what
 * is missing and offer the one-click fix, returning to the exact page they were
 * on. Domain-free: it knows about sessions, not about videos.
 */
export function useSignInPrompt(): (message?: string) => void {
  const router = useRouter();
  const pathname = usePathname();

  return useCallback(
    (message = "Sign in to continue.") => {
      const next = `${routes.login}?next=${encodeURIComponent(pathname)}`;
      toast(message, {
        action: {
          label: "Sign in",
          onClick: () => router.push(next),
        },
      });
    },
    [router, pathname],
  );
}
