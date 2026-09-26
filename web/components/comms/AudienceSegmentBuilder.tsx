'use client';

import { useEffect, useMemo, useState } from 'react';

import { api, ContactSummary, GuardianDirectoryEntry, ReachEstimate } from '@/lib/api';
import { useAuth } from '@/lib/auth';

interface Props {
  audienceType: string;
  setAudienceType: (type: string) => void;
  audienceFilter: Record<string, unknown>;
  setAudienceFilter: (filter: Record<string, unknown>) => void;
  onEstimate: () => void;
  estimate: ReachEstimate | null;
}

const AUDIENCE_TYPES = [
  { value: 'contacts', label: 'Saved contacts' },
  { value: 'all_parents', label: 'All Parents' },
  { value: 'grade', label: 'By Grade' },
  { value: 'stream', label: 'By Grade & Stream' },
  { value: 'transport', label: 'Transport Enrolled' },
  { value: 'fee_defaulters', label: 'Fee Defaulters' },
  { value: 'custom', label: 'Custom Selection' },
];

const GRADES = ['Grade 4', 'Grade 5', 'Grade 6'];
const STREAMS = ['North', 'South'];

/**
 * Searchable guardian multi-select for the "Custom Selection" audience.
 *
 * Staff pick guardians by name; the UUIDs are managed by this component. The
 * previous implementation asked users to paste comma-separated UUIDs, which
 * produced cryptic "invalid UUID length" errors from the API on any typo.
 */
