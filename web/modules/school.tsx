'use client';

import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import { apiRequest } from '@/lib/api';
import { page, resource, type Ctx, type FormField, type FormValues, type ModuleManifest, type Reveal } from '@/framework/types';
import { formatPhone } from '@/lib/phone';
import { ErrorNote, Spinner, Status, when } from '@/ui/kit';

// Setting a school up: what is still to do, and the people who can sign in.

const ADMINS = ['principal', 'super_admin'];

// --- Getting started --------------------------------------------------------

interface Step {
  id: string;
  title: string;
  detail: string;
  done: boolean;
  href: string;
  optional?: boolean;
}

interface SetupStatus {
  school_name: string;
  steps: Step[];
  done: number;
  total: number;
}

function Setup({ ctx }: { ctx: Ctx }) {
  const status = useQuery({
    queryKey: ['school', ctx.schoolId, 'setup'],
    queryFn: () => apiRequest<SetupStatus>('/onboarding/status'),
  });
  const data = status.data;
  const mayManageUsers = ADMINS.includes(ctx.session.role);

  return (
    <div className="mx-auto max-w-3xl space-y-4 p-4">
      <header>
        <h1 className="text-lg font-bold">Getting started</h1>
        <p className="text-sm text-gray-600">
          What a school sets up before it runs on Shule360. Each step is ticked from what is actually recorded, so it stays true as you work.
        </p>
      </header>
      {status.isLoading && <Spinner />}
      {status.error && <ErrorNote error={status.error} />}
      {data && (
        <>
          <div className="rounded-xl border border-gray-200 bg-white p-4">
            <p className="text-sm font-semibold">{data.school_name}: {data.done} of {data.total} done</p>
            <div
              className="mt-2 h-2 overflow-hidden rounded-full bg-gray-200"
              role="progressbar" aria-valuemin={0} aria-valuemax={data.total} aria-valuenow={data.done} aria-label="Setup progress"
            >
              <div className="h-full bg-emerald-600" style={{ width: `${(data.done / data.total) * 100}%` }} />
            </div>
          </div>
          <ol className="space-y-2">
            {data.steps.map((step, index) => {
              const hidden = step.id === 'users' && !mayManageUsers;
              return (
                <li key={step.id} className="flex items-start gap-3 rounded-xl border border-gray-200 bg-white p-4">
                  <span
                    aria-hidden="true"
                    className={`mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-bold ${step.done ? 'bg-emerald-600 text-white' : 'border border-gray-400 text-gray-700'}`}
                  >
                    {step.done ? '✓' : index + 1}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="font-semibold">
                      {step.title}
                      <span className="ml-2 text-xs font-normal text-gray-600">
                        {step.done ? 'Done' : step.optional ? 'Optional' : 'To do'}
                      </span>
                    </p>
                    <p className="mt-0.5 text-sm text-gray-700">{step.detail}</p>
                  </div>
                  {!hidden && (
                    <Link
                      href={step.href}
                      className={`shrink-0 rounded-lg px-3 py-1.5 text-sm font-semibold ${step.done ? 'border border-gray-300 bg-white text-gray-900 hover:bg-gray-50' : 'bg-blue-700 text-white hover:bg-blue-800'}`}
                    >
                      {step.done ? 'Open' : 'Start'}<span className="sr-only">: {step.title}</span>
                    </Link>
                  )}
                </li>
              );
            })}
          </ol>
        </>
      )}
    </div>
  );
}

const setup = page({
  id: 'setup',
  label: 'Getting started',
  scopes: ['school'],
  purpose: 'What to set up first, and what is already done.',
  page: Setup,
});

// --- Users ------------------------------------------------------------------

interface SchoolUser {
  id: string;
  full_name: string;
  email: string;
  phone?: string;
  role: string;
  is_active: boolean;
  created_at: string;
  temp_password?: string;
  note?: string;
}

const ROLES = [
  { value: 'teacher', label: 'Teacher', hint: 'Learners, attendance and assessments.' },
  { value: 'bursar', label: 'Bursar', hint: 'Fees, payments and procurement.' },
  { value: 'principal', label: 'Principal', hint: 'Everything in the school, including users.' },
  { value: 'hr', label: 'HR', hint: 'Staff records, leave and payroll.' },
  { value: 'transport_manager', label: 'Transport manager', hint: 'Vehicles, routes and trips.' },
];

const roleLabel = (role: string) => ROLES.find((candidate) => candidate.value === role)?.label ?? (role === 'super_admin' ? 'Shule360 administrator' : role);

const personFields: FormField[] = [
  { name: 'full_name', label: 'Full name', type: 'text', required: true },
  {
    name: 'role', label: 'What they do', type: 'select', required: true,
    options: ROLES.map((role) => ({ value: role.value, label: `${role.label} — ${role.hint}` })),
    help: 'This decides what they can see and change.',
  },
  { name: 'phone', label: 'Phone', type: 'text', placeholder: '0712 345 678' },
];

