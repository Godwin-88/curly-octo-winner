'use client';

// Editable table of staged roster rows.
//
// The whole point of staging is that a row is editable here until it is good
// enough to become a learner. Each cell saves itself on blur, and the server
// recomputes which fields are still missing -- so filling in a grade is all it
// takes for a row to become importable, and this component never decides that
// for itself. `missing` comes from the database.

import { useState } from 'react';
import { AlertTriangle, Check, CheckCircle2, Loader2, X } from 'lucide-react';

import {
  api,
  LearnerImportRequiredField,
  LearnerImportRow,
  LearnerImportRowPatch,
} from '@/lib/api';

type EditableField =
  | 'parent_name'
  | 'parent_phone'
  | 'student_name'
  | 'student_number'
  | 'grade'
  | 'stream'
  | 'tags';

/** Human labels for the fields the server reports as missing. */
const FIELD_LABELS: Record<LearnerImportRequiredField, string> = {
  student_name: 'Student name',
  student_number: 'Student number',
  grade: 'Grade',
};

const GRADES = [
  'PP1', 'PP2',
  'Grade 1', 'Grade 2', 'Grade 3', 'Grade 4', 'Grade 5', 'Grade 6',
  'Grade 7', 'Grade 8', 'Grade 9',
];

/** Per-row uncommitted edits, so a blur that changes nothing sends nothing. */
type Drafts = Record<string, Partial<Record<EditableField, string>>>;

function cellValue(row: LearnerImportRow, field: EditableField): string {
  if (field === 'tags') return row.tags.join(', ');
  return row[field] ?? '';
}

/** One editable cell: saves itself on blur, or on Enter. */
function EditableCell({
  row,
  field,
  label,
  draft,
  onChange,
  onCommit,
  saving,
  invalid,
  type = 'text',
  placeholder,
  list,
  disabled,
}: {
  row: LearnerImportRow;
  field: EditableField;
  label: string;
  draft: string | undefined;
  onChange: (value: string) => void;
  onCommit: () => void;
  saving: boolean;
  invalid: boolean;
  type?: string;
  placeholder?: string;
  list?: string[];
  disabled?: boolean;
}) {
  const value = draft ?? cellValue(row, field);
  return (
    <div className="relative">
      <label className="sr-only" htmlFor={`${row.id}-${field}`}>
        {label} for line {row.row_number}
      </label>
      <input
        id={`${row.id}-${field}`}
        type={type}
        className={`input text-sm py-1.5 ${invalid ? 'border-amber-400 bg-amber-50/60' : ''}`}
        value={value}
        placeholder={placeholder}
        list={list ? `${row.id}-${field}-list` : undefined}
        disabled={disabled}
        aria-invalid={invalid || undefined}
        onChange={(e) => onChange(e.target.value)}
        onBlur={onCommit}
        onKeyDown={(e) => {
          // Enter commits and does not submit anything else; Escape drops the edit.
          if (e.key === 'Enter') {
            e.currentTarget.blur();
          } else if (e.key === 'Escape') {
            onChange(cellValue(row, field));
            e.currentTarget.blur();
          }
        }}
      />
      {list && (
        <datalist id={`${row.id}-${field}-list`}>
          {list.map((option) => (
            <option key={option} value={option} />
          ))}
        </datalist>
      )}
      {saving && (
        <Loader2
          size={13}
          className="absolute right-2 top-1/2 -translate-y-1/2 animate-spin text-gray-400"
          aria-label="Saving"
        />
      )}
    </div>
  );
}

/** Builds the patch for one cell, or null when nothing actually changed. */
function patchFor(row: LearnerImportRow, field: EditableField, draft: string): LearnerImportRowPatch | null {
  const current = cellValue(row, field);
  if (draft === current) return null;
  if (field === 'tags') {
    return {
      tags: draft
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean),
    };
  }
  return { [field]: draft };
}


