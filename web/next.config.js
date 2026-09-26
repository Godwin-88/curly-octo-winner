/** @type {import('next').NextConfig} */

// The Go API origin that /api/v1/* requests are proxied to (server-side,
// same-origin). NEXT_PUBLIC_API_URL may point at the API root or include the
// /api/v1 suffix — both are normalised here. The browser never talks to this
// origin directly, which makes session cookies first-party and removes CORS
// from the request path entirely.
const API_ORIGIN = (process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080')
  .replace(/\/+$/, '')
  .replace(/\/api\/v1$/, '');

// Fail loudly at deploy time if a production build forgot the proxy target —
// otherwise every API call would hit localhost inside the serverless function.
if (process.env.NODE_ENV === 'production' && !process.env.NEXT_PUBLIC_API_URL) {
  console.warn(
    '\n[shule360] WARNING: NEXT_PUBLIC_API_URL is not set for this production build.\n' +
      '[shule360] /api/v1/* will proxy to http://localhost:8080, which will fail in production.\n' +
      '[shule360] Set NEXT_PUBLIC_API_URL to the Go API origin (e.g. https://shule360-api.onrender.com).\n'
  );
}

const nextConfig = {
  reactStrictMode: true,
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: '*.backblazeb2.com',
      },
    ],
  },
  async rewrites() {
    return [
      {
        source: '/api/v1/:path*',
        destination: `${API_ORIGIN}/api/v1/:path*`,
      },
    ];
  },
};

// Sentry wraps the config to inject instrumentation; it is inert unless
// SENTRY_DSN / NEXT_PUBLIC_SENTRY_DSN are set (and never uploads sourcemaps
// without SENTRY_ORG/SENTRY_PROJECT/auth token).
// Imported from '@sentry/nextjs/config': the root export is deprecated and
// stops working in Sentry v11.
import { withSentryConfig } from '@sentry/nextjs/config';

export default withSentryConfig(nextConfig, {
  silent: true,
  telemetry: false,
});