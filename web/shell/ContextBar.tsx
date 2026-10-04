'use client';

import { useQuery } from '@tanstack/react-query';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { apiRequest } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import type { Ctx } from '@/framework/types';
import { ALL, confirmLeave, rememberedSchool } from './context';

export interface School {
  id: string;
  name: string;
  slug: string;
  group_id?: string;
  group_name?: string;
  ownership?: 'public' | 'private';
  /** The modules this school has. */
  modules?: string[];
}

/** The schools this session may open. One request, shared by every screen. */
export function useSchools(enabled = true) {
  return useQuery({
    queryKey: ['context', 'schools'],
    queryFn: () => apiRequest<School[]>('/schools'),
    enabled,
    staleTime: 5 * 60_000,
  });
}

interface Choice {
  value: string;
  label: string;
  hint?: string;
}

/** A crumb that opens a searchable list. Escape or a click outside closes it. */
function Crumb({ label, current, choices, onChoose }: {
  label: string;
  current: string;
  choices: Choice[];
  onChoose: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [filter, setFilter] = useState('');
  const box = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const close = (event: MouseEvent | KeyboardEvent) => {
      if (event instanceof KeyboardEvent ? event.key === 'Escape' : !box.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', close);
    document.addEventListener('keydown', close);
    return () => {
      document.removeEventListener('mousedown', close);
      document.removeEventListener('keydown', close);
    };
  }, [open]);

  const shown = choices.filter((choice) => `${choice.label} ${choice.hint ?? ''}`.toLowerCase().includes(filter.toLowerCase()));
  return (
    <div ref={box} className="relative min-w-0">
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        className="flex max-w-[16rem] items-center gap-1.5 rounded-lg border border-gray-300 bg-white px-2.5 py-1.5 text-left text-sm hover:bg-gray-50"
      >
        <span className="hidden text-xs font-semibold uppercase tracking-wide text-gray-600 sm:inline">{label}</span>
        <span className="truncate font-semibold">{current}</span>
        <span aria-hidden="true" className="text-gray-600">▾</span>
      </button>
      {open && (
        <div className="absolute left-0 z-50 mt-1 w-72 rounded-xl border border-gray-200 bg-white p-2 shadow-lg">
          <input
            autoFocus
            aria-label={`Search ${label.toLowerCase()}`}
            placeholder="Search…"
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
            className="mb-2 w-full rounded-lg border border-gray-300 px-2.5 py-1.5 text-sm"
          />
          <ul role="listbox" aria-label={label} className="max-h-64 overflow-auto">
            {shown.map((choice) => (
              <li key={choice.value} role="option" aria-selected={choice.label === current}>
                <button
                  type="button"
                  onClick={() => {
                    setOpen(false);
                    onChoose(choice.value);
                  }}
                  className="flex w-full items-baseline justify-between gap-2 rounded-lg px-2.5 py-1.5 text-left text-sm hover:bg-blue-50"
                >
                  <span className="truncate font-semibold">{choice.label}</span>
                  {choice.hint && <span className="shrink-0 text-xs text-gray-600">{choice.hint}</span>}
                </button>
              </li>
            ))}
            {shown.length === 0 && <li className="px-2.5 py-2 text-sm text-gray-600">Nothing matches.</li>}
          </ul>
        </div>
      )}
    </div>
  );
}

function Fixed({ label, children }: { label: string; children: ReactNode }) {
  return (
    <span className="flex min-w-0 items-center gap-1.5 px-1 text-sm">
      <span className="hidden text-xs font-semibold uppercase tracking-wide text-gray-600 sm:inline">{label}</span>
      <span className="truncate font-semibold">{children}</span>
    </span>
  );
}

const ROLE_LABEL: Record<string, string> = {
  super_admin: 'Super admin', principal: 'Principal', teacher: 'Teacher', bursar: 'Bursar',
  transport_manager: 'Transport manager', hr: 'HR',
};

/**
 * The modules the school in view has, or undefined while that is not known
 * yet (nothing is hidden until it is). Without an id it is the one school the
 * session has, or the one last opened.
 */
export function useSchoolModules(schoolId?: string): string[] | undefined {
  const schools = useSchools();
  const all = schools.data;
  if (!all) return undefined;
  const id = schoolId ?? (all.length === 1 ? all[0].id : rememberedSchool());
  return all.find((school) => school.id === id)?.modules;
}

/**
 * The context switch: Group › School. A level the user can change is a drop-down; a level fixed
 * by their account is plain text. Choosing a level calls onSwitch, which returns to a list: a
 * record from the old school is never carried into the new one.
 */
export function ContextBar({ ctx, onSwitch, leading }: {
  ctx: Ctx;
  onSwitch: (group: string, school: string) => void;
  leading?: ReactNode;
}) {
  const { logoutStaff } = useAuth();
  const { session } = ctx;
  const schools = useSchools();
  const all = schools.data ?? [];
  const chosen = all.find((school) => school.id === ctx.schoolId);
  const groups = new Map<string, string>();
  for (const school of all) {
    if (school.group_id) groups.set(school.group_id, school.group_name ?? 'Unnamed group');
  }
  const inGroup = ctx.groupId ? all.filter((school) => school.group_id === ctx.groupId) : all;
  const groupName = ctx.groupId ? groups.get(ctx.groupId) : chosen?.group_name;

  function go(group: string, school: string) {
    if (!confirmLeave()) return;
    onSwitch(group, school);
  }

  return (
    <header className="flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-gray-200 bg-white px-3 py-2">
      {leading}
      <nav aria-label="Context" className="flex min-w-0 flex-1 items-center gap-1.5">
        {session.scope === 'platform' ? (
          <Crumb
            label="Group"
            current={ctx.groupId ? groupName ?? 'Group' : 'All groups'}
            choices={[
              { value: ALL, label: 'All groups' },
              ...Array.from(groups).map(([id, name]) => ({
                value: id,
                label: name,
                hint: `${all.filter((school) => school.group_id === id).length} schools`,
              })),
            ]}
            onChoose={(value) => go(value, ALL)}
          />
        ) : (
          groupName && <Fixed label="Group">{groupName}</Fixed>
        )}

        {(session.scope === 'platform' || groupName) && <span aria-hidden="true" className="text-gray-500">›</span>}

        {session.scope === 'school' ? (
          <Fixed label="School">{chosen?.name ?? (schools.isLoading ? '…' : 'Your school')}</Fixed>
        ) : (
          <Crumb
            label="School"
            current={ctx.schoolId ? chosen?.name ?? 'School' : 'All schools'}
            choices={[
              { value: ALL, label: 'All schools' },
              ...inGroup.map((school) => ({ value: school.id, label: school.name, hint: school.group_name })),
            ]}
            onChoose={(value) => {
              const owner = all.find((school) => school.id === value)?.group_id;
              go(session.scope === 'platform' ? ctx.groupId ?? owner ?? ALL : ALL, value);
            }}
          />
        )}
      </nav>

      <div className="flex items-center gap-2">
        <div className="hidden text-right leading-tight sm:block">
          <p className="text-sm font-semibold">{session.name}</p>
          <p className="text-xs text-gray-600">{ROLE_LABEL[session.role] ?? session.role}</p>
        </div>
        <button
          type="button"
          onClick={() => { if (confirmLeave()) logoutStaff(); }}
          className="rounded-lg border border-gray-300 bg-white px-2.5 py-1.5 text-xs font-semibold hover:bg-gray-50"
        >
          Sign out
        </button>
      </div>
    </header>
  );
}
