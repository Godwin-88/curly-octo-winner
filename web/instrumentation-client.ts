// Client-side Sentry config (loaded automatically by @sentry/nextjs via
// instrumentation-client.ts convention) — enabled only with
// NEXT_PUBLIC_SENTRY_DSN.
import * as Sentry from '@sentry/nextjs';

if (process.env.NEXT_PUBLIC_SENTRY_DSN) {
  Sentry.init({
    dsn: process.env.NEXT_PUBLIC_SENTRY_DSN,
    environment: process.env.NODE_ENV,
    sendDefaultPii: false,
    tracesSampleRate: 0,
    // Route transitions create spans only when traces are on; skip for now.
  });
}

// @sentry/nextjs requires this export to instrument App Router navigations.
// Without it the SDK logs an ACTION REQUIRED warning on every build/start.
export const onRouterTransitionStart = Sentry.captureRouterTransitionStart;
