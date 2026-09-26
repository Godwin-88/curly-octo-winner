'use client';

// Settings — the tenant configuration centre.
//
// Tabs are chosen by what a school principal actually has to configure:
//   School        identity, contact, M-Pesa wiring
//   Operations    term, attendance window, document footers, feature switches
//   Integrations  provider credentials + live connectivity tests
//   Access        the signed-in user's own account, session and related tools
//
// Every value shown comes from the API. Reads are open to all staff; writes are
// enabled only for principal/super_admin (the API enforces this too).

import { useState } from 'react';
import Link from 'next/link';
import {
  Building2,
  FileBarChart,
  KeyRound,
  LogOut,
  Plug,
  SlidersHorizontal,
  UserCheck,
  UserCircle,
} from 'lucide-react';
import { useAuth } from '@/lib/auth';
import { Skeleton } from '@/components/ui/Skeleton';
import { IntegrationsPanel } from '@/components/settings/IntegrationsPanel';
import { SchoolSettingsForms } from '@/components/settings/SchoolSettingsForms';

const TABS = [
  { id: 'school', label: 'School', icon: Building2 },
  { id: 'operations', label: 'Operations', icon: SlidersHorizontal },
  { id: 'integrations', label: 'Integrations', icon: Plug },
  { id: 'access', label: 'Access', icon: UserCircle },
] as const;

type TabId = (typeof TABS)[number]['id'];

const MANAGEMENT_ROLES = ['super_admin', 'principal'];

const ROLE_LABELS: Record<string, string> = {
  super_admin: 'Super Admin',
  principal: 'Principal',
  teacher: 'Teacher',
  bursar: 'Bursar',
  transport_manager: 'Transport Manager',
  hr: 'HR Officer',
};

function initials(name: string) {
  return name
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? '')
    .join('');
}

function AccountCard({ staff }: { staff: { id: string; full_name: string; email: string; role: string; phone?: string } }) {
  return (
    <section className="card p-5" aria-labelledby="account-heading">
      <div className="flex items-center gap-4">
        <div
          aria-hidden="true"
          className="w-12 h-12 rounded-full bg-primary-50 text-primary-700 flex items-center justify-center text-lg font-semibold"
        >
          {initials(staff.full_name)}
        </div>
        <div>
          <h2 id="account-heading" className="text-base font-semibold text-gray-900">
            {staff.full_name}
          </h2>
          <p className="text-sm text-gray-600">{ROLE_LABELS[staff.role] ?? staff.role}</p>
        </div>
      </div>
      <dl className="mt-5 text-sm">
        <div className="flex gap-4 border-b border-gray-100 py-2.5">
          <dt className="w-40 shrink-0 text-gray-500">Email</dt>
          <dd className="font-medium text-gray-900 break-words">{staff.email || '—'}</dd>
        </div>
        <div className="flex gap-4 border-b border-gray-100 py-2.5">
          <dt className="w-40 shrink-0 text-gray-500">Phone</dt>
          <dd className="font-medium text-gray-900">{staff.phone || 'Not on file'}</dd>
        </div>
        <div className="flex gap-4 py-2.5">
          <dt className="w-40 shrink-0 text-gray-500">Staff ID</dt>
          <dd className="font-medium text-gray-900 break-all">{staff.id}</dd>
        </div>
      </dl>
    </section>
  );
}

function SessionCard({ canEdit, onSignOut }: { canEdit: boolean; onSignOut: () => void }) {
  return (
    <section className="card p-5" aria-labelledby="session-heading">
      <h2 id="session-heading" className="text-base font-semibold text-gray-900">
        Session &amp; security
      </h2>
      <ul className="mt-3 space-y-2 text-sm text-gray-600">
        <li>Your session lives in an HttpOnly cookie, so page scripts cannot read or steal it.</li>
        <li>It renews while you work and expires after 24 hours of inactivity.</li>
        <li>On a shared device, sign out when you are done.</li>
      </ul>
      <div className="mt-4 flex flex-wrap items-center gap-3 border-t border-gray-100 pt-4">
        <button
          type="button"
          onClick={onSignOut}
          className="btn-secondary inline-flex items-center gap-2 focus-visible:ring-2 focus-visible:ring-primary-500"
        >
          <LogOut size={15} aria-hidden="true" />
          Sign out
        </button>
        {canEdit && (
          <Link
            href="/security"
            className="text-sm font-medium text-primary-700 hover:underline focus-visible:ring-2 focus-visible:ring-primary-500 rounded"
          >
            Review access &amp; audit log
          </Link>
        )}
      </div>
    </section>
  );
}

