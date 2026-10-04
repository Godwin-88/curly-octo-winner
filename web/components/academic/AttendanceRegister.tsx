'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { CalendarCheck, Check, Loader2 } from 'lucide-react';

import {
  APIError,
  AttendanceStatus,
  Learner,
  asList,
  api,
} from '@/lib/api';
import { useAuth } from '@/lib/auth';

const STATUSES: { value: AttendanceStatus; label: string; active: string }[] = [
  { value: 'present', label: 'Present', active: 'bg-green-700 text-white' },
  { value: 'absent', label: 'Absent', active: 'bg-red-700 text-white' },
  { value: 'late', label: 'Late', active: 'bg-amber-700 text-white' },
  { value: 'excused', label: 'Excused', active: 'bg-blue-700 text-white' },
];

export interface RegisterSaved {
  saved: number;
  /** What was done about texting the parents of absent learners, if asked. */
  alerts?: { sent: number; skipped: { learner_name: string; reason: string }[] };
  alertError?: string;
}

interface Props {
  date: string;
  onSaved: (summary: RegisterSaved) => void;
}

const classOf = (l: Learner) => [l.grade, l.stream].filter(Boolean).join(' ');

/** Today's date in Kenya, which is the only day parents are texted about. */
const todayInKenya = () =>
  new Intl.DateTimeFormat('en-CA', { timeZone: 'Africa/Nairobi' }).format(new Date());

/**
 * The daily roll call.
 *
 * Design decisions that matter for a teacher standing in front of a class:
 *   - "Mark all present" first, then exceptions. Marking 40 learners one by one
 *     is how registers get abandoned halfway.
 *   - The save button is disabled until every learner has a status, so a
 *     half-marked register can never be recorded as complete.
 *   - One bulk request per save: the server either stores the whole register or
 *     none of it.
 */
