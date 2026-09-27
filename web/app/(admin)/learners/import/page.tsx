'use client';

// Import a learner roster from a CSV.
//
// The flow follows the data model rather than fighting it: a file is staged
// whole, the incomplete rows stay editable, and each row becomes a real
// learner on its own once it has a student name, a number and a grade. There is
// no "all or nothing" commit button, because losing 298 good rows to one typo is
// the thing this is meant to avoid.

import { useState } from 'react';
import Link from 'next/link';
import { ArrowLeft, Upload } from 'lucide-react';

import { api, LearnerImportBatch, LearnerImportRowStatus } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { useApiQuery, useApiMutation } from '@/lib/query';
import { SkeletonTable } from '@/components/ui/Skeleton';
import { ErrorState } from '@/components/ui/ErrorState';
import { EmptyState } from '@/components/ui/EmptyState';
import LearnerImportUploader from '@/components/learners/LearnerImportUploader';
import LearnerImportRows from '@/components/learners/LearnerImportRows';

const STATUS_FILTERS: { value: LearnerImportRowStatus | ''; label: string }[] = [
  { value: '', label: 'All rows' },
  { value: 'draft', label: 'Not imported' },
  { value: 'imported', label: 'Imported' },
];

export default function LearnerImportPage() {
  const { token, staff } = useAuth();
  // The batch being worked on. Null means "show the picker".
  const [batchId, setBatchId] = useState<string | null>(null);
  const [status, setStatus] = useState<LearnerImportRowStatus | ''>('');
  const [bulkMessage, setBulkMessage] = useState<string | null>(null);

  const batchesQuery = useApiQuery(
    ['learner-import-batches'],
    () => api.listLearnerImportBatches(token),
    { enabled: !!staff }
  );

  const batches = batchesQuery.data?.items ?? [];
  // Open the most recent upload by default, so a returning user resumes work
  // rather than landing on a picker and having to remember which file was open.
  const activeBatchId = batchId ?? batches[0]?.id ?? null;
  const batch: LearnerImportBatch | undefined = batches.find((b) => b.id === activeBatchId);

  const rowsQuery = useApiQuery(
    ['learner-import-rows', activeBatchId, status],
    () => api.listLearnerImportRows(activeBatchId as string, { status: status || undefined }, token),
    { enabled: !!staff && !!activeBatchId }
  );

  const rows = rowsQuery.data?.items ?? [];
  const refresh = () => {
    void rowsQuery.refetch();
    void batchesQuery.refetch();
  };

  const promoteMutation = useApiMutation(
    (id: string) => api.promoteLearnerImportRow(id, token),
    [['learner-import-rows'], ['learner-import-batches']]
  );

  const readyRows = rows.filter((r) => r.is_sufficient && r.status === 'draft');
  const allPromoted = readyRows.length > 0 && readyRows.every((r) => r.status === 'imported');

  /**
   * Imports every ready row, one at a time, and reports what happened.
   *
   * Deliberately sequential and fault-tolerant: a roster of 300 rows will hit
   * at least one clash with a learner number already in the school, and a bulk
   * endpoint that stopped at the first failure would leave the user unable to
   * tell how far it got. Sequential also keeps the batch's ready/imported
   * counters meaningful as it goes.
   */
  const importAllReady = async () => {
    const targets = rows.filter((r) => r.is_sufficient && r.status === 'draft');
    let done = 0;
    const failures: string[] = [];
    for (const row of targets) {
      try {
        await promoteMutation.mutateAsync(row.id);
        done += 1;
      } catch (err) {
        failures.push(`Line ${row.row_number}: ${err instanceof Error ? err.message : 'failed'}`);
      }
    }
    refresh();
    if (failures.length > 0) {
      setBulkMessage(
        `Imported ${done} of ${targets.length}. ${failures.length} could not be imported — ` +
          failures.slice(0, 3).join('; ') +
          (failures.length > 3 ? `; and ${failures.length - 3} more` : '')
      );
    } else {
      setBulkMessage(`Imported ${done} learner${done === 1 ? '' : 's'}.`);
    }
  };

  return (
    <div className="space-y-6">
      <header>
        <Link
          href="/learners"
          className="inline-flex items-center gap-1 text-sm text-gray-600 hover:text-gray-900 mb-2"
        >
          <ArrowLeft size={14} aria-hidden="true" /> Learners
        </Link>
        <h1 className="text-2xl font-bold text-gray-900">Import learners from a spreadsheet</h1>
        <p className="text-gray-600 mt-1 max-w-3xl">
          Upload a roster and fix it here. Rows are saved as they arrive, so an
          incomplete spreadsheet is not a failed import — fill in the blanks as
          you go and import each row when it is ready.
        </p>
      </header>

      <LearnerImportUploader
        token={token}
        onStaged={(b) => {
          setBulkMessage(null);
          setStatus('');
          // Switching the batch is what fetches the new rows: the query key
          // changes, so React Query refetches on its own. Calling refetch()
          // here would fire against the *previous* batch and race the one the
          // user is about to see.
          setBatchId(b.id);
          void batchesQuery.refetch();
        }}
      />


      {batchesQuery.error && (
        <ErrorState
          title="Could not load your imports"
          message={String(batchesQuery.error)}
          onRetry={() => void batchesQuery.refetch()}
        />
      )}

      {!batchesQuery.isLoading && batches.length > 0 && (
        <section className="card p-4" aria-labelledby="batches-heading">
          <h2 id="batches-heading" className="text-sm font-semibold text-gray-900 mb-3">
            Previous uploads
          </h2>
          <ul className="flex flex-wrap gap-2">
            {batches.map((b) => {
              const active = b.id === activeBatchId;
              return (
                <li key={b.id}>
                  <button
                    type="button"
                    onClick={() => {
                      setBatchId(b.id);
                      setBulkMessage(null);
                    }}
                    aria-current={active ? 'true' : undefined}
                    className={`text-left px-3 py-2 rounded-md border text-sm ${
                      active
                        ? 'border-blue-500 bg-blue-50 ring-1 ring-blue-500'
                        : 'border-gray-200 hover:bg-gray-50'
                    }`}
                  >
                    <span className="block font-medium text-gray-900">
                      {b.filename || 'Untitled upload'}
                    </span>
                    <span className="block text-xs text-gray-600">
                      {b.total_rows} rows · {b.ready_rows} ready · {b.imported_rows} imported
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        </section>
      )}

      {batch && (
        <section className="space-y-3" aria-labelledby="rows-heading">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <h2 id="rows-heading" className="text-lg font-semibold text-gray-900">
              {batch.filename || 'Staged rows'}
            </h2>
            <div className="flex flex-wrap items-center gap-2">
              <label className="sr-only" htmlFor="status-filter">
                Filter by status
              </label>
              <select
                id="status-filter"
                className="input text-sm py-1.5 w-auto"
                value={status}
                onChange={(e) => setStatus(e.target.value as LearnerImportRowStatus | '')}
              >
                {STATUS_FILTERS.map((f) => (
                  <option key={f.value} value={f.value}>
                    {f.label}
                  </option>
                ))}
              </select>
              {readyRows.length > 0 && (
                <button
                  type="button"
                  className="btn-primary text-sm"
                  onClick={() => void importAllReady()}
                  disabled={promoteMutation.isPending || allPromoted}
                >
                  {promoteMutation.isPending ? (
                    <>
                      <Upload size={14} className="animate-pulse" aria-hidden="true" /> Importing…
                    </>
                  ) : (
                    <>Import {readyRows.length} ready row{readyRows.length === 1 ? '' : 's'}</>
                  )}
                </button>
              )}
            </div>
          </div>

          {bulkMessage && (
            <p
              role="status"
              className="text-sm text-gray-700 bg-blue-50 border border-blue-200 rounded px-3 py-2"
            >
              {bulkMessage}
            </p>
          )}

          <p className="text-sm text-gray-600">
            {batch.total_rows} row{batch.total_rows === 1 ? '' : 's'} staged ·{' '}
            <span className="font-medium text-amber-700">{batch.ready_rows} ready to import</span> ·{' '}
            {batch.imported_rows} imported. Cells save when you click away; a row can be
            imported once it has a student name, student number and grade.
          </p>

          {rowsQuery.isLoading ? (
            <SkeletonTable rows={6} />
          ) : rowsQuery.error ? (
            <ErrorState
              title="Could not load these rows"
              message={String(rowsQuery.error)}
              onRetry={() => void rowsQuery.refetch()}
            />
          ) : rows.length === 0 ? (
            <EmptyState
              title="No rows here"
              message={
                status
                  ? 'No rows in this file have that status.'
                  : 'Upload a spreadsheet above to stage your roster.'
              }
            />
          ) : (
            <LearnerImportRows token={token} rows={rows} onChanged={refresh} />
          )}
        </section>
      )}
    </div>
  );
}

