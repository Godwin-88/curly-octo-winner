// Sentry server/edge instrumentation (Next.js convention).
// No-ops unless SENTRY_DSN / NEXT_PUBLIC_SENTRY_DSN are configured.
import * as Sentry from '@sentry/nextjs';

export async function register() {
  if (process.env.NEXT_RUNTIME === 'nodejs') {
    await import('./sentry.server.config');
  }
  if (process.env.NEXT_RUNTIME === 'edge') {
    await import('./sentry.edge.config');
  }
}

// Captures errors from server actions, route handlers, and nested React
// render passes.
export const onRequestError = Sentry.captureRequestError;
