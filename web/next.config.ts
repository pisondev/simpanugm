import type { NextConfig } from "next"
import createNextIntlPlugin from "next-intl/plugin"

const apiInternal = process.env.API_INTERNAL_URL ?? "http://localhost:8090"

const nextConfig: NextConfig = {
  output: "standalone",
  poweredByHeader: false,
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
  // the same origin setup of production, so cookies and the CSRF check behave identically when Caddy is absent
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiInternal}/api/:path*` }]
  },
}

export default createNextIntlPlugin("./src/i18n/request.ts")(nextConfig)
