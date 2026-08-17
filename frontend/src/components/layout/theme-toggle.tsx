"use client";

import { MoonIcon, SunIcon } from "lucide-react";
import { useTheme } from "next-themes";

import { Button } from "@/components/ui/button";

/**
 * Both icons stay in the DOM and cross-fade via the `dark:` variant — the
 * visible state comes from the class on <html>, not from React state, so
 * there is no hydration mismatch and no mounted-flag flash. Cross-fade values
 * follow the house icon-swap recipe: scale 0.25 ⇄ 1, blur 4px ⇄ 0.
 */
export function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme();

  return (
    <Button
      variant="ghost"
      size="icon"
      aria-label="Toggle theme"
      onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
    >
      <span className="relative flex size-4 items-center justify-center">
        <SunIcon
          aria-hidden
          className="absolute inset-0 scale-[0.25] opacity-0 blur-[4px] transition-[opacity,filter,scale] duration-(--motion-slow) ease-swap dark:scale-100 dark:opacity-100 dark:blur-none"
        />
        <MoonIcon
          aria-hidden
          className="scale-100 opacity-100 transition-[opacity,filter,scale] duration-(--motion-slow) ease-swap dark:scale-[0.25] dark:opacity-0 dark:blur-[4px]"
        />
      </span>
    </Button>
  );
}
