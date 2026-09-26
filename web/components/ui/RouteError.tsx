// Shared content for route-level error boundaries (app/**/error.tsx).
// Next.js passes an error + reset() to every error.tsx; this component keeps
// the actual UI in one place so each segment stays a three-liner.

'use client';

import Link from 'next/link';

export function RouteError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="min-h-[60vh] flex items-center justify-center p-6">
      <div className="card max-w-lg w-full p-8 text-center space-y-3">
        <div className="text-4xl" aria-hidden="true">
          🛠️
        </div>
        <h1 className="text-xl font-bold text-gray-900">This page hit a problem</h1>
        <p className="text-sm text-gray-500">
          The error was logged. You can retry — if it keeps happening, sign out and back
          in, or contact support{error.digest ? ` (ref: ${error.digest})` : ''}.
        </p>
        <div className="flex items-center justify-center gap-3 pt-2">
          <button type="button" onClick={reset} className="btn-primary">
            Try again
          </button>
          <Link href="/dashboard" className="btn-secondary">
            Go to dashboard
          </Link>
        </div>
      </div>
    </div>
  );
}
