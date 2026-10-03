'use client';

import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useEffect, useMemo } from 'react';
import { AdminChrome } from '@/components/layout/AdminChrome';
import { ResourceSection } from '@/framework/ResourceSection';
import { visibleModules, visibleSections } from '@/framework/registry';
import { useSchoolModules } from './ContextBar';
import type { Session } from '@/framework/types';
import { setSchoolContext } from '@/lib/api';
import { Empty, Spinner } from '@/ui/kit';
import { basePath, confirmLeave, pathTo, rememberSchool, resolveContext, unsaved } from './context';

/**
 * The frame every list → view → edit screen lives in: context bar, sidebar, the module's
 * sections, and the workspace. It knows nothing about any module beyond its manifest.
 */
export function Shell({ session }: { session: Session }) {
  const params = useParams<{ group: string; school: string; path?: string[] }>();
  const router = useRouter();
  const [moduleId, sectionId, recordParam] = params.path ?? [];
  const record = recordParam ? decodeURIComponent(recordParam) : undefined;
  const ctx = useMemo(() => resolveContext(session, params.group, params.school), [session, params.group, params.school]);

  // Every request made below acts on the school in the address. Set during
  // render (not in an effect) so the first query of a new school already carries it.
  setSchoolContext(session.scope === 'school' ? undefined : ctx.schoolId);
  useEffect(() => {
    if (session.scope !== 'school') rememberSchool(ctx.schoolId);
  }, [session.scope, ctx.schoolId]);

  useEffect(() => {
    const warn = (event: BeforeUnloadEvent) => {
      if (unsaved.dirty) event.preventDefault();
    };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, []);

  const home = basePath(ctx);
  const enabled = useSchoolModules(ctx.schoolId);
  const modules = visibleModules(ctx, ctx.scope === 'school' ? enabled : undefined);
  const current = modules.find((candidate) => candidate.id === moduleId);
  const sections = current ? visibleSections(ctx, current) : [];
  const section = sections.find((candidate) => candidate.id === sectionId);

  // Where the address should point, when it does not point at something this
  // session can open: the scope it has, the first module, the first section.
  let redirect: string | undefined;
  if (`/w/${params.group}/${params.school}` !== home) {
    redirect = [home, ...(params.path ?? [])].join('/');
  } else if (modules.length > 0 && !current) {
    redirect = pathTo(ctx, modules[0].id);
  } else if (current && !section && sections.length > 0) {
    redirect = pathTo(ctx, current.id, sections[0].id);
  }
  useEffect(() => {
    if (redirect) router.replace(redirect);
  }, [redirect, router]);

  // Switching keeps the module and returns to its list.
  const onSwitch = (group: string, school: string) => {
    router.push(`/w/${group}/${school}${current ? `/${current.id}` : ''}`);
  };

  if (redirect) {
    return <Spinner />;
  }

  if (!current || !section) {
    return (
      <AdminChrome ctx={ctx} onSwitch={onSwitch}>
        <Empty title="Nothing is available here for your role">Ask your administrator to check the role on your account.</Empty>
      </AdminChrome>
    );
  }

  const guard = (event: React.MouseEvent) => {
    if (!confirmLeave()) event.preventDefault();
  };

  return (
    <AdminChrome ctx={ctx} onSwitch={onSwitch} flush>
      {sections.length > 1 && (
        <nav aria-label={`${current.label} sections`} className="flex gap-1 overflow-x-auto border-b border-gray-200 bg-white px-3 py-1.5">
          {sections.map((candidate) => {
            const active = candidate.id === section.id;
            return (
              <Link
                key={candidate.id}
                href={pathTo(ctx, current.id, candidate.id)}
                onClick={guard}
                aria-current={active ? 'page' : undefined}
                className={`whitespace-nowrap rounded-lg px-3 py-1.5 text-sm ${active ? 'bg-blue-50 font-bold text-blue-900' : 'font-medium text-gray-700 hover:bg-gray-50'}`}
              >
                {candidate.label}
              </Link>
            );
          })}
        </nav>
      )}

      <div className="min-h-0 flex-1 overflow-auto p-3">
        {/* The key remounts the section when the school or the section changes, so no state crosses schools. */}
        {section.kind === 'page' ? (
          <section.page key={`${home}/${current.id}/${section.id}`} ctx={ctx} />
        ) : (
          <ResourceSection key={`${home}/${current.id}/${section.id}`} moduleId={current.id} def={section} ctx={ctx} record={record} />
        )}
      </div>
    </AdminChrome>
  );
}
