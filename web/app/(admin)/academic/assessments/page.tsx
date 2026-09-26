'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { ClipboardList, Plus, Trash2 } from 'lucide-react';
import Link from 'next/link';

import {
  APIError,
  AssessmentSummary,
  asList,
  api,
} from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { EmptyState } from '@/components/ui/EmptyState';
import { ErrorState } from '@/components/ui/ErrorState';
import { Skeleton } from '@/components/ui/Skeleton';
import ObservationModal, { RUBRIC_LEVELS } from '@/components/academic/ObservationModal';

const TABS = [
  { id: 'record', label: 'Observations' },
  { id: 'report-cards', label: 'Report cards' },
] as const;

type Tab = (typeof TABS)[number]['id'];

const LEVEL_TONES: Record<number, string> = {
  1: 'bg-red-100 text-red-800',
  2: 'bg-yellow-100 text-yellow-800',
  3: 'bg-green-100 text-green-800',
  4: 'bg-blue-100 text-blue-800',
};

/**
 * Formative assessment.
 *
 * Observations are the working record a teacher keeps all term; the competency
 * panel is computed from them rather than hard-coded, so it answers the question
 * a head teacher actually asks: where is the class weak?
 */
export default function AssessmentsPage() {
  const { token } = useAuth();

  const [tab, setTab] = useState<Tab>('record');
  const [term, setTerm] = useState(currentTerm());
  const [year, setYear] = useState(new Date().getFullYear());

  const [observations, setObservations] = useState<AssessmentSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const [modalOpen, setModalOpen] = useState(false);
  const [pendingDelete, setPendingDelete] = useState<AssessmentSummary | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      setError(null);
      const rows = await api.listTermObservations({ term, year }, token);
      setObservations(asList(rows));
    } catch (err) {
      setError(err instanceof APIError ? err.message : 'Could not load the observations.');
    } finally {
      setLoading(false);
    }
  }, [token, term, year]);

  useEffect(() => {
    load();
  }, [load]);

  /** Real distribution, aggregated client-side from this term's observations. */
  const distribution = useMemo(() => {
    const counts: Record<number, number> = { 1: 0, 2: 0, 3: 0, 4: 0 };
    for (const o of observations) {
      if (o.rubric_level in counts) counts[o.rubric_level] += 1;
    }
    return counts;
  }, [observations]);

  const total = observations.length;
  const weakest = useMemo(() => {
    if (total === 0) return null;
    const entry = Object.entries(distribution)
      .map(([level, count]) => ({ level: Number(level), count }))
      .sort((a, b) => b.count - a.count)[0];
    return entry && entry.count > 0 ? entry : null;
  }, [distribution, total]);

  async function onDeleteConfirmed() {
    if (!pendingDelete) return;
    setBusy(true);
    try {
      await api.deleteAssessment(pendingDelete.id, token);
      setObservations((prev) => prev.filter((o) => o.id !== pendingDelete.id));
      setNotice('Observation deleted.');
    } catch (err) {
      setNotice(
        err instanceof APIError ? err.message : 'Could not delete the observation.'
      );
    } finally {
      setBusy(false);
      setPendingDelete(null);
    }
  }

  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error && observations.length === 0) {
    return <ErrorState title="Assessments unavailable" message={error} onRetry={load} />;
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-4 mb-6">
        <div>
          <h1 className="text-2xl font-bold">Formative Assessment</h1>
          <p className="text-sm text-gray-500 mt-1">
            Record what you observed; the distribution updates as you go.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <select
            className="input w-auto"
            value={term}
            onChange={(e) => setTerm(Number(e.target.value))}
            aria-label="Term"
          >
            <option value={1}>Term 1</option>
            <option value={2}>Term 2</option>
            <option value={3}>Term 3</option>
          </select>
          <input
            type="number"
            className="input w-28"
            value={year}
            onChange={(e) => setYear(Number(e.target.value))}
            aria-label="Year"
          />
          <button
            type="button"
            className="btn-primary flex items-center gap-2"
            onClick={() => setModalOpen(true)}
          >
            <Plus size={16} /> New Observation
          </button>
        </div>
      </div>

      <div className="flex gap-2 mb-4 border-b" role="tablist" aria-label="Assessment views">
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
          className="mb-4 p-3 rounded bg-blue-50 text-blue-800 text-sm flex justify-between items-center gap-3"
        >
          <span>{notice}</span>
          <button type="button" className="text-blue-700 hover:text-blue-900" onClick={() => setNotice(null)}>
            Dismiss
          </button>
        </div>
      )}

      {tab === 'report-cards' ? (
        <div className="card p-8 text-center">
          <ClipboardList size={32} className="mx-auto mb-3 text-gray-400" aria-hidden="true" />
          <p className="font-medium">Report cards live in their own section</p>
          <p className="text-sm text-gray-500 mt-1 mb-4 max-w-md mx-auto">
            Termly report cards are generated from the scores and observations recorded here.
          </p>
          <Link href="/reports" className="btn-primary">
            Go to report cards
          </Link>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2">
            <div className="card">
              {observations.length === 0 ? (
                <div className="p-4">
                  <EmptyState
                    title="No observations this term"
                    message="Record what a learner did against a sub-strand; the class profile builds itself as you go."
                    action={
                      <button type="button" className="btn-primary" onClick={() => setModalOpen(true)}>
                        <Plus size={16} /> New Observation
                      </button>
                    }
                  />
                </div>
              ) : (
                <ul className="divide-y">
                  {observations.map((o) => (
                    <li key={o.id} className="p-4 flex items-start gap-3">
                      <div className="min-w-0 flex-1">
                        <p className="font-medium text-gray-900">
                          {o.learner_name}
                          <span className="ml-2 text-xs font-normal text-gray-500">
                            {[o.grade, o.stream].filter(Boolean).join(' · ')}
                          </span>
                        </p>
                        <p className="text-sm text-gray-600">
                          {[o.learning_area, o.strand_name, o.sub_strand_name]
                            .filter(Boolean)
                            .join(' › ')}
                        </p>
                        {o.note && <p className="text-sm text-gray-500 mt-1">{o.note}</p>}
                        <p className="text-xs text-gray-400 mt-1">
                          {new Date(o.created_at).toLocaleString()}
                        </p>
                      </div>
                      <span
                        className={`px-2 py-1 rounded text-xs font-medium whitespace-nowrap ${
                          LEVEL_TONES[o.rubric_level] ?? 'bg-gray-100 text-gray-700'
                        }`}
                      >
                        {o.rubric_label}
                      </span>
                      <button
                        type="button"
                        className="p-2 text-gray-500 hover:text-red-600"
                        aria-label={`Delete observation for ${o.learner_name}`}
                        onClick={() => setPendingDelete(o)}
                      >
                        <Trash2 size={16} />
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>

          <div className="space-y-4">
            <div className="card p-4">
              <h2 className="font-semibold mb-3">Class distribution</h2>
              {total === 0 ? (
                <p className="text-sm text-gray-500">
                  Record an observation to see the class profile.
                </p>
              ) : (
                <div className="space-y-3">
                  {RUBRIC_LEVELS.map((r) => {
                    const count = distribution[r.value] ?? 0;
                    const pct = total > 0 ? Math.round((count / total) * 100) : 0;
                    return (
                      <div key={r.value}>
                        <div className="flex justify-between text-sm">
                          <span className="text-gray-600">{r.label}</span>
                          <span className="font-medium text-gray-900">{count}</span>
                        </div>
                        <div className="h-2 bg-gray-100 rounded mt-1 overflow-hidden">
                          <div
                            className="h-full bg-blue-500"
                            style={{ width: `${pct}%` }}
                            role="presentation"
                          />
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>

            {weakest && (
              <div className="card p-4">
                <h2 className="font-semibold mb-2">Most common level</h2>
                <p className="text-sm text-gray-600">
                  {weakest.count} of {total} observations sit at{' '}
                  <strong>{RUBRIC_LEVELS.find((r) => r.value === weakest.level)?.label}</strong>.
                </p>
              </div>
            )}
          </div>
        </div>
      )}

      {modalOpen && (
        <ObservationModal
          onClose={() => setModalOpen(false)}
          onSaved={async (message) => {
            setModalOpen(false);
            setNotice(message);
            await load();
          }}
        />
      )}

      {pendingDelete && (
        <ConfirmDialog
          title="Delete this observation?"
          message={`The observation for ${pendingDelete.learner_name} on ${pendingDelete.sub_strand_name} will be removed permanently.`}
          busy={busy}
          onCancel={() => setPendingDelete(null)}
          onConfirm={onDeleteConfirmed}
        />
      )}
    </div>
  );
}

function currentTerm(): number {
  const month = new Date().getMonth() + 1;
  if (month <= 4) return 1;
  if (month <= 8) return 2;
  return 3;
}