function GuardianPicker({
  selected,
  onChange,
}: {
  selected: string[];
  onChange: (ids: string[]) => void;
}) {
  const { token } = useAuth();
  const [search, setSearch] = useState('');
  const [guardians, setGuardians] = useState<GuardianDirectoryEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    setFailed(false);

    // Debounce so typing a name does not fire a request per keystroke.
    const t = setTimeout(() => {
      api
        .listGuardians(search, token)
        .then((rows) => {
          if (alive) setGuardians(rows);
        })
        .catch(() => {
          if (alive) setFailed(true);
        })
        .finally(() => {
          if (alive) setLoading(false);
        });
    }, 250);

    return () => {
      alive = false;
      clearTimeout(t);
    };
  }, [search, token]);

  const selectedRows = useMemo(
    () => guardians.filter((g) => selected.includes(g.id)),
    [guardians, selected]
  );

  const toggle = (id: string) => {
    onChange(selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id]);
  };

  return (
    <div className="space-y-3">
      <div>
        <label className="label" htmlFor="guardian-search">
          Search guardians
        </label>
        <input
          id="guardian-search"
          className="input"
          value={search}
          placeholder="Search by name or phone…"
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {selected.length > 0 && (
        <div>
          <p className="text-sm text-gray-600 mb-2">
            {selected.length} guardian{selected.length === 1 ? '' : 's'} selected
          </p>
          <ul className="flex flex-wrap gap-2" aria-label="Selected guardians">
            {selectedRows.map((g) => (
              <li key={g.id}>
                <button
                  type="button"
                  className="text-xs font-medium text-blue-700 bg-blue-50 rounded-full px-2 py-0.5"
                  onClick={() => toggle(g.id)}
                  aria-label={`Remove ${g.full_name}`}
                >
                  {g.full_name} ×
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      {failed && (
        <p className="text-sm text-red-600" role="alert">
          Could not load guardians. Check your connection and try again.
        </p>
      )}

      <fieldset className="border border-gray-200 rounded-md p-3 max-h-64 overflow-y-auto">
        <legend className="sr-only">Guardians</legend>
        {loading && <p className="text-sm text-gray-500">Loading guardians…</p>}
        {!loading && !failed && guardians.length === 0 && (
          <p className="text-sm text-gray-500">No guardians match “{search}”.</p>
        )}
        {guardians.map((g) => (
          <label
            key={g.id}
            className="flex items-center gap-3 py-1.5 text-sm cursor-pointer"
          >
            <input
              type="checkbox"
              checked={selected.includes(g.id)}
              onChange={() => toggle(g.id)}
            />
            <span className="flex-1">
              {g.full_name}
              <span className="text-gray-500"> · {g.phone || 'no phone'}</span>
            </span>
            {g.is_sms_opted_out && (
              <span
                className="text-xs font-medium text-amber-700 bg-amber-50 rounded-full px-2 py-0.5"
                title="This guardian has opted out of SMS"
              >
                opted out
              </span>
            )}
          </label>
        ))}
      </fieldset>
    </div>
  );
}

/**
 * Filter for the "Saved contacts" audience.
 *
 * Reads the tag breakdown from the contact book so the sender can see what is
 * actually available ("12 contacts, 2 segments") instead of guessing, and
 * links straight to the contact screen when the list needs work.
 */
function SavedContactsFilter({
  tag,
  onChange,
}: {
  tag: string;
  onChange: (tag: string) => void;
}) {
  const { token } = useAuth();
  const [summary, setSummary] = useState<ContactSummary | null>(null);

  useEffect(() => {
    let alive = true;
    api
      .getContactSummary(token)
      .then((s) => {
        if (alive) setSummary(s);
      })
      .catch(() => {
        if (alive) setSummary(null);
      });
    return () => {
      alive = false;
    };
  }, [token]);

  const tags = summary ? Object.entries(summary.by_tag) : [];

  return (
    <div className="border border-indigo-200 bg-indigo-50 rounded-md p-4">
      <p className="text-sm text-indigo-900">
        This audience is the contact book under{' '}
        <a href="/communications/contacts" className="underline font-medium">
          Communications → Contacts
        </a>
        . Archived and opted-out contacts are never messaged.
      </p>

      {summary && (
        <p className="text-sm text-indigo-800 mt-2">
          {summary.active} contact{summary.active === 1 ? '' : 's'} available
          {summary.opted_out > 0 && ` (${summary.opted_out} opted out)`}.
        </p>
      )}

      {tags.length > 0 && (
        <fieldset className="mt-3">
          <legend className="text-xs font-medium text-indigo-900 uppercase tracking-wide">
            Limit to a tag
          </legend>
          <div className="flex flex-wrap gap-2 mt-2">
            {tags.map(([t, n]) => (
              <label
                key={t}
                className="flex items-center gap-1.5 text-sm bg-white rounded-full px-3 py-1 border border-indigo-200 cursor-pointer"
              >
                <input
                  type="radio"
                  name="contacts-tag"
                  checked={tag === t}
                  onChange={() => onChange(t)}
                />
                {t} <span className="text-gray-500">({n})</span>
              </label>
            ))}
          </div>
        </fieldset>
      )}

      {summary && summary.active === 0 && (
        <p className="text-sm text-amber-800 mt-2">
          Your contact book is empty. Add contacts first, then come back to send.
        </p>
      )}
    </div>
  );
}

export default function AudienceSegmentBuilder({
  audienceType,
  setAudienceType,
  audienceFilter,
  setAudienceFilter,
  onEstimate,
  estimate,
}: Props) {
  const selectedIds = (audienceFilter.guardian_ids as string[] | undefined) ?? [];

  return (
    <div className="card p-6 max-w-2xl">
      <h2 className="text-lg font-semibold mb-4">Select Audience</h2>

      <div className="space-y-4">
        <div>
          <label className="label">Audience Type</label>
          <select
            className="input"
            value={audienceType}
            onChange={(e) => {
              setAudienceType(e.target.value);
              setAudienceFilter({});
            }}
          >
            {AUDIENCE_TYPES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </div>

        {audienceType === 'contacts' && (
          <SavedContactsFilter
            tag={(audienceFilter.tag as string) || ''}
            onChange={(t) => setAudienceFilter({ tag: t })}
          />
        )}

        {audienceType === 'grade' && (
          <div>
            <label className="label">Grade</label>
            <select
              className="input"
              value={(audienceFilter.grade as string) || ''}
              onChange={(e) => setAudienceFilter({ ...audienceFilter, grade: e.target.value })}
            >
              <option value="">Select grade</option>
              {GRADES.map((g) => (
                <option key={g} value={g}>
                  {g}
                </option>
              ))}
            </select>
          </div>
        )}

        {audienceType === 'stream' && (
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label">Grade</label>
              <select
                className="input"
                value={(audienceFilter.grade as string) || ''}
                onChange={(e) => setAudienceFilter({ ...audienceFilter, grade: e.target.value })}
              >
                <option value="">Select grade</option>
                {GRADES.map((g) => (
                  <option key={g} value={g}>
                    {g}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">Stream</label>
              <select
                className="input"
                value={(audienceFilter.stream as string) || ''}
                onChange={(e) => setAudienceFilter({ ...audienceFilter, stream: e.target.value })}
              >
                <option value="">Select stream</option>
                {STREAMS.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </select>
            </div>
          </div>
        )}

        {audienceType === 'custom' && (
          <GuardianPicker
            selected={selectedIds}
            onChange={(ids) => setAudienceFilter({ ...audienceFilter, guardian_ids: ids })}
          />
        )}

        <button className="btn-secondary" onClick={onEstimate}>
          Estimate Reach
        </button>

        {estimate && (
          <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
            <div className="flex justify-between">
              <span className="text-sm text-gray-600">Recipients</span>
              <span className="font-semibold">{estimate.recipient_count}</span>
            </div>
            <div className="flex justify-between mt-2">
              <span className="text-sm text-gray-600">Estimated Cost</span>
              <span className="font-semibold">KES {estimate.estimated_kes.toFixed(2)}</span>
            </div>
            <div className="flex justify-between mt-2">
              <span className="text-sm text-gray-600">SMS Units</span>
              <span className="font-semibold">{estimate.sms_units}</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}