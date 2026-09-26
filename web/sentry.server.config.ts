// Server (Node.js runtime) Sentry config — enabled only with SENTRY_DSN.
import * as Sentry from '@sentry/nextjs';

if (process.env.SENTRY_DSN) {
  Sentry.init({
    dsn: process.env.SENTRY_DSN,
    environment: process.env.APP_ENV ?? process.env.NODE_ENV,
    sendDefaultPii: false,
    // Server traces off by default; enable deliberately when perf tuning.
    tracesSampleRate: 0,
  });
}
