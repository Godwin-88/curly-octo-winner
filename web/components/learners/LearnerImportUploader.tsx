'use client';

// Upload step for a learner roster CSV.
//
// The upload never validates rows. A school's spreadsheet is rarely complete,
// and rejecting the file over a blank cell would throw away the rows that are
// fine. Everything is staged as-is and becomes editable below, which is why
// this component has no preview step and no "3 rows are invalid" gate.

import { useRef, useState } from 'react';
import { AlertTriangle, FileUp, Loader2 } from 'lucide-react';

import { api, LearnerImportBatch } from '@/lib/api';

const SAMPLE = `Parent Name,Parent Phone,Student Name,Student Number,Grade,Stream,Tags
Mary Achieng,0712345001,Brian Achieng,S-1001,4,Blue,"boarding;2025 intake"
Grace Wanjiru,0712345002,Faith Wanjiru,S-1002,4,Blue,
Peter Otieno,,Daniel Otieno,,4,Red,`;

/** Columns the server recognises, and what each one is for. */
const COLUMNS: { name: string; required: boolean; note: string }[] = [
  { name: 'Parent Name', required: false, note: 'becomes a guardian when a phone is also given' },
  { name: 'Parent Phone', required: false, note: 'needed to create a guardian' },
  { name: 'Student Name', required: true, note: 'becomes the learner name' },
  { name: 'Student Number', required: true, note: 'becomes the learner number (UPI)' },
  { name: 'Grade', required: true, note: 'needed to create the learner' },
  { name: 'Stream', required: false, note: 'e.g. Blue, Red' },
  { name: 'Tags', required: false, note: 'separate several with ; or ,' },
];

export default function LearnerImportUploader({
  token,
  onStaged,
}: {
  token: string;
  onStaged: (batch: LearnerImportBatch) => void;
}) {
  const [csv, setCsv] = useState('');
  const [filename, setFilename] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  // Reads a chosen file into the textarea. Size is capped in the browser so a
  // huge spreadsheet does not freeze the tab before the server ever sees it.
  const onFile = async (file: File | undefined) => {
    if (!file) return;
    if (file.size > 5 * 1024 * 1024) {
      setError('That file is larger than 5 MB. Please split it into smaller files.');
      return;
    }
    setCsv(await file.text());
    setFilename(file.name);
    setError(null);
  };

  const submit = async () => {
    if (!csv.trim()) {
      setError('Choose a file or paste your rows first.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const batch = await api.createLearnerImport({ csv, filename: filename || undefined }, token);
      setCsv('');
      setFilename('');
      if (fileRef.current) fileRef.current.value = '';
      onStaged(batch);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not read that file');
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="card p-6 space-y-4" aria-labelledby="upload-heading">
      <div>
        <h2 id="upload-heading" className="text-lg font-semibold text-gray-900">
          Upload a roster
        </h2>
        <p className="text-sm text-gray-600 mt-1">
          Every row is saved, even if it is incomplete. You can fill in the gaps and
          finish the rest later.
        </p>
      </div>

      <div className="flex flex-wrap items-end gap-3">
        <div className="flex-1 min-w-48">
          <label className="label" htmlFor="roster-file">
            Spreadsheet (CSV)
          </label>
          <input
            id="roster-file"
            ref={fileRef}
            type="file"
            accept=".csv,text/csv"
            className="input"
            onChange={(e) => void onFile(e.target.files?.[0])}
            disabled={busy}
          />
        </div>
        <button type="button" className="btn-secondary" onClick={() => setCsv(SAMPLE)} disabled={busy}>
          Use a sample file
        </button>
      </div>

      <div>
        <label className="label" htmlFor="roster-csv">
          …or paste the rows
        </label>
        <textarea
          id="roster-csv"
          className="input font-mono text-xs"
          rows={5}
          value={csv}
          placeholder={SAMPLE}
          onChange={(e) => {
            setCsv(e.target.value);
            setError(null);
          }}
          disabled={busy}
        />
      </div>

      {error && (
        <p role="alert" className="flex items-start gap-2 text-sm text-red-700">
          <AlertTriangle size={16} className="mt-0.5 flex-shrink-0" aria-hidden="true" />
          {error}
        </p>
      )}

      <div className="flex justify-end">
        <button type="button" className="btn-primary" onClick={submit} disabled={busy || !csv.trim()}>
          {busy ? (
            <>
              <Loader2 size={16} className="animate-spin" aria-hidden="true" /> Staging rows…
            </>
          ) : (
            <>
              <FileUp size={16} aria-hidden="true" /> Stage these rows
            </>
          )}
        </button>
      </div>

      <details className="text-sm">
        <summary className="cursor-pointer text-gray-700 font-medium">What columns should I use?</summary>
        <div className="mt-3 space-y-3">
          <ul className="space-y-1">
            {COLUMNS.map((c) => (
              <li key={c.name} className="flex flex-wrap items-baseline gap-2">
                <code className="font-mono text-xs bg-gray-100 px-1.5 py-0.5 rounded">{c.name}</code>
                {c.required ? (
                  <span className="text-xs font-medium text-amber-700">needed to import a row</span>
                ) : (
                  <span className="text-xs text-gray-500">optional</span>
                )}
                <span className="text-xs text-gray-500">{c.note}</span>
              </li>
            ))}
          </ul>
          <p className="text-xs text-gray-500">
            Headings are matched loosely — <code className="font-mono">Parent Name</code>,{' '}
            <code className="font-mono">parent_name</code> and <code className="font-mono">Parent</code> all
            work. Columns like <em>House</em> or <em>Remarks</em> are ignored rather than rejected.
          </p>
        </div>
      </details>
    </section>
  );
}

