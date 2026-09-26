'use client';

import { useEffect, useRef, useState } from 'react';
import { X } from 'lucide-react';

import {
  APIError,
  CoreCompetency,
  LearningArea,
  Strand,
  SubStrand,
  Value,
  api,
} from '@/lib/api';
import { useAuth } from '@/lib/auth';

/**
 * The five kinds of thing the CBC curriculum is made of. They differ only in
 * whether they hang off a parent and whether they carry a grade level, so one
 * form serves all five instead of five near-identical dialogs.
 */
export type CurriculumKind = 'learning-area' | 'strand' | 'sub-strand' | 'competency' | 'value';

export const CURRICULUM_LABELS: Record<
  CurriculumKind,
  { singular: string; plural: string; parent: string | null }
> = {
  'learning-area': { singular: 'learning area', plural: 'learning areas', parent: null },
  strand: { singular: 'strand', plural: 'strands', parent: 'Learning area' },
  'sub-strand': { singular: 'sub-strand', plural: 'sub-strands', parent: 'Strand' },
  competency: { singular: 'core competency', plural: 'core competencies', parent: null },
  value: { singular: 'value', plural: 'values', parent: null },
};

export const GRADE_LEVELS = [
  'Pre-Primary',
  'Lower Primary',
  'Upper Primary',
  'Junior School',
  'Secondary',
];

export interface ParentOption {
  id: string;
  label: string;
}

export type AnyCurriculumItem = LearningArea | Strand | SubStrand | CoreCompetency | Value;

interface Props {
  kind: CurriculumKind;
  /** null = create a new item; an item = edit it. */
  item: AnyCurriculumItem | null;
  /** Options for the parent select (strands need areas, sub-strands need strands). */
  parents?: ParentOption[];
  onClose: () => void;
  onSaved: (item: AnyCurriculumItem, mode: 'created' | 'updated') => void;
}

interface FormState {
  name: string;
  kicd_code: string;
  grade_level: string;
  description: string;
  parent_id: string;
}

const EMPTY: FormState = {
  name: '',
  kicd_code: '',
  grade_level: 'Lower Primary',
  description: '',
  parent_id: '',
};

const PLACEHOLDERS: Record<CurriculumKind, string> = {
  'learning-area': 'Mathematics',
  strand: 'Numbers',
  'sub-strand': 'Counting and place value',
  competency: 'Critical Thinking',
  value: 'Responsibility',
};

/**
 * Create / edit dialog for any curriculum item.
 *
 * The KICD code is optional on purpose: schools add their own strands before
 * the ministry code is known, and forcing a code would mean inventing data.
 * What is not optional is the name, and a parent where the hierarchy needs one
 * — both are checked here so the user learns before the round trip, and the
 * server checks them again because it is the authority.
 */
