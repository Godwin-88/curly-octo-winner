'use client';

import Link from 'next/link';
import { apiRequest } from '@/lib/api';
import { resource, type Ctx, type FormField, type FormValues, type ModuleManifest, type Option, type Reveal } from '@/framework/types';
import type { School } from '@/shell/ContextBar';
import { ALL } from '@/shell/context';
import { Status, when } from '@/ui/kit';

// What sits above a school: the schools themselves, school groups, and the
// people who can open more than one school. Shown when no school is chosen.
// Groups and users are for platform administrators; a group user sees only
// the schools of their group.

interface Group {
  id: string;
  name: string;
  slug: string;
  school_count: number;
  user_count: number;
  created_at: string;
}

interface PlatformUser {
  id: string;
  email: string;
  full_name: string;
  scope: 'platform' | 'group';
  group_id?: string;
  group_name?: string;
  role: string;
  is_active: boolean;
  created_at: string;
  temp_password?: string;
  note?: string;
}

const isPlatform = (_row: unknown, ctx: Ctx) => ctx.session.scope === 'platform';

const groupOptions = async (): Promise<Option[]> =>
  (await apiRequest<Group[]>('/platform/groups')).map((group) => ({ value: group.id, label: group.name }));

// --- Schools ----------------------------------------------------------------

function OpenSchool({ row, ctx }: { row: School; ctx: Ctx }) {
  const group = ctx.session.scope === 'platform' ? row.group_id ?? ALL : ALL;
  return (
    <p className="text-sm">
      <Link
        className="inline-flex rounded-lg bg-blue-700 px-3.5 py-2 font-semibold text-white hover:bg-blue-800"
        href={`/w/${group}/${row.id}`}
      >
        Open this school
      </Link>
      <span className="ml-3 text-gray-600">Switches the context to this school: its messages, learners and records.</span>
    </p>
  );
}

const schoolFields: FormField[] = [
  { name: 'name', label: 'School name', type: 'text', required: true, placeholder: 'Jua Kali Primary School' },
  { name: 'group_id', label: 'Group', type: 'select', options: groupOptions, help: 'Leave as "Any" for a school that belongs to no group.' },
];

const schools = resource<School>({
  id: 'schools',
  label: 'Schools',
  noun: 'school',
  scopes: ['portfolio'],
  purpose: 'Every school you can open. Open one to work inside it; the bar above then shows that school.',
  list: async (ctx) => {
    const items = await apiRequest<School[]>('/schools');
    return { items: items.filter((school) => !ctx.groupId || school.group_id === ctx.groupId) };
  },
  rowId: (row) => row.id,
  title: (row) => row.name,
  columns: [
    { header: 'School', cell: (row) => row.name },
    { header: 'Group', cell: (row) => row.group_name ?? '—' },
  ],
  fields: [
    { label: 'Group', value: (row) => row.group_name ?? 'Not in a group' },
    { label: 'Short name', value: (row) => row.slug },
  ],
  extra: OpenSchool,
  create: {
    id: 'create',
    label: 'Add school',
    when: isPlatform,
    fields: [
      ...schoolFields,
      { name: 'slug', label: 'Short name', type: 'text', placeholder: 'juakali-primary', help: 'Lowercase letters, numbers and dashes. Leave empty to make one from the name. It cannot be changed later.' },
    ],
    initial: (_row, ctx) => ({ group_id: ctx.groupId ?? '' }),
    run: (_ctx, _row, values) => apiRequest<School>('/platform/schools', { method: 'POST', body: schoolBody(values) }),
    createdId: (school: School) => school.id,
  },
  actions: [
    {
      id: 'edit',
      label: 'Rename or move',
      when: isPlatform,
      fields: schoolFields,
      initial: (row) => ({ name: row.name, group_id: row.group_id ?? '' }),
      confirm: 'Moving a school changes which group users can open it, from their next request.',
      submitLabel: 'Save changes',
      run: (_ctx, row, values) => apiRequest<School>(`/platform/schools/${row.id}`, { method: 'PATCH', body: schoolBody(values) }),
    },
  ],
});

