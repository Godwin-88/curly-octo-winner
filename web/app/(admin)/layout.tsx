'use client';

// Auth guard and frame for the admin screens that have not moved to the
// list → view → edit workspace (/w/...) yet. Renders a loading state until the
// session has been hydrated, then redirects anonymous visitors to sign-in.
// (web/proxy.ts performs the same check on the edge via the session cookie.)
//
// These screens have no school in their address. School staff do not need one
// (the API pins them to their school). A platform or group user works in the
// school they last opened in this tab; with none chosen they are sent to pick one.

import { useEffect, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { AdminChrome } from '@/components/layout/AdminChrome';
import { setSchoolContext } from '@/lib/api';
import { ALL, rememberSchool, rememberedSchool, resolveContext } from '@/shell/context';
import { useShellSession } from '@/shell/session';

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { ready, session } = useShellSession();
  const router = useRouter();

  // Guard on the verified staff session (hydrated from the HttpOnly cookie),
  // NOT on a token: the token only exists in memory after a sign-in, so
  // guarding on it bounced every page to the sign-in form on any hard reload.
  useEffect(() => {
    if (ready && !session) {
      router.replace('/auth/login');
    }
  }, [ready, session, router]);

  const ctx = useMemo(
    () => (session ? resolveContext(session, undefined, session.scope === 'school' ? undefined : rememberedSchool()) : null),
    [session]
  );
  const mustChoose = Boolean(ctx && ctx.scope === 'portfolio');
  useEffect(() => {
    if (mustChoose) router.replace(`/w/${ALL}/${ALL}`);
  }, [mustChoose, router]);

  if (!ready || !session || !ctx || mustChoose) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-sm text-gray-600">
          {ready && !session ? 'Redirecting to sign in…' : 'Loading…'}
        </div>
      </div>
    );
  }

  setSchoolContext(session.scope === 'school' ? undefined : ctx.schoolId);

  // These screens load their data once, so a switch reloads the page: nothing
  // read for the previous school stays on screen.
  const onSwitch = (group: string, school: string) => {
    if (school === ALL) {
      router.push(`/w/${group}/${ALL}`);
      return;
    }
    rememberSchool(school);
    window.location.reload();
  };

  return (
    <AdminChrome ctx={ctx} onSwitch={onSwitch}>
      {children}
    </AdminChrome>
  );
}