export default function CurriculumItemModal({ kind, item, parents = [], onClose, onSaved }: Props) {
  const { token } = useAuth();
  const editing = Boolean(item);
  const meta = CURRICULUM_LABELS[kind];
  const needsParent = Boolean(meta.parent);

  const [form, setForm] = useState<FormState>({ ...EMPTY });
  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [saving, setSaving] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);

  const nameRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (item) {
      const anyItem = item as unknown as Record<string, string | undefined>;
      setForm({
        name: anyItem.name ?? '',
        kicd_code: anyItem.kicd_code ?? '',
        grade_level: anyItem.grade_level || 'Lower Primary',
        description: anyItem.description ?? '',
        parent_id: (anyItem.learning_area_id as string) || (anyItem.strand_id as string) || '',
      });
    } else {
      setForm({ ...EMPTY, parent_id: parents[0]?.id ?? '' });
    }
    setServerError(null);
    setTouched({});
    nameRef.current?.focus();
  }, [item, parents]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !saving) onClose();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose, saving]);

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setForm((f) => ({ ...f, [key]: value }));

  const nameError = touched.name && !form.name.trim() ? `Name the ${meta.singular}` : '';
  const parentError =
    touched.parent_id && needsParent && !form.parent_id
      ? `Choose the ${meta.parent?.toLowerCase()} this belongs to`
      : '';

  const canSubmit = Boolean(form.name.trim()) && (!needsParent || Boolean(form.parent_id));

  async function save() {
    setTouched({ name: true, parent_id: true });
    if (!canSubmit || saving) return;

    setSaving(true);
    setServerError(null);
    try {
      const payload = {
        name: form.name.trim(),
        kicd_code: form.kicd_code.trim(),
        description: form.description.trim(),
      };
      let saved: AnyCurriculumItem;

      if (kind === 'learning-area') {
        const body = { ...payload, grade_level: form.grade_level };
        saved = editing
          ? await api.updateLearningArea((item as LearningArea).id, body, token)
          : await api.createLearningArea(body as unknown as LearningArea, token);
      } else if (kind === 'strand') {
        const body = { ...payload, learning_area_id: form.parent_id };
        saved = editing
          ? await api.updateStrand((item as Strand).id, body, token)
          : await api.createStrand(body as unknown as Strand, token);
      } else if (kind === 'sub-strand') {
        const body = { ...payload, strand_id: form.parent_id };
        saved = editing
          ? await api.updateSubStrand((item as SubStrand).id, body, token)
          : await api.createSubStrand(body as unknown as SubStrand, token);
      } else if (kind === 'competency') {
        saved = editing
          ? await api.updateCoreCompetency((item as CoreCompetency).id, payload, token)
          : await api.createCoreCompetency(payload as unknown as CoreCompetency, token);
      } else {
        saved = editing
          ? await api.updateValue((item as Value).id, payload, token)
          : await api.createValue(payload as unknown as Value, token);
      }

      onSaved(saved, editing ? 'updated' : 'created');
    } catch (err) {
      setServerError(
        err instanceof APIError
          ? err.message
          : 'Could not save. Check your connection and try again.'
      );
    } finally {
      setSaving(false);
    }
  }

  return (
    <div
      className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="curriculum-modal-title"
      onClick={(e) => {
        if (e.target === e.currentTarget && !saving) onClose();
      }}
    >
      <div className="bg-white rounded-lg shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-5 border-b sticky top-0 bg-white">
          <h2 id="curriculum-modal-title" className="text-lg font-semibold">
            {editing ? 'Edit' : 'Add'} {meta.singular}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
            aria-label="Close"
            disabled={saving}
          >
            <X size={20} />
          </button>
        </div>

        <form
          className="p-5 space-y-4"
          onSubmit={(e) => {
            e.preventDefault();
            save();
          }}
        >
          {needsParent && (
            <div>
              <label className="label" htmlFor="curriculum-parent">
                {meta.parent}
              </label>
              <select
                id="curriculum-parent"
                className="input"
                value={form.parent_id}
                onChange={(e) => set('parent_id', e.target.value)}
                onBlur={() => setTouched((t) => ({ ...t, parent_id: true }))}
                aria-invalid={Boolean(parentError)}
                disabled={editing}
              >
                <option value="">Select…</option>
                {parents.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.label}
                  </option>
                ))}
              </select>
              {parentError && <p className="text-sm text-red-600 mt-1">{parentError}</p>}
              {editing && (
                <p className="text-xs text-gray-500 mt-1">
                  An item cannot be moved between parents — delete and re-add it instead.
                </p>
              )}
            </div>
          )}

          <div>
            <label className="label" htmlFor="curriculum-name">
              Name <span className="text-red-500">*</span>
            </label>
            <input
              id="curriculum-name"
              ref={nameRef}
              className="input"
              value={form.name}
              onChange={(e) => set('name', e.target.value)}
              onBlur={() => setTouched((t) => ({ ...t, name: true }))}
              aria-invalid={Boolean(nameError)}
              placeholder={PLACEHOLDERS[kind]}
            />
            {nameError && <p className="text-sm text-red-600 mt-1">{nameError}</p>}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label" htmlFor="curriculum-code">
                KICD code
              </label>
              <input
                id="curriculum-code"
                className="input"
                value={form.kicd_code}
                onChange={(e) => set('kicd_code', e.target.value.toUpperCase())}
                aria-describedby="curriculum-code-help"
                placeholder="e.g. MTH"
              />
              <p id="curriculum-code-help" className="text-xs text-gray-500 mt-1">
                Optional. Must be unique in your school.
              </p>
            </div>

            {kind === 'learning-area' && (
              <div>
                <label className="label" htmlFor="curriculum-grade">
                  Grade level
                </label>
                <select
                  id="curriculum-grade"
                  className="input"
                  value={form.grade_level}
                  onChange={(e) => set('grade_level', e.target.value)}
                >
                  {GRADE_LEVELS.map((g) => (
                    <option key={g} value={g}>
                      {g}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>

          <div>
            <label className="label" htmlFor="curriculum-description">
              Description
            </label>
            <textarea
              id="curriculum-description"
              className="input"
              rows={3}
              value={form.description}
              onChange={(e) => set('description', e.target.value)}
              placeholder="What this covers, or how it is taught."
            />
          </div>

          {serverError && (
            <p role="alert" className="text-sm text-red-600 bg-red-50 p-3 rounded">
              {serverError}
            </p>
          )}

          <div className="flex justify-end gap-3 pt-4 border-t">
            <button type="button" className="btn-secondary" onClick={onClose} disabled={saving}>
              Cancel
            </button>
            <button type="submit" className="btn-primary" disabled={!canSubmit || saving}>
              {saving ? 'Saving…' : editing ? 'Save changes' : `Add ${meta.singular}`}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
