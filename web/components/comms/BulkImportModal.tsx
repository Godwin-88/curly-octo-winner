'use client';

import { useEffect, useState } from 'react';
import { AlertTriangle, CheckCircle2, X } from 'lucide-react';

import {
  api,
  ContactImportPreview,
  ContactImportResult,
} from '@/lib/api';
import { formatPhone } from '@/lib/phone';
import { useAuth } from '@/lib/auth';

interface Props {
  onClose: () => void;
  onImported: (result: ContactImportResult) => void;
}

const SAMPLE = `Name,Phone,Class,Tags
Jane Doe,0712345678,Grade 4 North,"grade 4, boarding"
John Roe,+254722334455,Grade 5 North,grade 5`;

type Stage = 'paste' | 'preview' | 'done';

/**
 * Bulk import for the contact book.
 *
 * Flow: paste -> preview (server-side dry run, nothing saved) -> import.
 * The preview exists because a school pasting 300 rows must see "12 ready,
 * 2 need fixing" and the exact line numbers BEFORE anything is written, and
 * must never lose the 298 good rows to one typo.
 */
export default function BulkImportModal({ onClose, onImported }: Props) {
  const { token } = useAuth();

  const [stage, setStage] = useState<Stage>('paste');
  const [csv, setCsv] = useState('');
  const [defaultRelationship, setDefaultRelationship] = useState('parent');
  const [updateExisting, setUpdateExisting] = useState(true);
  const [preview, setPreview] = useState<ContactImportPreview | null>(null);
  const [result, setResult] = useState<ContactImportResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !busy) onClose();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose, busy]);

  const runPreview = async () => {
    if (!csv.trim()) {
      setError('Paste your rows first.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const p = await api.previewContactImport(
        { csv, default_relationship: defaultRelationship },
        token
      );
      setPreview(p);
      setStage('preview');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not read those rows');
    } finally {
      setBusy(false);
    }
  };

  const runImport = async () => {
    setBusy(true);
    setError(null);
    try {
      const r = await api.runContactImport(
        { csv, default_relationship: defaultRelationship, update_existing: updateExisting },
        token
      );
      setResult(r);
      setStage('done');
      onImported(r);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'The import failed');
    } finally {
      setBusy(false);
    }
  };

  // Rows that the import will actually write: new contacts, plus duplicates
  // when the user chose to merge them. The button must not promise fewer rows
  // than the request will actually send.
  const willWrite = preview ? preview.valid + (updateExisting ? preview.duplicates : 0) : 0;

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/50 p-4 overflow-y-auto"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget && !busy) onClose();
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="bulk-import-title"
        className="bg-white rounded-lg shadow-xl w-full max-w-3xl my-8"
      >
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 id="bulk-import-title" className="text-lg font-semibold">
            Import contacts in bulk
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded-md text-gray-500 hover:bg-gray-100"
            aria-label="Close"
            disabled={busy}
          >
            <X size={18} />
          </button>
        </div>

        {error && (
          <p role="alert" className="mx-6 mt-4 bg-red-50 border border-red-200 text-red-700 rounded-md p-3 text-sm">
            {error}
          </p>
        )}

        {stage === 'paste' && (
          <div className="px-6 py-5 space-y-4">
            <p className="text-sm text-gray-600">
              Copy rows straight from Excel or Google Sheets and paste them below.
              The first row may be a header — column names like <em>Name</em>,{' '}
              <em>Phone</em>, <em>Class</em> and <em>Tags</em> are recognised.
            </p>

            <div>
              <label className="label" htmlFor="bulk-csv">
                Rows to import
              </label>
              <textarea
                id="bulk-csv"
                className="input font-mono text-sm"
                rows={10}
                placeholder={SAMPLE}
                value={csv}
                onChange={(e) => setCsv(e.target.value)}
                aria-describedby="bulk-csv-help"
              />
              <div id="bulk-csv-help" className="flex flex-wrap items-center justify-between gap-2 mt-1">
                <p className="text-sm text-gray-500">
                  Phone numbers may be 0712345678 or +254712345678.
                </p>
                <button
                  type="button"
                  className="text-sm text-blue-600 hover:underline"
                  onClick={() => setCsv(SAMPLE)}
                >
                  Use a sample
                </button>
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="label" htmlFor="bulk-relationship">
                  Default relationship
                </label>
                <select
                  id="bulk-relationship"
                  className="input"
                  value={defaultRelationship}
                  onChange={(e) => setDefaultRelationship(e.target.value)}
                >
                  <option value="parent">Parent / guardian</option>
                  <option value="sponsor">Sponsor</option>
                  <option value="staff">Staff</option>
                  <option value="prospective">Prospective family</option>
                  <option value="alumni">Alumni</option>
                  <option value="other">Other</option>
                </select>
                <p className="text-sm text-gray-500 mt-1">
                  Used when a row does not name one.
                </p>
              </div>

              <div className="flex items-end">
                <label className="flex items-start gap-2 text-sm pb-2">
                  <input
                    type="checkbox"
                    className="mt-0.5"
                    checked={updateExisting}
                    onChange={(e) => setUpdateExisting(e.target.checked)}
                  />
                  <span>
                    Update contacts that already exist
                    <span className="block text-xs text-gray-500">
                      Matches on phone number. Tags are merged, never replaced.
                    </span>
                  </span>
                </label>
              </div>
            </div>

            <div className="flex justify-end gap-3 pt-2 border-t">
              <button type="button" className="btn-secondary" onClick={onClose} disabled={busy}>
                Cancel
              </button>
              <button type="button" className="btn-primary" onClick={runPreview} disabled={busy}>
                {busy ? 'Checking…' : 'Check rows'}
              </button>
            </div>
          </div>
        )}


        {stage === 'preview' && preview && (
          <div className="px-6 py-5 space-y-4">
            <div className="grid grid-cols-3 gap-3 text-center">
              <div className="rounded-md border border-green-200 bg-green-50 p-3">
                <p className="text-2xl font-semibold text-green-700">{preview.valid}</p>
                <p className="text-xs text-green-700">new contacts</p>
              </div>
              <div className="rounded-md border border-amber-200 bg-amber-50 p-3">
                <p className="text-2xl font-semibold text-amber-700">{preview.duplicates}</p>
                <p className="text-xs text-amber-700">
                  already saved{updateExisting ? ', will merge' : ', will skip'}
                </p>
              </div>
              <div className="rounded-md border border-red-200 bg-red-50 p-3">
                <p className="text-2xl font-semibold text-red-700">{preview.invalid}</p>
                <p className="text-xs text-red-700">need fixing</p>
              </div>
            </div>

            <div className="max-h-64 overflow-y-auto border rounded-md">
              <table className="w-full text-sm">
                <caption className="sr-only">Import preview by row</caption>
                <thead className="bg-gray-50 text-left sticky top-0">
                  <tr>
                    <th scope="col" className="px-3 py-2 font-medium">Line</th>
                    <th scope="col" className="px-3 py-2 font-medium">Name</th>
                    <th scope="col" className="px-3 py-2 font-medium">Phone</th>
                    <th scope="col" className="px-3 py-2 font-medium">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {preview.rows.map((row) => (
                    <tr
                      key={row.line}
                      className={
                        !row.valid
                          ? 'bg-red-50'
                          : row.duplicate
                            ? 'bg-amber-50'
                            : ''
                      }
                    >
                      <td className="px-3 py-2 text-gray-500">{row.line}</td>
                      <td className="px-3 py-2">{row.name || '—'}</td>
                      <td className="px-3 py-2">
                        {row.phone ? formatPhone(row.phone) : row.phone_raw || '—'}
                      </td>
                      <td className="px-3 py-2">
                        {!row.valid || row.duplicate ? (
                          <span className="flex items-start gap-1 text-red-700">
                            <AlertTriangle size={14} className="mt-0.5 flex-shrink-0" />
                            {row.message}
                          </span>
                        ) : (
                          <span className="flex items-center gap-1 text-green-700">
                            <CheckCircle2 size={14} /> Ready
                          </span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <p className="text-sm text-gray-600">
              Rows marked <span className="font-medium">need fixing</span> are skipped; the rest
              are saved. Line numbers match your pasted file.
            </p>

            <div className="flex justify-between gap-3 pt-2 border-t">
              <button
                type="button"
                className="btn-secondary"
                onClick={() => setStage('paste')}
                disabled={busy}
              >
                Back to editing
              </button>
              <button
                type="button"
                className="btn-primary"
                onClick={runImport}
                disabled={busy || willWrite === 0}
              >
                {busy
                  ? 'Importing…'
                  : `Import ${willWrite} contact${willWrite === 1 ? '' : 's'}`}
              </button>
            </div>
          </div>
        )}

        {stage === 'done' && result && (
          <div className="px-6 py-5 space-y-4">
            <div className="flex items-center gap-2 text-green-700">
              <CheckCircle2 size={20} />
              <p className="font-medium">
                {result.created} new, {result.updated} updated, {result.skipped} skipped
              </p>
            </div>

            {result.errors.length > 0 && (
              <div className="max-h-56 overflow-y-auto border rounded-md">
                <table className="w-full text-sm">
                  <caption className="sr-only">Rows that were not imported</caption>
                  <thead className="bg-gray-50 text-left sticky top-0">
                    <tr>
                      <th scope="col" className="px-3 py-2 font-medium">Line</th>
                      <th scope="col" className="px-3 py-2 font-medium">Reason</th>
                    </tr>
                  </thead>
                  <tbody>
                    {result.errors.map((e, i) => (
                      <tr key={`${e.line}-${i}`} className="bg-red-50">
                        <td className="px-3 py-2 text-gray-500">{e.line}</td>
                        <td className="px-3 py-2 text-red-700">{e.message}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            <div className="flex justify-end pt-2 border-t">
              <button type="button" className="btn-primary" onClick={onClose}>
                Done
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