/** The password of a new or reset account: shown once, never retrievable. */
function passwordReveal(user: SchoolUser): Reveal | undefined {
  if (user.temp_password) {
    return {
      title: `Sign-in details for ${user.full_name}`,
      note: 'Give these to them in person or by phone. The password is shown this once and cannot be shown again; if it is lost, reset it.',
      items: [
        { label: 'Email', value: user.email },
        { label: 'Password', value: user.temp_password },
      ],
    };
  }
  if (user.note) return { title: `${user.full_name} was added`, note: user.note, items: [{ label: 'Email', value: user.email }] };
  return undefined;
}

const isMe = (row: SchoolUser, ctx: Ctx) => row.full_name === ctx.session.name;

const users = resource<SchoolUser>({
  id: 'users',
  label: 'Users',
  noun: 'user',
  scopes: ['school'],
  roles: ADMINS,
  purpose: 'The people who can sign in to this school, and what each may do.',
  list: async (_ctx, filters) => {
    const query = filters.search ? `?search=${encodeURIComponent(filters.search)}` : '';
    return { items: (await apiRequest<SchoolUser[] | null>(`/school/users${query}`)) ?? [] };
  },
  get: (_ctx, id) => apiRequest<SchoolUser>(`/school/users/${id}`),
  rowId: (row) => row.id,
  title: (row) => row.full_name,
  status: (row) => (row.is_active ? 'active' : 'deactivated'),
  filters: [{ name: 'search', label: 'Search', type: 'text', placeholder: 'Name or email' }],
  columns: [
    { header: 'Name', cell: (row) => row.full_name },
    { header: 'Role', cell: (row) => roleLabel(row.role) },
    { header: 'Status', cell: (row) => <Status value={row.is_active ? 'active' : 'deactivated'} /> },
  ],
  fields: [
    { label: 'Email (their sign-in)', value: (row) => row.email },
    { label: 'Role', value: (row) => roleLabel(row.role) },
    { label: 'Phone', value: (row) => (row.phone ? formatPhone(row.phone) : '—') },
    { label: 'Added', value: (row) => when(row.created_at) },
  ],
  create: {
    id: 'create',
    label: 'Add user',
    fields: [
      personFields[0],
      { name: 'email', label: 'Email', type: 'text', required: true, placeholder: 'name@school.ac.ke', help: 'They sign in with this. It cannot be changed later.' },
      ...personFields.slice(1),
    ],
    initial: () => ({ role: 'teacher' }),
    confirm: 'A password is created for them and shown to you once, to hand over.',
    run: (_ctx, _row, values: FormValues) =>
      apiRequest<SchoolUser>('/school/users', {
        method: 'POST',
        body: { full_name: values.full_name, email: values.email, phone: values.phone ?? '', role: values.role },
      }),
    createdId: (user: SchoolUser) => user.id,
    reveal: passwordReveal,
  },
  actions: [
    {
      id: 'edit',
      label: 'Edit',
      when: (row) => row.role !== 'super_admin',
      fields: personFields,
      initial: (row) => ({ full_name: row.full_name, role: row.role, phone: row.phone ?? '' }),
      confirm: 'A change of role applies from their next click; they do not need to sign in again.',
      submitLabel: 'Save',
      run: (_ctx, row, values) =>
        apiRequest<SchoolUser>(`/school/users/${row.id}`, {
          method: 'PATCH',
          body: { full_name: values.full_name, phone: values.phone ?? '', role: values.role },
        }),
    },
    {
      id: 'reset-password',
      label: 'Reset password',
      when: (row) => row.is_active && row.role !== 'super_admin',
      confirm: 'Their current password stops working and a new one is shown to you once, to hand over.',
      submitLabel: 'Reset the password',
      run: (_ctx, row) => apiRequest<SchoolUser>(`/school/users/${row.id}/reset-password`, { method: 'POST' }),
      reveal: passwordReveal,
    },
    {
      id: 'deactivate',
      label: 'Deactivate',
      tone: 'danger',
      when: (row, ctx) => row.is_active && row.role !== 'super_admin' && !isMe(row, ctx),
      confirm: 'They are signed out at once and cannot sign in again. Everything they recorded is kept, and the account can be reactivated.',
      submitLabel: 'Deactivate this user',
      run: (_ctx, row) => apiRequest<SchoolUser>(`/school/users/${row.id}/deactivate`, { method: 'POST' }),
    },
    {
      id: 'activate',
      label: 'Reactivate',
      when: (row) => !row.is_active,
      confirm: 'They can sign in again with the password they had. Reset it if they no longer know it.',
      submitLabel: 'Reactivate this user',
      run: (_ctx, row) => apiRequest<SchoolUser>(`/school/users/${row.id}/activate`, { method: 'POST' }),
    },
  ],
});

export const school: ModuleManifest = {
  id: 'school',
  label: 'School setup',
  sections: [setup, users],
};
