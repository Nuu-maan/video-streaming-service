import type { NextConfig } from "next";

/**
 * Turbopack is the default bundler in Next 16 — there is deliberately no
 * `webpack` key here, because adding one fails the build.
 */
const nextConfig: NextConfig = {
  images: {
    /**
     * Thumbnails are served by the Go API, not by this origin: `thumbnail_url`
     * arrives as a path (`/api/v1/videos/{id}/thumbnail`) which `mediaUrl()`
     * resolves against the API's host. `next/image` refuses to optimise a remote
     * host it has not been told about, so the API origin is declared here.
     *
     * `images.domains` is deprecated in 16 — `remotePatterns` is the supported
     * form, and it is stricter: host *and* scheme *and* path prefix.
     *
     * NOTE: the app currently renders thumbnails with plain <img>, on purpose.
     * A private or unlisted video's thumbnail is only reachable through this
     * origin's `/api/media` proxy (the API 404s it without a bearer token, and
     * the browser has none), and routing that through the image optimiser would
     * put a per-viewer-authorised image into a shared, unauthenticated cache.
     * This entry exists so `<Image>` works for public API-origin media; it is
     * not an invitation to convert the private paths.
     */
    remotePatterns: [
      {
        protocol: "http",
        hostname: "localhost",
        port: "8080",
        pathname: "/api/v1/**",
      },
    ],
  },
};

export default nextConfig;
