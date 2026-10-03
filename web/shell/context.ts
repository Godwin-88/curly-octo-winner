import type { Ctx, Session } from '@/framework/types';

/** The word in the address for "no group chosen" or "no school chosen". */
export const ALL = 'all';

/**
 * The context lives in the address: /w/{group}/{school}/{module}/{section}/{record}.
 * A link therefore reopens the same school. What the address asks for is
 * narrowed to what the session allows; the server decides the real scope and
 * answers 404 outside it.
 */
export function resolveContext(session: Session, group: string | undefined, school: string | undefined): Ctx {
  const groupId = session.groupId ?? (group && group !== ALL ? group : undefined);
  const schoolId = session.schoolId ?? (school && school !== ALL ? school : undefined);
  return { session, groupId, schoolId, scope: schoolId ? 'school' : 'portfolio' };
}

export function basePath(ctx: Pick<Ctx, 'groupId' | 'schoolId'>): string {
  return `/w/${ctx.groupId ?? ALL}/${ctx.schoolId ?? ALL}`;
}

export function pathTo(
  ctx: Pick<Ctx, 'groupId' | 'schoolId'>,
  moduleId: string,
  sectionId?: string,
  recordId?: string,
): string {
  return [basePath(ctx), moduleId, sectionId, recordId && encodeURIComponent(recordId)].filter(Boolean).join('/');
}

/**
 * The sidebar's links are written without a context (/communications/messages).
 * This maps a workspace address back to that form so the right item is lit.
 */
export function navPath(pathname: string): string {
  const match = /^\/w\/[^/]+\/[^/]+((?:\/[^/]+){0,2})/.exec(pathname);
  return match ? match[1] || '/' : pathname;
}

// --- The school the API client is working in -------------------------------
//
// The screens that have not moved to /w/... yet have no school in their
// address. For them the chosen school is remembered per tab, so two tabs can
// hold two schools without one changing the other.

const STORAGE_KEY = 'shule360.school';

export function rememberSchool(schoolId: string | undefined): void {
  if (typeof window === 'undefined') return;
  if (schoolId) window.sessionStorage.setItem(STORAGE_KEY, schoolId);
  else window.sessionStorage.removeItem(STORAGE_KEY);
}

export function rememberedSchool(): string | undefined {
  if (typeof window === 'undefined') return undefined;
  return window.sessionStorage.getItem(STORAGE_KEY) ?? undefined;
}

/** Set while a form has edits that are not saved, so a context switch can ask first. */
export const unsaved = { dirty: false };

export function confirmLeave(): boolean {
  if (!unsaved.dirty) return true;
  const leave = window.confirm('You have changes that are not saved. Discard them?');
  if (leave) unsaved.dirty = false;
  return leave;
}
