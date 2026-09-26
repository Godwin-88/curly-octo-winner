'use client';

// Global error boundary: last resort when the ROOT LAYOUT itself throws
// (app/error.tsx only covers segments below it). Must render its own
// <html>/<body>.

import { useEffect } from 'react';
import * as Sentry from '@sentry/nextjs';

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error('[global error boundary]', error);
    // Sentry init is DSN-gated; this is a no-op until configured.
    Sentry.captureException(error);
  }, [error]);

  return (
    <html lang="en">
      <body className="bg-gray-50">
        <div className="min-h-screen flex items-center justify-center p-6">
          <div className="bg-white rounded-lg shadow max-w-lg w-full p-8 text-center space-y-3">
            <div className="text-4xl" aria-hidden="true">
              🛠️
            </div>
            <h1 className="text-xl font-bold text-gray-900">Shule360 hit a problem</h1>
            <p className="text-sm text-gray-500">
              The application failed to start properly. Try reloading the page.
            </p>
            <button type="button" onClick={reset} className="btn-primary">
              Reload
            </button>
          </div>
        </div>
      </body>
    </html>
  );
}
