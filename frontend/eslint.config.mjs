import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

/**
 * The architecture is enforced here, not in a README.
 *
 * The rules below encode one idea: dependencies point one way,
 * `shared -> features -> app`. Without them `components/` and `lib/` decay into
 * dumping grounds within a month — that is the most commonly cited failure mode
 * of exactly this layout, and good intentions do not prevent it.
 *
 * There are also deliberately no barrel files (`index.ts` re-exports) in `src`.
 * Next's own docs flag them as a build-speed problem: the compiler must parse the
 * whole barrel to find module-scope side effects, so importing one component
 * drags in every sibling. `optimizePackageImports` fixes that for `node_modules`
 * only, never for your own code. Direct imports cost one path segment and save
 * every rebuild thereafter.
 */
const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,

  globalIgnores([
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
    // Machine-generated from the backend's OpenAPI spec. Regenerate, never edit.
    "src/types/api.ts",
  ]),

  {
    files: ["src/**/*.{ts,tsx}"],
    rules: {
      // `..` is the first step towards a file reaching into a sibling's
      // internals. Anything outside the current folder goes through `@/`, where
      // the boundary rules below can actually see it.
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["../*"],
              message:
                "Use the @/ alias for anything outside the current folder. Relative imports are for siblings only.",
            },
            {
              group: ["@/app/**/_components/*"],
              message:
                "A route's _components are private to that route. If a second route needs it, promote it to src/features/<domain>/components/ or src/components/common/.",
            },
          ],
        },
      ],
    },
  },

  // Shared code must not know the product exists. A file in components/, hooks/
  // or lib/ that imports a feature has inverted the dependency: it is a feature
  // component living in the wrong folder.
  {
    files: ["src/components/**/*.{ts,tsx}", "src/hooks/**/*.{ts,tsx}", "src/lib/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/features/*", "@/features/**", "@/app/*", "@/app/**"],
              message:
                "Shared code cannot depend on a feature or a route. If it needs a domain concept, it belongs in src/features/<domain>/.",
            },
            { group: ["../*"], message: "Use the @/ alias for anything outside the current folder." },
          ],
        },
      ],
    },
  },

  // Features own the product logic. They may use shared code freely, but a route
  // is downstream of them: routes compose features, never the reverse.
  {
    files: ["src/features/**/*.{ts,tsx}"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["@/app/*", "@/app/**"],
              message: "A feature cannot import from a route. Routes compose features, not the reverse.",
            },
            { group: ["../*"], message: "Use the @/ alias for anything outside the current folder." },
          ],
        },
      ],
    },
  },
]);

export default eslintConfig;
