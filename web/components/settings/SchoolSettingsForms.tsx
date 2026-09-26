'use client';

// Settings → School and Operations: the configuration a principal maintains
// for the whole school (identity and contact, payments wiring, term, attendance
// window, feature switches and document footers).

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  settings as settingsApi,
  type SchoolProfile,
  type SchoolSettings,
  type SettingsBundle,
} from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { Skeleton } from '@/components/ui/Skeleton';

const inputClass =
  'mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:outline-none disabled:bg-gray-50';

function Toggle({
  id,
  label,
  help,
  checked,
  disabled,
  onChange,
}: {
  id: string;
  label: string;
  help: string;
  checked: boolean;
  disabled: boolean;
  onChange: (next: boolean) => void;
}) {
  return (
    <div className="flex items-start gap-3">
      <input
        id={id}
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
        className="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus-visible:ring-2 focus-visible:ring-primary-500"
      />
      <label htmlFor={id} className="text-sm">
        <span className="font-medium text-gray-900">{label}</span>
        <span className="block text-gray-600">{help}</span>
      </label>
    </div>
  );
}

function TextField({
  id,
  label,
  value,
  onChange,
  disabled,
  placeholder,
  help,
  type = 'text',
}: {
  id: string;
  label: string;
  value: string;
  onChange: (next: string) => void;
  disabled: boolean;
  placeholder?: string;
  help?: string;
  type?: string;
}) {
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-gray-700">
        {label}
      </label>
      <input
        id={id}
        type={type}
        value={value}
        disabled={disabled}
        placeholder={placeholder}
        aria-describedby={help ? `${id}-help` : undefined}
        onChange={(e) => onChange(e.target.value)}
        className={inputClass}
      />
      {help && (
        <p id={`${id}-help`} className="mt-1 text-xs text-gray-500">
          {help}
        </p>
      )}
    </div>
  );
}

function SaveRow({
  saving,
  error,
  onSave,
}: {
  saving: boolean;
  error?: string;
  onSave: () => void;
}) {
  const [saved, setSaved] = useState(false);
  return (
    <div className="mt-4 flex flex-wrap items-center gap-3 border-t border-gray-100 pt-4">
      <button
        type="button"
        onClick={() => {
          onSave();
          setSaved(true);
        }}
        disabled={saving}
        className="btn-primary focus-visible:ring-2 focus-visible:ring-primary-500"
      >
        {saving ? 'Saving…' : 'Save changes'}
      </button>
      {error && (
        <p role="alert" className="text-sm text-red-700">
          {error}
        </p>
      )}
      {!error && saved && !saving && (
        <p role="status" className="text-sm text-green-700">
          Saved.
        </p>
      )}
    </div>
  );
}

