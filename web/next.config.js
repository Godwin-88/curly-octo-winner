/** @type {import('next').NextConfig} */

// The Go API origin that /api/v1/* requests are proxied to (server-side,
// same-origin). NEXT_PUBLIC_API_URL may point at the API root or include the
// /api/v1 suffix — both are normalised here. The browser never talks to this
// origin directly, which makes session cookies first-party and removes CORS
// from the request path entirely.
const API_ORIGIN = (process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080')
  .replace(/\/+$/, '')
  .replace(/\/api\/v1$/, '');

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

export default nextConfig;