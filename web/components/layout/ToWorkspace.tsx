'use client';

// Keeps the old Communications addresses working: each one forwards to its
// place in the list → view → edit workspace, in the school the user is in.

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { pathTo, rememberedSchool, resolveContext } from '@/shell/context';
import { useShellSession } from '@/shell/session';

export function ToWorkspace({ module, section, record, action }: {
  module: string;
  section: string;
  record?: string;
  action?: string;
}) {
  const { ready, session } = useShellSession();
  const router = useRouter();

  useEffect(() => {
    if (!ready || !session) return;
    const ctx = resolveContext(session, undefined, session.scope === 'school' ? undefined : rememberedSchool());
    router.replace(pathTo(ctx, module, section, record) + (action ? `?do=${action}` : ''));
  }, [ready, session, router, module, section, record, action]);

  return <p className="p-6 text-sm text-gray-600" role="status">Opening…</p>;
}