export default function LearnerImportRows({
  token,
  rows,
  onChanged,
}: {
  token: string;
  rows: LearnerImportRow[];
  /** Called after any successful write, so the caller can refetch. */
  onChanged: () => void;
}) {
  const [drafts, setDrafts] = useState<Drafts>({});
  const [savingCell, setSavingCell] = useState<string | null>(null);
  const [busyRow, setBusyRow] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const setDraft = (rowId: string, field: EditableField, value: string) =>
    setDrafts((d) => ({ ...d, [rowId]: { ...d[rowId], [field]: value } }));

  const commit = async (row: LearnerImportRow, field: EditableField) => {
    const draft = drafts[row.id]?.[field];
    if (draft === undefined) return;
    const patch = patchFor(row, field, draft);
    if (!patch) {
      // Typed and reverted, or blurred without changing anything: drop the
      // draft rather than sending a pointless write.
      setDrafts((d) => {
        const next = { ...d };
        if (next[row.id]) {
          const fields = { ...next[row.id] };
          delete fields[field];
          if (Object.keys(fields).length === 0) delete next[row.id];
          else next[row.id] = fields;
        }
        return next;
      });
      return;
    }

    const cellKey = `${row.id}:${field}`;
    setSavingCell(cellKey);
    setError(null);
    try {
      await api.updateLearnerImportRow(row.id, patch, token);
      setDrafts((d) => {
        const next = { ...d };
        const fields = { ...(next[row.id] ?? {}) };
        delete fields[field];
        if (Object.keys(fields).length === 0) delete next[row.id];
        else next[row.id] = fields;
        return next;
      });
      onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save that change');
    } finally {
      setSavingCell(null);
    }
  };

  const promote = async (row: LearnerImportRow) => {
    setBusyRow(row.id);
    setError(null);
    try {
      await api.promoteLearnerImportRow(row.id, token);
      onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not import that row');
    } finally {
      setBusyRow(null);
    }
  };

  const remove = async (row: LearnerImportRow) => {
    setBusyRow(row.id);
    setError(null);
    try {
      await api.deleteLearnerImportRow(row.id, token);
      onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not remove that row');
    } finally {
      setBusyRow(null);
    }
  };


  return (
    <div className="card overflow-hidden">
      {error && (
        <p role="alert" className="flex items-start gap-2 text-sm text-red-700 border-b border-red-200 bg-red-50 px-4 py-3">
          <AlertTriangle size={16} className="mt-0.5 flex-shrink-0" aria-hidden="true" />
          {error}
        </p>
      )}

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <caption className="sr-only">
            Staged roster rows. Edit any cell; a row can be imported once it has a
            student name, student number and grade.
          </caption>
          <thead>
            <tr className="bg-gray-50 text-left text-gray-700 border-b border-gray-200">
              <th scope="col" className="px-3 py-3 font-medium w-12">Line</th>
              <th scope="col" className="px-3 py-3 font-medium">Parent name</th>
              <th scope="col" className="px-3 py-3 font-medium">Parent phone</th>
              <th scope="col" className="px-3 py-3 font-medium">Student name</th>
              <th scope="col" className="px-3 py-3 font-medium">Student number</th>
              <th scope="col" className="px-3 py-3 font-medium w-28">Grade</th>
              <th scope="col" className="px-3 py-3 font-medium hidden lg:table-cell">Stream</th>
              <th scope="col" className="px-3 py-3 font-medium hidden lg:table-cell">Tags</th>
              <th scope="col" className="px-3 py-3 font-medium w-44">Status</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => {
              const missing = new Set(row.missing);
              const locked = row.status === 'imported';
              const cellProps = (field: EditableField) => ({
                row,
                field,
                draft: drafts[row.id]?.[field],
                onChange: (v: string) => setDraft(row.id, field, v),
                onCommit: () => void commit(row, field),
                saving: savingCell === `${row.id}:${field}`,
                invalid: missing.has(field as LearnerImportRequiredField),
                disabled: locked,
              });

              return (
                <tr
                  key={row.id}
                  className={`border-b border-gray-100 last:border-b-0 ${
                    locked ? 'bg-green-50/40' : 'hover:bg-gray-50'
                  }`}
                >
                  <td className="px-3 py-2 align-top text-gray-500 font-mono text-xs">{row.row_number}</td>
                  <td className="px-3 py-2 align-top">
                    <EditableCell {...cellProps('parent_name')} label="Parent name" />
                  </td>
                  <td className="px-3 py-2 align-top">
                    <EditableCell {...cellProps('parent_phone')} label="Parent phone" type="tel" />
                  </td>
                  <td className="px-3 py-2 align-top">
                    <EditableCell {...cellProps('student_name')} label="Student name" />
                  </td>
                  <td className="px-3 py-2 align-top">
                    <EditableCell {...cellProps('student_number')} label="Student number" />
                  </td>
                  <td className="px-3 py-2 align-top">
                    <EditableCell {...cellProps('grade')} label="Grade" list={GRADES} placeholder="—" />
                  </td>
                  <td className="px-3 py-2 align-top hidden lg:table-cell">
                    <EditableCell {...cellProps('stream')} label="Stream" />
                  </td>
                  <td className="px-3 py-2 align-top hidden lg:table-cell">
                    <EditableCell
                      {...cellProps('tags')}
                      label="Tags"
                      placeholder="boarding, 2025 intake"
                    />
                  </td>

                  <td className="px-3 py-2 align-top">
                    {locked ? (
                      <span className="flex items-center gap-1 text-green-700 text-sm">
                        <CheckCircle2 size={15} aria-hidden="true" /> Imported
                      </span>
                    ) : row.is_sufficient ? (
                      <button
                        type="button"
                        className="btn-primary text-sm py-1.5"
                        onClick={() => void promote(row)}
                        disabled={busyRow === row.id}
                      >
                        {busyRow === row.id ? (
                          <>
                            <Loader2 size={14} className="animate-spin" aria-hidden="true" /> Importing…
                          </>
                        ) : (
                          <>
                            <Check size={14} aria-hidden="true" /> Import
                          </>
                        )}
                      </button>
                    ) : (
                      <span className="flex flex-col gap-1">
                        <span className="flex items-start gap-1 text-amber-700 text-xs">
                          <AlertTriangle size={13} className="mt-0.5 flex-shrink-0" aria-hidden="true" />
                          <span>Needs {row.missing.map((f) => FIELD_LABELS[f]).join(', ')}</span>
                        </span>
                        <button
                          type="button"
                          className="flex items-center gap-1 text-xs text-gray-500 hover:text-red-700 self-start"
                          onClick={() => void remove(row)}
                          disabled={busyRow === row.id}
                        >
                          <X size={12} aria-hidden="true" /> Remove this row
                        </button>
                      </span>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

