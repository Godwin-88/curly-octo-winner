'use client';

// Admin dashboard error boundary.

import { useEffect } from 'react';
import { RouteError } from '@/components/ui/RouteError';

export default function AdminError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error('[admin error boundary]', error);
  }, [error]);

  return <RouteError error={error} reset={reset} />;
}