export default function AttendanceRegister({ date, onSaved }: Props) {
  const { token } = useAuth();

  const [roster, setRoster] = useState<Learner[]>([]);
  const [klass, setKlass] = useState('');
  const [notify, setNotify] = useState(false);
  const [marks, setMarks] = useState<Record<string, AttendanceStatus>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [everyone, existing] = await Promise.all([
        api.listLearners({}, token),
        api.listAttendanceByDate(date, token),
      ]);

      const safeRoster = asList(everyone);
      setRoster(safeRoster);

      // Prefill from what is already recorded so re-opening a day shows the
      // marks rather than a blank register that would overwrite them.
      const saved: Record<string, AttendanceStatus> = {};
      for (const row of asList(existing)) {
        saved[row.learner_id] = row.status;
      }
      setMarks(saved);
    } catch (err) {
      setError(err instanceof APIError ? err.message : 'Could not load the register.');
    } finally {
      setLoading(false);
    }
  }, [token, date]);

  useEffect(() => {
    load();
  }, [load]);

  // A register is one class's. Asking a teacher to mark the whole school
  // before anything can be saved is how registers do not get saved.
  const classes = useMemo(
    () => Array.from(new Set(roster.map(classOf))).sort((a, b) => a.localeCompare(b, undefined, { numeric: true })),
    [roster]
  );
  const chosen = classes.includes(klass) ? klass : classes[0] ?? '';
  const learners = useMemo(() => roster.filter((l) => classOf(l) === chosen), [roster, chosen]);
  const isToday = date === todayInKenya();

  const mark = (id: string, status: AttendanceStatus) =>
    setMarks((prev) => ({ ...prev, [id]: status }));

  const markAll = (status: AttendanceStatus) =>
    setMarks((prev) => ({ ...prev, ...Object.fromEntries(learners.map((l) => [l.id, status])) }));

  // Clearing removes this class's marks rather than storing an empty-string
  // status: an unmarked learner is "no entry", not a status the API would have
  // to reject.
  const clearAll = () =>
    setMarks((prev) => {
      const next = { ...prev };
      for (const l of learners) delete next[l.id];
      return next;
    });

  const unmarked = learners.filter((l) => !marks[l.id]);
  const complete = learners.length > 0 && unmarked.length === 0;

  const tally = useMemo(() => {
    const counts: Record<AttendanceStatus, number> = {
      present: 0,
      absent: 0,
      late: 0,
      excused: 0,
    };
    for (const l of learners) {
      const status = marks[l.id];
      if (status && status in counts) counts[status] += 1;
    }
    return counts;
  }, [marks, learners]);

  async function save() {
    if (!complete || saving) return;
    setSaving(true);
    setError(null);
    try {
      const result = await api.markAttendanceBulk(
        {
          date,
          marks: learners.map((l) => ({
            learner_id: l.id,
            status: marks[l.id],
          })),
          notify_guardians: notify && isToday,
        },
        token
      );
      onSaved({ saved: result.saved, alerts: result.alerts, alertError: result.alert_error });
    } catch (err) {
      setError(
        err instanceof APIError ? err.message : 'Could not save the register. Please try again.'
      );
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="card p-6 flex items-center gap-2 text-gray-500" aria-busy="true">
        <Loader2 size={16} className="animate-spin" /> Loading the register…
      </div>
    );
  }

  if (error && roster.length === 0) {
    return (
      <div role="alert" className="card p-6 text-red-700">
        {error}
        <button type="button" className="btn-secondary mt-3" onClick={load}>
          Try again
        </button>
      </div>
    );
  }

  if (roster.length === 0) {
    return (
      <div className="card p-8 text-center">
        <CalendarCheck size={32} className="mx-auto mb-3 text-gray-400" aria-hidden="true" />
        <p className="font-medium">No learners on the roll yet</p>
        <p className="text-sm text-gray-500 mt-1">
          Add learners to the school before marking a register.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="card p-4 flex flex-wrap items-center gap-2 justify-between">
        <div className="flex flex-wrap items-center gap-2">
          <label className="text-sm text-gray-600" htmlFor="register-class">
            Class
          </label>
          <select
            id="register-class"
            className="input w-auto text-sm"
            value={chosen}
            onChange={(e) => setKlass(e.target.value)}
          >
            {classes.map((c) => (
              <option key={c} value={c}>
                {c || 'No class'}
              </option>
            ))}
          </select>
          <button type="button" className="btn-secondary text-sm" onClick={() => markAll('present')}>
            Mark all present
          </button>
          <button type="button" className="btn-secondary text-sm" onClick={clearAll}>
            Clear
          </button>
        </div>
        <div className="flex flex-wrap items-center gap-3 text-sm">
          <Tally label="Present" value={tally.present} className="text-green-700" />
          <Tally label="Absent" value={tally.absent} className="text-red-700" />
          <Tally label="Late" value={tally.late} className="text-yellow-700" />
          <Tally label="Excused" value={tally.excused} className="text-blue-700" />
        </div>
      </div>

      {unmarked.length > 0 && (
        <p role="status" className="text-sm text-amber-800 bg-amber-50 p-3 rounded">
          {unmarked.length} learner{unmarked.length === 1 ? '' : 's'} still to mark.
        </p>
      )}

      {error && (
        <p role="alert" className="text-sm text-red-700 bg-red-50 p-3 rounded">
          {error}
        </p>
      )}

      <div className="card overflow-hidden">
        <ul className="divide-y">
          {learners.map((learner) => {
            const current = marks[learner.id];
            return (
              <li key={learner.id} className="p-3 flex flex-wrap items-center gap-3">
                <div className="min-w-0 flex-1">
                  <p className="font-medium text-gray-900 truncate">{learner.full_name}</p>
                  <p className="text-xs text-gray-500">
                    {[learner.grade, learner.stream].filter(Boolean).join(' · ')}
                  </p>
                </div>
                <div className="flex gap-1" role="group" aria-label={`Mark ${learner.full_name}`}>
                  {STATUSES.map((s) => {
                    const active = current === s.value;
                    return (
                      <button
                        key={s.value}
                        type="button"
                        aria-pressed={active}
                        onClick={() => mark(learner.id, s.value)}
                        className={`px-3 py-1.5 text-xs font-medium rounded border transition-colors ${
                          active
                            ? s.active
                            : 'border-gray-300 text-gray-600 hover:bg-gray-100'
                        }`}
                      >
                        {active && <Check size={12} className="inline mr-1" aria-hidden="true" />}
                        {s.label}
                      </button>
                    );
                  })}
                </div>
              </li>
            );
          })}
        </ul>
      </div>

      <div className="flex flex-wrap items-center justify-end gap-3">
        {isToday && (
          <label className="flex items-center gap-2 text-sm text-gray-700 mr-auto">
            <input
              type="checkbox"
              checked={notify}
              onChange={(e) => setNotify(e.target.checked)}
              className="h-4 w-4"
            />
            Text the parents of absent learners
            {notify && tally.absent > 0 && (
              <span className="text-gray-500">
                ({tally.absent} learner{tally.absent === 1 ? '' : 's'}; each parent is texted once a day)
              </span>
            )}
          </label>
        )}
        <p className="text-sm text-gray-500">
          {complete ? 'All learners marked.' : `${unmarked.length} remaining`}
        </p>
        <button type="button" className="btn-primary" onClick={save} disabled={!complete || saving}>
          {saving ? 'Saving register…' : 'Save register'}
        </button>
      </div>
    </div>
  );
}

function Tally({ label, value, className }: { label: string; value: number; className: string }) {
  return (
    <span className="flex items-center gap-1">
      <span className="text-gray-500">{label}:</span>
      <span className={`font-semibold ${className}`}>{value}</span>
    </span>
  );
}
