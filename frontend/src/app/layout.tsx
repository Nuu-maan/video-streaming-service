import type { Metadata, Viewport } from "next";
import { Geist, Geist_Mono } from "next/font/google";

import { site } from "@/config/site";
import { AppProviders } from "@/providers/app-providers";

import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  metadataBase: new URL(site.url),
  title: {
    default: site.name,
    template: `%s · ${site.name}`,
  },
  description: site.description,
  applicationName: site.name,
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  /* Hex equivalents of --background in each theme (browser chrome can't read
     CSS custom properties). */
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#fbfcfd" },
    { media: "(prefers-color-scheme: dark)", color: "#0a0c10" },
  ],
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    /* suppressHydrationWarning: next-themes stamps the theme class on <html>
       before hydration, which React would otherwise flag as a mismatch. */
    <html
      lang="en"
      suppressHydrationWarning
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="flex min-h-full flex-col">
        {/*
         * Skip link. Every shell in this app puts a sticky header (hamburger,
         * logo, search combobox, upload, bell, theme, avatar) and a sidebar rail
         * (~10 links plus a collapse toggle) in front of the content — so a
         * keyboard or switch user was tabbing through fifteen-odd repeated stops
         * on EVERY navigation before reaching anything they came for. WCAG 2.4.1
         * Bypass Blocks. Every <main> in every layout carries #main-content.
         *
         * It must be the first focusable thing in the body, and it must be
         * visually hidden until focused rather than `display: none` — a link the
         * browser cannot focus is not a skip link.
         */}
        <a
          href="#main-content"
          className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-100 focus:rounded-lg focus:bg-background focus:px-4 focus:py-2.5 focus:text-sm focus:font-medium focus:shadow-border-overlay focus:outline-none focus:ring-3 focus:ring-ring/50"
        >
          Skip to content
        </a>
        <AppProviders>{children}</AppProviders>
      </body>
    </html>
  );
}