function schoolBody(values: FormValues) {
  return { name: values.name, slug: values.slug ?? '', group_id: values.group_id || null };
}

// --- Groups -----------------------------------------------------------------

const groups = resource<Group>({
  id: 'groups',
  label: 'Groups',
  noun: 'group',
  scopes: ['portfolio'],
  sessions: ['platform'],
  purpose: 'An owner of several schools, such as a trust or a diocese. A group user can open every school in their group.',
  list: async () => ({ items: await apiRequest<Group[]>('/platform/groups') }),
  get: (_ctx, id) => apiRequest<Group>(`/platform/groups/${id}`),
  rowId: (row) => row.id,
  title: (row) => row.name,
  columns: [
    { header: 'Group', cell: (row) => row.name },
    { header: 'Schools', cell: (row) => row.school_count, align: 'right' },
    { header: 'Users', cell: (row) => row.user_count, align: 'right' },
  ],
  fields: [
    { label: 'Schools', value: (row) => (row.school_count === 0 ? 'None yet: add or move a school into this group under Schools.' : row.school_count) },
    { label: 'Active users', value: (row) => row.user_count },
    { label: 'Short name', value: (row) => row.slug },
    { label: 'Created', value: (row) => when(row.created_at) },
  ],
  create: {
    id: 'create',
    label: 'Add group',
    fields: [
      { name: 'name', label: 'Group name', type: 'text', required: true, placeholder: 'Jua Kali Schools Trust' },
      { name: 'slug', label: 'Short name', type: 'text', help: 'Leave empty to make one from the name.' },
    ],
    run: (_ctx, _row, values) => apiRequest<Group>('/platform/groups', { method: 'POST', body: { name: values.name, slug: values.slug ?? '' } }),
    createdId: (group: Group) => group.id,
  },
  actions: [
    {
      id: 'rename',
      label: 'Rename',
      fields: [{ name: 'name', label: 'Group name', type: 'text', required: true }],
      initial: (row) => ({ name: row.name }),
      submitLabel: 'Save',
      run: (_ctx, row, values) => apiRequest<Group>(`/platform/groups/${row.id}`, { method: 'PATCH', body: { name: values.name } }),
    },
  ],
});

// --- Users ------------------------------------------------------------------

const ROLE_OPTIONS: Option[] = [
  { value: 'super_admin', label: 'Super admin: everything in a school' },
  { value: 'principal', label: 'Principal: everything in a school' },
  { value: 'bursar', label: 'Bursar: finance and procurement' },
  { value: 'hr', label: 'HR' },
  { value: 'transport_manager', label: 'Transport manager' },
  { value: 'teacher', label: 'Teacher' },
];

const userFields: FormField[] = [
  { name: 'full_name', label: 'Full name', type: 'text', required: true },
  {
    name: 'scope', label: 'Can open', type: 'select', required: true,
    options: [
      { value: 'platform', label: 'Every school (platform administrator)' },
      { value: 'group', label: 'The schools of one group' },
    ],
    help: 'A platform administrator can also manage schools, groups and these users.',
  },
  { name: 'group_id', label: 'Group', type: 'select', required: true, options: groupOptions, showIf: (values) => values.scope === 'group' },
  { name: 'role', label: 'Role inside a school', type: 'select', required: true, options: ROLE_OPTIONS, help: 'What they may do once they open a school.' },
];

function userBody(values: FormValues) {
  return {
    email: values.email,
    full_name: values.full_name,
    scope: values.scope,
    group_id: values.scope === 'group' ? values.group_id : null,
    role: values.role,
  };
}

