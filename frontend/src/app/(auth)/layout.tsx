import { Clapperboard } from "lucide-react";
import Link from "next/link";

import { routes } from "@/config/routes";
import { site } from "@/config/site";

/**
 * The auth shell: no header, no sidebar, nothing to click but the thing you came
 * to do. This is the first screen a new person sees, and the only screen a
 * returning one sees before they are back where they left off, so it earns its
 * own layout rather than borrowing the app's.
 *
 * The one decoration is a single soft brand-red bloom above the card. It is
 * aria-hidden, pointer-events-none, and painted with a radial gradient (no blur
 * filter — a blur on a full-bleed element is a real compositing cost on a page
 * whose whole job is to render instantly and take a password).
 */
export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative flex flex-1 flex-col items-center justify-center overflow-hidden px-4 py-12">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 -top-40 h-96 [background:radial-gradient(50%_60%_at_50%_50%,var(--color-brand-500)_0%,transparent_70%)] opacity-[0.10] dark:opacity-[0.16]"
      />

      <Link
        href={routes.home}
        className="relative mb-8 flex items-center gap-2 rounded-md text-base font-semibold tracking-tight outline-none transition-opacity duration-(--motion-fast) hover:opacity-80 focus-visible:ring-3 focus-visible:ring-ring/50"
      >
        <Clapperboard aria-hidden className="size-5 text-brand-500" />
        {site.name}
      </Link>

      {/* Rare, first-run screen: a 300ms settle is the one place in this app a
          purely welcoming animation is worth its cost. Reduced motion kills it
          globally. */}
      <main id="main-content" className="relative flex w-full animate-in justify-center fade-in-0 slide-in-from-bottom-2 duration-(--motion-slow) ease-out-quart">
        {children}
      </main>
    </div>
  );
}
