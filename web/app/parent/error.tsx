'use client';

// Parent portal error boundary.

import { useEffect } from 'react';
import { RouteError } from '@/components/ui/RouteError';

export default function ParentError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error('[parent error boundary]', error);
  }, [error]);

  return <RouteError error={error} reset={reset} />;
}
