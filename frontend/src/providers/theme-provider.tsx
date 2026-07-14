"use client";

import { ThemeProvider as NextThemesProvider } from "next-themes";

/**
 * Dark is the default: video is watched in the dark, and bright chrome around
 * a dark player is unpleasant. `disableTransitionOnChange` matters — without
 * it every theme switch cross-fades the whole page, which looks broken.
 */
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  return (
    <NextThemesProvider
      attribute="class"
      defaultTheme="dark"
      enableSystem
      disableTransitionOnChange
    >
      {children}
    </NextThemesProvider>
  );
}
