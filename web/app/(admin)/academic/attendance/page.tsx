'use client';

import { useCallback, useEffect, useState } from 'react';
import { AlertTriangle, CalendarDays } from 'lucide-react';

import {
  APIError,
  AttendanceSummaryCounts,
  ChronicAbsentee,
  asList,
  api,
} from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { StatCard } from '@/components/ui/StatCard';
import { Skeleton } from '@/components/ui/Skeleton';
import AttendanceRegister from '@/components/academic/AttendanceRegister';

const TABS = [
  { id: 'daily', label: 'Daily' },
  { id: 'chronic', label: 'Chronic absence' },
] as const;

type Tab = (typeof TABS)[number]['id'];

/**
 * Attendance management.
 *
 * The register is the point of this screen: pick a date, mark the class, save.
 * The stats beside it are derived from what has actually been recorded, and the
 * chronic-absence tab ranks the learners who need a conversation.
 */
export default function AttendancePage() {
  const { token } = useAuth();

  // Default to today in the browser's own timezone — a teacher in Nairobi
  // marking the register at 7am should not be handed yesterday's date because
  // the server is on UTC.
  const [date, setDate] = useState(() => new Intl.DateTimeFormat('en-CA', { timeZone: 'Africa/Nairobi' }).format(new Date()));
  const [tab, setTab] = useState<Tab>('daily');

  const [summary, setSummary] = useState<AttendanceSummaryCounts | null>(null);
  const [chronic, setChronic] = useState<ChronicAbsentee[]>([]);
  const [loadingStats, setLoadingStats] = useState(true);
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const loadStats = useCallback(async () => {
    setLoadingStats(true);
    try {
      const [counts, chronicRows] = await Promise.all([
        api.getAttendanceSummary(date, token),
        api.listChronicAbsenteeism({ threshold: 75 }, token),
      ]);
      setSummary(counts);
      setChronic(asList(chronicRows));
      setError(null);
    } catch (err) {
      setError(err instanceof APIError ? err.message : 'Could not load attendance figures.');
    } finally {
      setLoadingStats(false);
    }
  }, [token, date]);

  useEffect(() => {
    loadStats();
  }, [loadStats]);

  const rate =
    summary && summary.marked > 0
      ? Math.round(((summary.present + summary.late) / summary.marked) * 100)
      : null;

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-4 mb-6">
        <div>
          <h1 className="text-2xl font-bold">Attendance Management</h1>
          <p className="text-sm text-gray-500 mt-1">
            Mark the class, then check who needs following up.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <label className="text-sm text-gray-600" htmlFor="attendance-date">
            Date
          </label>
          <input
            id="attendance-date"
            type="date"
            className="input"
            value={date}
            max={new Intl.DateTimeFormat('en-CA', { timeZone: 'Africa/Nairobi' }).format(new Date())}
            onChange={(e) => setDate(e.target.value)}
          />
        </div>
      </div>

      <div className="flex gap-2 mb-4 border-b" role="tablist" aria-label="Attendance views">
        {TABS.map((t) => (
          <button
            key={t.id}
            role="tab"
            aria-selected={tab === t.id}
            onClick={() => setTab(t.id)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              tab === t.id
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {notice && (
        <div
          role="status"
          className="mb-4 p-3 rounded bg-green-50 text-green-800 text-sm flex justify-between items-center gap-3"
        >
          <span>{notice}</span>
          <button type="button" className="text-green-700 hover:text-green-900" onClick={() => setNotice(null)}>
            Dismiss
          </button>
        </div>
      )}

      {tab === 'daily' ? (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2">
            <AttendanceRegister
              date={date}
              onSaved={async ({ saved, alerts, alertError }) => {
                // Say what happened to the texts as plainly as what happened to
                // the register: who was told, who was not and why.
                const parts = [`Register saved — ${saved} learner${saved === 1 ? '' : 's'} recorded.`];
                if (alertError) parts.push(alertError);
                if (alerts) {
                  parts.push(
                    alerts.sent > 0
                      ? `A text is on its way to the parents of ${alerts.sent} absent learner${alerts.sent === 1 ? '' : 's'}; see Communications for delivery.`
                      : 'No new texts were sent.'
                  );
                  for (const s of alerts.skipped) parts.push(`${s.learner_name} — not texted: ${s.reason}`);
                }
                setNotice(parts.join(' '));
                await loadStats();
              }}
            />
          </div>

          <div className="space-y-4">
            <h2 className="font-semibold text-sm text-gray-500 uppercase tracking-wide">
              {formatDay(date)}
            </h2>
            {loadingStats ? (
              <div className="space-y-3" aria-busy="true">
                <Skeleton className="h-20 w-full" />
                <Skeleton className="h-20 w-full" />
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-3">
                <StatCard label="Present" value={summary?.present ?? 0} icon={CalendarDays} tone="green" />
                <StatCard label="Absent" value={summary?.absent ?? 0} icon={AlertTriangle} tone="red" />
                <StatCard label="Late" value={summary?.late ?? 0} icon={AlertTriangle} tone="yellow" />
                <StatCard label="Excused" value={summary?.excused ?? 0} icon={CalendarDays} tone="blue" />
              </div>
            )}

            <div className="card p-4">
              <div className="flex justify-between items-center">
                <span className="text-sm text-gray-500">Attendance rate</span>
                <span className="text-lg font-bold text-gray-900">
                  {rate === null ? '—' : `${rate}%`}
                </span>
              </div>
              <p className="text-xs text-gray-500 mt-1">
                {summary?.marked
                  ? `${summary.marked} learner${summary.marked === 1 ? '' : 's'} marked so far.`
                  : 'Nothing marked for this date yet.'}
              </p>
            </div>

            {error && (
              <p role="alert" className="text-sm text-red-700 bg-red-50 p-3 rounded">
                {error}
              </p>
            )}
          </div>
        </div>
      ) : (
        <ChronicList rows={chronic} loading={loadingStats} />
      )}
    </div>
  );
}

function ChronicList({ rows, loading }: { rows: ChronicAbsentee[]; loading: boolean }) {
  if (loading) {
    return <Skeleton className="h-48 w-full" />;
  }
  if (rows.length === 0) {
    return (
      <div className="card p-8 text-center">
        <p className="font-medium">No chronic absentees</p>
        <p className="text-sm text-gray-500 mt-1">
          Learners below 75% attendance for the current term will appear here.
        </p>
      </div>
    );
  }
  return (
    <div className="card overflow-x-auto">
      <table className="w-full text-sm">
        <thead className="bg-gray-50 text-left">
          <tr>
            <th className="px-4 py-3 font-medium text-gray-600">Learner</th>
            <th className="px-4 py-3 font-medium text-gray-600">Class</th>
            <th className="px-4 py-3 font-medium text-gray-600">Days absent</th>
            <th className="px-4 py-3 font-medium text-gray-600">Attendance</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {rows.map((row) => (
            <tr key={row.learner_id}>
              <td className="px-4 py-3 font-medium text-gray-900">{row.learner_name}</td>
              <td className="px-4 py-3 text-gray-600">{[row.grade, row.stream].filter(Boolean).join(' · ')}</td>
              <td className="px-4 py-3 text-gray-600">
                {row.absent_days} of {row.total_days}
              </td>
              <td className="px-4 py-3">
                <span
                  className={`px-2 py-1 rounded text-xs font-medium ${
                    row.attendance_rate < 50
                      ? 'bg-red-100 text-red-800'
                      : 'bg-yellow-100 text-yellow-800'
                  }`}
                >
                  {row.attendance_rate}%
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

/** "Friday, 26 September 2026" — or the raw string if it cannot be parsed. */
function formatDay(date: string): string {
  const parsed = new Date(`${date}T00:00:00`);
  if (Number.isNaN(parsed.getTime())) return date;
  return parsed.toLocaleDateString('en-GB', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });
}