function PrivacyCard({ canEdit }: { canEdit: boolean }) {
  const linkClass =
    'inline-flex items-center gap-2 text-sm font-medium text-primary-700 hover:underline focus-visible:ring-2 focus-visible:ring-primary-500 rounded';
  return (
    <section className="card p-5" aria-labelledby="privacy-heading">
      <h2 id="privacy-heading" className="text-base font-semibold text-gray-900">
        Data &amp; compliance
      </h2>
      <p className="mt-1 text-sm text-gray-600">
        {canEdit
          ? 'Processing register, consent records and data subject rights'
          : 'Handled by your school administrator under Digital Security'}
      </p>
      {canEdit && (
        <ul className="mt-3 space-y-2">
          <li>
            <Link href="/security/roles" className={linkClass}>
              <KeyRound size={15} aria-hidden="true" />
              Role-based access — who can view or change what
            </Link>
          </li>
          <li>
            <Link href="/security/data-protection" className={linkClass}>
              <FileBarChart size={15} aria-hidden="true" />
              Data processing register (KDPA)
            </Link>
          </li>
          <li>
            <Link href="/security/consent" className={linkClass}>
              <UserCheck size={15} aria-hidden="true" />
              Parent consent records
            </Link>
          </li>
        </ul>
      )}
    </section>
  );
}

export default function SettingsPage() {
  const { staff, ready, logoutStaff } = useAuth();
  const [tab, setTab] = useState<TabId>('school');

  const canEdit = !!staff && MANAGEMENT_ROLES.includes(staff.role);

  return (
    <div className="p-6 max-w-5xl">
      <header>
        <h1 className="text-2xl font-bold">Settings</h1>
        <p className="text-gray-500">
          {canEdit
            ? 'Configure your school, its payments and its provider credentials'
            : 'Your school configuration and account'}
        </p>
      </header>

      {!ready ? (
        <div className="mt-8 space-y-4" aria-hidden="true">
          <Skeleton className="h-10 w-full max-w-lg" />
          <div className="card p-5 space-y-3">
            <Skeleton className="h-4 w-48" />
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-9 w-2/3" />
          </div>
        </div>
      ) : !staff ? (
        <div className="card mt-8 p-6">
          <h2 className="text-lg font-semibold">No active session</h2>
          <p className="mt-1 text-sm text-gray-600">
            Your session could not be verified. Sign in again to manage your school.
          </p>
          <Link
            href="/auth/login"
            className="btn-primary mt-4 inline-block focus-visible:ring-2 focus-visible:ring-primary-500"
          >
            Go to sign in
          </Link>
        </div>
      ) : (
        <>
          <div className="mt-6 border-b border-gray-200">
            <div role="tablist" aria-label="Settings sections" className="flex flex-wrap gap-1 -mb-px">
              {TABS.map((t) => {
                const Icon = t.icon;
                const selected = tab === t.id;
                return (
                  <button
                    key={t.id}
                    role="tab"
                    id={`tab-${t.id}`}
                    aria-selected={selected}
                    aria-controls={`panel-${t.id}`}
                    type="button"
                    onClick={() => setTab(t.id)}
                    className={`inline-flex items-center gap-2 rounded-t-lg px-4 py-2.5 text-sm font-medium border-b-2 transition-colors focus-visible:ring-2 focus-visible:ring-primary-500 ${
                      selected
                        ? 'border-primary-600 text-primary-700 bg-white'
                        : 'border-transparent text-gray-600 hover:text-gray-900 hover:border-gray-300'
                    }`}
                  >
                    <Icon size={16} aria-hidden="true" />
                    {t.label}
                  </button>
                );
              })}
            </div>
          </div>

          <div className="mt-6" id={`panel-${tab}`} role="tabpanel" aria-labelledby={`tab-${tab}`}>
            {tab === 'school' && <SchoolSettingsForms canEdit={canEdit} />}
            {tab === 'operations' && <SchoolSettingsForms canEdit={canEdit} />}
            {tab === 'integrations' && <IntegrationsPanel canEdit={canEdit} />}
            {tab === 'access' && (
              <div className="space-y-4">
                <AccountCard staff={staff} />
                <SessionCard canEdit={canEdit} onSignOut={() => logoutStaff()} />
                <PrivacyCard canEdit={canEdit} />
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
