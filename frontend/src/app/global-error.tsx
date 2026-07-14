"use client";

import { RotateCw, TriangleAlert } from "lucide-react";

import "./globals.css";

/**
 * The last resort.
 *
 * `global-error` only fires when the ROOT LAYOUT itself throws — a nested
 * error.tsx catches everything below it. That means this component REPLACES the
 * root layout, so it has to render its own <html> and <body>, and it cannot
 * assume a single thing the layout normally sets up: no AppProviders, no
 * ThemeProvider, no Toaster. Anything imported here has to stand alone, which is
 * why the stylesheet is imported directly and the theme is resolved by hand
 * below rather than by next-themes.
 *
 * It is deliberately plain. A page that renders when the application's own root
 * has failed is not the place for anything clever: no data fetching, no context,
 * no animation. Say what happened, and offer the one action that can help.
 */

/**
 * next-themes writes the preference to localStorage under "theme" and stamps a
 * class on <html>. That provider is gone here, so this replays the same
 * resolution before first paint: stored value, or the system preference when it
 * is "system", falling back to the app's own default of dark.
 *
 * <html> ships WITH the dark class and the script removes it for a light
 * preference, so the no-JS and JS-disabled paths land on the app's default
 * rather than a white flash. Wrapped in try/catch because localStorage throws in
 * some privacy modes, and a crash page that crashes is a bad crash page.
 */
const THEME_SCRIPT = `
try {
  var t = localStorage.getItem('theme') || 'dark';
  if (t === 'system') {
    t = matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  if (t !== 'dark') document.documentElement.classList.remove('dark');
} catch (e) {}
`;

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html lang="en" className="dark h-full antialiased" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: THEME_SCRIPT }} />
      </head>
      <body className="flex min-h-full flex-col bg-background text-foreground">
        <main className="mx-auto flex w-full max-w-md flex-1 flex-col items-center justify-center px-6 py-16 text-center">
          <div className="mb-5 flex size-14 items-center justify-center rounded-2xl bg-destructive/10 text-destructive ring-1 ring-destructive/20 ring-inset">
            <TriangleAlert aria-hidden className="size-6" />
          </div>

          <h1 className="text-title text-balance">Something went badly wrong</h1>
          <p className="mt-2 text-sm text-pretty text-muted-foreground">
            The app failed to load. This is on us, not on you. Try again — and if it keeps
            happening, come back in a few minutes.
          </p>

          <button
            type="button"
            onClick={reset}
            className="mt-6 inline-flex h-9 items-center gap-2 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground outline-none transition-colors duration-(--motion-fast) hover:bg-primary/90 focus-visible:ring-3 focus-visible:ring-ring/50"
          >
            <RotateCw aria-hidden className="size-4" />
            Try again
          </button>

          {/*
           * The digest, and only the digest. It is the id that ties this crash to
           * a line in the server log, which is the one thing worth quoting in a
           * bug report. The error's own message is deliberately not printed: in
           * production React replaces it with a generic string anyway, and in
           * development it can carry internals that have no business on screen.
           */}
          {error.digest ? (
            <p className="mt-8 font-mono text-xs text-muted-foreground/70">
              Reference: <span className="tabular-nums">{error.digest}</span>
            </p>
          ) : null}
        </main>
      </body>
    </html>
  );
}