/** The password is shown once, here; it is not stored and cannot be shown again. */
function passwordReveal(user: PlatformUser): Reveal | undefined {
  if (!user.temp_password) {
    return user.note ? { title: 'Account created', note: user.note, items: [] } : undefined;
  }
  return {
    title: `Password for ${user.full_name}`,
    note: 'Pass this on securely (not by email with the address). It is shown only now and cannot be recovered; if it is lost, reset the password.',
    items: [
      { label: 'Sign-in email', value: user.email },
      { label: 'Password', value: user.temp_password },
    ],
  };
}

const users = resource<PlatformUser>({
  id: 'users',
  label: 'Users',
  noun: 'user',
  scopes: ['portfolio'],
  sessions: ['platform'],
  purpose: 'People who can open more than one school. Staff of a single school are managed inside that school.',
  list: async () => ({ items: await apiRequest<PlatformUser[]>('/platform/users') }),
  get: (_ctx, id) => apiRequest<PlatformUser>(`/platform/users/${id}`),
  rowId: (row) => row.id,
  title: (row) => row.full_name,
  status: (row) => (row.is_active ? 'active' : 'deactivated'),
  columns: [
    { header: 'Name', cell: (row) => row.full_name },
    { header: 'Can open', cell: (row) => (row.scope === 'platform' ? 'Every school' : row.group_name ?? 'A group') },
    { header: 'Status', cell: (row) => <Status value={row.is_active ? 'active' : 'deactivated'} /> },
  ],
  fields: [
    { label: 'Sign-in email', value: (row) => row.email },
    { label: 'Can open', value: (row) => (row.scope === 'platform' ? 'Every school, and this Platform area' : `The schools of ${row.group_name ?? 'their group'}`) },
    { label: 'Role inside a school', value: (row) => ROLE_OPTIONS.find((option) => option.value === row.role)?.label ?? row.role },
    { label: 'Added', value: (row) => when(row.created_at) },
  ],
  create: {
    id: 'create',
    label: 'Add user',
    fields: [
      { name: 'email', label: 'Email', type: 'text', required: true, placeholder: 'name@example.org', help: 'What they sign in with. It must not be the email of a school staff member.' },
      ...userFields,
    ],
    initial: () => ({ scope: 'group', role: 'principal' }),
    run: (_ctx, _row, values) => apiRequest<PlatformUser>('/platform/users', { method: 'POST', body: userBody(values) }),
    createdId: (user: PlatformUser) => user.id,
    reveal: passwordReveal,
  },
  actions: [
    {
      id: 'edit',
      label: 'Edit',
      fields: userFields,
      initial: (row) => ({ full_name: row.full_name, scope: row.scope, group_id: row.group_id ?? '', role: row.role }),
      submitLabel: 'Save changes',
      run: (_ctx, row, values) => apiRequest<PlatformUser>(`/platform/users/${row.id}`, { method: 'PATCH', body: userBody(values) }),
    },
    {
      id: 'reset-password',
      label: 'Reset password',
      when: (row) => row.is_active,
      confirm: 'Their current password stops working at once. A new one is generated and shown to you once.',
      submitLabel: 'Reset password',
      run: (_ctx, row) => apiRequest<PlatformUser>(`/platform/users/${row.id}/reset-password`, { method: 'POST' }),
      reveal: passwordReveal,
    },
    {
      id: 'deactivate',
      label: 'Deactivate',
      tone: 'danger',
      when: (row) => row.is_active,
      confirm: 'They are signed out of every school on their next click and cannot sign in again. What they did stays on record. You can reactivate them later.',
      run: (_ctx, row) => apiRequest<PlatformUser>(`/platform/users/${row.id}/deactivate`, { method: 'POST' }),
    },
    {
      id: 'activate',
      label: 'Reactivate',
      when: (row) => !row.is_active,
      run: (_ctx, row) => apiRequest<PlatformUser>(`/platform/users/${row.id}/activate`, { method: 'POST' }),
    },
  ],
});

export const platform: ModuleManifest = {
  id: 'platform',
  label: 'Platform',
  sections: [schools, groups, users],
};
