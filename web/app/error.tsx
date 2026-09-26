'use client';

// Root app error boundary: catches render errors in any route segment that
// doesn't define its own error.tsx (and errors outside the route groups).

import { useEffect } from 'react';
import { RouteError } from '@/components/ui/RouteError';

export default function AppError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // Surfaced in the browser console for now; wire Sentry (Phase 3) here.
    console.error('[app error boundary]', error);
  }, [error]);

  return <RouteError error={error} reset={reset} />;
}
