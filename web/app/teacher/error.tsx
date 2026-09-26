'use client';

// Teacher portal error boundary.

import { useEffect } from 'react';
import { RouteError } from '@/components/ui/RouteError';

export default function TeacherError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error('[teacher error boundary]', error);
  }, [error]);

  return <RouteError error={error} reset={reset} />;
}
