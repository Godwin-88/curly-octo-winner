'use client';

// The list → view → edit workspace. The address carries the context:
//   /w/{group}/{school}/{module}/{section}/{record}?do={action}
// ("all" = not chosen). See shell/context.ts.

import { Suspense, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Shell } from '@/shell/Shell';
import { useShellSession } from '@/shell/session';

export default function WorkspacePage() {
  const { ready, session } = useShellSession();
  const router = useRouter();

  useEffect(() => {
    if (ready && !session) router.replace('/auth/login');
  }, [ready, session, router]);

  if (!ready || !session) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-sm text-gray-600">{ready ? 'Redirecting to sign in…' : 'Loading…'}</div>
      </div>
    );
  }

  return (
    <Suspense>
      <Shell session={session} />
    </Suspense>
  );
}
