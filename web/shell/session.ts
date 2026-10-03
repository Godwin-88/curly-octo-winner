'use client';

import { useMemo } from 'react';
import { useAuth } from '@/lib/auth';
import type { Session } from '@/framework/types';

/** The signed-in staff user as the shell sees them, or null while signed out or still loading. */
export function useShellSession(): { ready: boolean; session: Session | null } {
  const { ready, staff, session } = useAuth();
  const shellSession = useMemo<Session | null>(() => {
    if (!staff || !session) return null;
    return {
      scope: session.scope,
      name: staff.full_name,
      role: staff.role,
      schoolId: session.scope === 'school' ? session.school_id ?? staff.tenant_id : undefined,
      groupId: session.scope === 'group' ? session.group_id : undefined,
    };
  }, [staff, session]);
  return { ready, session: shellSession };
}