export function SchoolSettingsForms({ canEdit }: { canEdit: boolean }) {
  const { token, staff, ready } = useAuth();
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ['settings'],
    queryFn: () => settingsApi.get(token),
    // Keyed on the verified session, not the in-memory token: after a hard
    // reload there is no token but the cookie session is valid.
    enabled: ready && !!staff,
  });

  const [profile, setProfile] = useState<SchoolProfile | null>(null);
  const [operations, setOperations] = useState<SchoolSettings | null>(null);

  // Adopt the server response as the form baseline. This adjusts state during
  // render rather than in an effect, so the form paints once with real values
  // instead of flashing empty fields and rendering twice.
  const [baseline, setBaseline] = useState<SettingsBundle | null>(null);
  if (data && data !== baseline) {
    setBaseline(data);
    setProfile(data.profile);
    setOperations(data.settings);
  }

  const save = useMutation({
    mutationFn: (payload: { profile?: Partial<SchoolProfile>; settings?: Partial<SchoolSettings> }) =>
      settingsApi.update(token, payload),
    onSuccess: (result) => {
      setProfile(result.profile);
      setOperations(result.settings);
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
  });

  if (isLoading || !profile || !operations) {
    return (
      <div className="card p-5 space-y-3" aria-hidden="true">
        <Skeleton className="h-4 w-40" />
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-full" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="card p-5 border-red-200 bg-red-50" role="alert">
        <p className="text-sm text-red-800">Could not load school settings: {error.message}</p>
      </div>
    );
  }

  const setProfileField = <K extends keyof SchoolProfile>(key: K, value: SchoolProfile[K]) =>
    setProfile({ ...profile, [key]: value });
  const setOp = <K extends keyof SchoolSettings>(key: K, value: SchoolSettings[K]) =>
    setOperations({ ...operations, [key]: value });

  return (
    <div className="space-y-4">
      {/* --- School identity --- */}
      <section className="card p-5" aria-labelledby="school-heading">
        <h2 id="school-heading" className="text-base font-semibold text-gray-900">
          School profile
        </h2>
        <p className="mt-0.5 text-sm text-gray-600">
          Appears on report cards, receipts and messages sent to parents.
        </p>

        <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
          <TextField
            id="school-name"
            label="School name"
            value={profile.name}
            disabled={!canEdit}
            onChange={(v) => setProfileField('name', v)}
          />
          <TextField
            id="school-slug"
            label="School code (URL)"
            value={profile.slug}
            disabled
            onChange={() => {}}
            help="Fixed when the school was created."
          />
          <TextField
            id="school-phone"
            label="Phone"
            type="tel"
            value={profile.phone ?? ''}
            disabled={!canEdit}
            onChange={(v) => setProfileField('phone', v)}
            placeholder="+254…"
          />
          <TextField
            id="school-email"
            label="Official email"
            type="email"
            value={profile.email ?? ''}
            disabled={!canEdit}
            onChange={(v) => setProfileField('email', v)}
          />
          <TextField
            id="school-county"
            label="County"
            value={profile.county ?? ''}
            disabled={!canEdit}
            onChange={(v) => setProfileField('county', v)}
          />
          <TextField
            id="school-address"
            label="Postal address"
            value={profile.address ?? ''}
            disabled={!canEdit}
            onChange={(v) => setProfileField('address', v)}
          />
          <TextField
            id="school-logo"
            label="Logo URL"
            value={profile.logo_url ?? ''}
            disabled={!canEdit}
            onChange={(v) => setProfileField('logo_url', v)}
            placeholder="https://…/logo.png"
            help="Used on the parent portal and report cards."
          />
        </div>

        {canEdit && (
          <SaveRow
            saving={save.isPending}
            error={save.error?.message}
            onSave={() => save.mutate({ profile })}
          />
        )}
      </section>

      {/* --- Fee collection --- */}
      <section className="card p-5" aria-labelledby="payments-heading">
        <h2 id="payments-heading" className="text-base font-semibold text-gray-900">
          Fee collection
        </h2>
        <p className="mt-0.5 text-sm text-gray-600">
          How parents pay: the paybill, and what each STK push charges against.
        </p>

        <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
          <TextField
            id="mpesa-shortcode"
            label="M-Pesa paybill / shortcode"
            value={profile.mpesa_shortcode ?? ''}
            disabled={!canEdit}
            onChange={(v) => setProfileField('mpesa_shortcode', v)}
            placeholder="174379"
          />
          <div>
            <label htmlFor="mpesa-basis" className="block text-sm font-medium text-gray-700">
              Charge against
            </label>
            <select
              id="mpesa-basis"
              value={profile.mpesa_account_basis}
              disabled={!canEdit}
              onChange={(e) =>
                setProfileField('mpesa_account_basis', e.target.value as SchoolProfile['mpesa_account_basis'])
              }
              className={inputClass}
            >
              <option value="phone">Parent phone number</option>
              <option value="account">Parent account number</option>
              <option value="customer">Parent customer / paybill number</option>
            </select>
            <p className="mt-1 text-xs text-gray-500">
              Sent as <code>AccountReference</code> in the STK push.
            </p>
          </div>
          <div className="sm:col-span-2">
            <TextField
              id="mpesa-callback"
              label="M-Pesa callback URL"
              value={profile.mpesa_callback_url ?? ''}
              disabled={!canEdit}
              onChange={(v) => setProfileField('mpesa_callback_url', v)}
              placeholder="https://your-api.onrender.com/api/v1/webhooks/mpesa/callback"
              help="Must match the URL registered with Safaricom, or payment results never arrive."
            />
          </div>
        </div>

        {canEdit && (
          <SaveRow
            saving={save.isPending}
            error={save.error?.message}
            onSave={() =>
              save.mutate({
                profile: {
                  mpesa_shortcode: profile.mpesa_shortcode,
                  mpesa_account_basis: profile.mpesa_account_basis,
                  mpesa_callback_url: profile.mpesa_callback_url,
                },
              })
            }
          />
        )}
      </section>

      {/* --- Operations --- */}
      <section className="card p-5" aria-labelledby="operations-heading">
        <h2 id="operations-heading" className="text-base font-semibold text-gray-900">
          School operations
        </h2>
        <p className="mt-0.5 text-sm text-gray-600">The defaults staff work with every day.</p>

        <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
          <TextField
            id="current-term"
            label="Current term"
            value={operations.current_term ?? ''}
            disabled={!canEdit}
            onChange={(v) => setOp('current_term', v)}
            placeholder="Term 2"
          />
          <TextField
            id="current-year"
            label="Academic year"
            value={operations.current_academic_year ?? ''}
            disabled={!canEdit}
            onChange={(v) => setOp('current_academic_year', v)}
            placeholder="2026"
          />
          <TextField
            id="attendance-time"
            label="Attendance opens"
            type="time"
            value={operations.attendance_time}
            disabled={!canEdit}
            onChange={(v) => setOp('attendance_time', v)}
            help="Teachers can mark attendance from this time."
          />
          <TextField
            id="attendance-deadline"
            label="Attendance closes"
            type="time"
            value={operations.attendance_deadline}
            disabled={!canEdit}
            onChange={(v) => setOp('attendance_deadline', v)}
            help="After this, marking needs an administrator."
          />
          <div className="sm:col-span-2">
            <TextField
              id="report-card-footer"
              label="Report card footer"
              value={operations.report_card_footer ?? ''}
              disabled={!canEdit}
              onChange={(v) => setOp('report_card_footer', v)}
              placeholder="Printed at the bottom of every report card"
            />
          </div>
          <div className="sm:col-span-2">
            <TextField
              id="receipt-footer"
              label="Receipt / invoice footer"
              value={operations.receipt_footer ?? ''}
              disabled={!canEdit}
              onChange={(v) => setOp('receipt_footer', v)}
              placeholder="e.g. Fees are payable before the end of term"
            />
          </div>
        </div>

        {canEdit && (
          <SaveRow
            saving={save.isPending}
            error={save.error?.message}
            onSave={() =>
              save.mutate({
                settings: {
                  current_term: operations.current_term,
                  current_academic_year: operations.current_academic_year,
                  attendance_time: operations.attendance_time,
                  attendance_deadline: operations.attendance_deadline,
                  report_card_footer: operations.report_card_footer,
                  receipt_footer: operations.receipt_footer,
                },
              })
            }
          />
        )}
      </section>

      {/* --- Feature switches --- */}
      <section className="card p-5" aria-labelledby="features-heading">
        <h2 id="features-heading" className="text-base font-semibold text-gray-900">
          What this school uses
        </h2>
        <p className="mt-0.5 text-sm text-gray-600">
          Switch off what you do not use, so staff are not offered features that cannot work.
        </p>

        <div className="mt-4 space-y-3">
          <Toggle
            id="feature-mpesa"
            label="M-Pesa fee payments"
            help="Parents pay invoices from the parent portal."
            checked={operations.mpesa_enabled}
            disabled={!canEdit}
            onChange={(v) => setOp('mpesa_enabled', v)}
          />
          <Toggle
            id="feature-sms"
            label="SMS (fee reminders, results)"
            help="Needs Africa's Talking credentials under Integrations."
            checked={operations.sms_enabled}
            disabled={!canEdit}
            onChange={(v) => setOp('sms_enabled', v)}
          />
          <Toggle
            id="feature-whatsapp"
            label="WhatsApp messaging"
            help="Needs WhatsApp Cloud API credentials under Integrations."
            checked={operations.whatsapp_enabled}
            disabled={!canEdit}
            onChange={(v) => setOp('whatsapp_enabled', v)}
          />
          <Toggle
            id="feature-consent"
            label="Require parent consent before messaging"
            help="KDPA: collect and store consent before contacting a guardian."
            checked={operations.require_parent_consent}
            disabled={!canEdit}
            onChange={(v) => setOp('require_parent_consent', v)}
          />
          <Toggle
            id="feature-transfer"
            label="Staff approval required for learner transfers"
            help="A learner cannot be moved between classes without approval."
            checked={operations.require_staff_approval_on_transfer}
            disabled={!canEdit}
            onChange={(v) => setOp('require_staff_approval_on_transfer', v)}
          />
        </div>

        {canEdit && (
          <SaveRow
            saving={save.isPending}
            error={save.error?.message}
            onSave={() =>
              save.mutate({
                settings: {
                  mpesa_enabled: operations.mpesa_enabled,
                  sms_enabled: operations.sms_enabled,
                  whatsapp_enabled: operations.whatsapp_enabled,
                  require_parent_consent: operations.require_parent_consent,
                  require_staff_approval_on_transfer: operations.require_staff_approval_on_transfer,
                },
              })
            }
          />
        )}
      </section>
    </div>
  );
}
