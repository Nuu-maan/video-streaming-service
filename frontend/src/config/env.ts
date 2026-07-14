import { z } from "zod";

/**
 * Environment is validated once, at module load, so a missing or malformed
 * variable fails the build or the boot — not the first request that happens to
 * need it. An app that starts successfully and then 500s on the login page
 * because NEXT_PUBLIC_API_URL was a typo is strictly worse than one that
 * refuses to start.
 */
const schema = z.object({
  /**
   * Origin of the Go API, with no trailing slash and no /api/v1 suffix — the
   * client appends the version prefix itself, so the version lives in one place.
   */
  NEXT_PUBLIC_API_URL: z
    .string()
    .url()
    .transform((value) => value.replace(/\/+$/, "")),
  NEXT_PUBLIC_SITE_URL: z
    .string()
    .url()
    .transform((value) => value.replace(/\/+$/, "")),
});

/**
 * Next inlines `process.env.NEXT_PUBLIC_*` at build time by matching the literal
 * text `process.env.NEXT_PUBLIC_FOO`. Destructuring `process.env` or indexing it
 * dynamically defeats that substitution and yields undefined in the browser, so
 * each variable is named in full here.
 */
const parsed = schema.safeParse({
  NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL,
  NEXT_PUBLIC_SITE_URL: process.env.NEXT_PUBLIC_SITE_URL,
});

if (!parsed.success) {
  const issues = parsed.error.issues
    .map((issue) => `  ${issue.path.join(".")}: ${issue.message}`)
    .join("\n");
  throw new Error(`Invalid environment variables:\n${issues}`);
}

export const env = parsed.data;

/** Every API path is built from this, so the version prefix is declared once. */
export const API_BASE = `${env.NEXT_PUBLIC_API_URL}/api/v1` as const;
