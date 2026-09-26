'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { X } from 'lucide-react';

import { APIError, Learner, LearningArea, Strand, SubStrand, api } from '@/lib/api';
import { useAuth } from '@/lib/auth';

export const RUBRIC_LEVELS = [
  { value: 1, label: 'Below Expectation', tone: 'bg-red-100 text-red-800 border-red-300' },
  { value: 2, label: 'Approaching', tone: 'bg-yellow-100 text-yellow-800 border-yellow-300' },
  { value: 3, label: 'Meeting', tone: 'bg-green-100 text-green-800 border-green-300' },
  { value: 4, label: 'Exceeding', tone: 'bg-blue-100 text-blue-800 border-blue-300' },
] as const;

interface Props {
  onClose: () => void;
  onSaved: (note: string) => void;
}

/**
 * Record one formative assessment observation.
 *
 * The curriculum is picked top-down (area → strand → sub-strand) because a
 * teacher thinks in those terms; the API only stores the sub-strand id, so the
 * three dropdowns are the cost of making that mental model usable. Each level
 * loads only once its parent is chosen, so the form stays fast on a school with
 * a full CBC curriculum.
 */
export default function ObservationModal({ onClose, onSaved }: Props) {
  const { token } = useAuth();

  const [learners, setLearners] = useState<Learner[]>([]);
  const [areas, setAreas] = useState<LearningArea[]>([]);
  const [strands, setStrands] = useState<Strand[]>([]);
  const [subStrands, setSubStrands] = useState<SubStrand[]>([]);
  const [loading, setLoading] = useState(true);

  const [learnerId, setLearnerId] = useState('');
  const [areaId, setAreaId] = useState('');
  const [strandId, setStrandId] = useState('');
  const [subStrandId, setSubStrandId] = useState('');
  const [level, setLevel] = useState(3);
  const [note, setNote] = useState('');
  const [term, setTerm] = useState(currentTerm());
  const [year, setYear] = useState(new Date().getFullYear());

  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [saving, setSaving] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);

  const learnerRef = useRef<HTMLSelectElement>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [learnerRows, areaRows] = await Promise.all([
          api.listLearners({}, token),
          api.listLearningAreas(token),
        ]);
        if (cancelled) return;
        setLearners(Array.isArray(learnerRows) ? learnerRows : []);
        setAreas(Array.isArray(areaRows) ? areaRows : []);
        if (areaRows?.[0]) setAreaId(areaRows[0].id);
      } catch (err) {
        if (!cancelled) {
          setServerError(
            err instanceof APIError ? err.message : 'Could not load learners and curriculum.'
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token]);

  useEffect(() => {
    if (!areaId) return;
    let cancelled = false;
    api
      .listStrands(areaId, token)
      .then((rows) => {
        if (cancelled) return;
        const safe = Array.isArray(rows) ? rows : [];
        setStrands(safe);
        setStrandId(safe[0]?.id ?? '');
      })
      .catch(() => {
        if (!cancelled) setStrands([]);
      });
    return () => {
      cancelled = true;
    };
  }, [areaId, token]);

  useEffect(() => {
    if (!strandId) {
      setSubStrands([]);
      return;
    }
    let cancelled = false;
    api
      .listSubStrands(strandId, token)
      .then((rows) => {
        if (cancelled) return;
        const safe = Array.isArray(rows) ? rows : [];
        setSubStrands(safe);
        setSubStrandId(safe[0]?.id ?? '');
      })
      .catch(() => {
        if (!cancelled) setSubStrands([]);
      });
    return () => {
      cancelled = true;
    };
  }, [strandId, token]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !saving) onClose();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose, saving]);

  const learnerError = touched.learnerId && !learnerId ? 'Choose the learner' : '';
  const subStrandError = touched.subStrandId && !subStrandId ? 'Choose a sub-strand' : '';
  const canSubmit = Boolean(learnerId) && Boolean(subStrandId);

  const noCurriculum = useMemo(
    () => !loading && (areas.length === 0 || subStrands.length === 0),
    [loading, areas.length, subStrands.length]
  );

  async function save() {
    setTouched({ learnerId: true, subStrandId: true });
    if (!canSubmit || saving) return;

    setSaving(true);
    setServerError(null);
    try {
      await api.createAssessment(
        {
          learner_id: learnerId,
          sub_strand_id: subStrandId,
          rubric_level: level,
          note: note.trim(),
          term,
          year,
        },
        token
      );
      onSaved('Observation recorded.');
    } catch (err) {
      setServerError(
        err instanceof APIError ? err.message : 'Could not save the observation. Please try again.'
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
      aria-labelledby="observation-modal-title"
      onClick={(e) => {
        if (e.target === e.currentTarget && !saving) onClose();
      }}
    >
      <div className="bg-white rounded-lg shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-5 border-b sticky top-0 bg-white">
          <h2 id="observation-modal-title" className="text-lg font-semibold">
            New Observation
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
          <div>
            <label className="label" htmlFor="observation-learner">
              Learner <span className="text-red-500">*</span>
            </label>
            <select
              id="observation-learner"
              ref={learnerRef}
              className="input"
              value={learnerId}
              onChange={(e) => setLearnerId(e.target.value)}
              onBlur={() => setTouched((t) => ({ ...t, learnerId: true }))}
              aria-invalid={Boolean(learnerError)}
              disabled={loading}
            >
              <option value="">{loading ? 'Loading learners…' : 'Select a learner…'}</option>
              {learners.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.full_name}
                  {[l.grade, l.stream].filter(Boolean).join(' · ')}
                </option>
              ))}
            </select>
            {learnerError && <p className="text-sm text-red-600 mt-1">{learnerError}</p>}
          </div>

          <fieldset>
            <legend className="label">What is being assessed</legend>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <div>
                <label className="text-xs text-gray-500" htmlFor="observation-area">
                  Learning area
                </label>
                <select
                  id="observation-area"
                  className="input"
                  value={areaId}
                  onChange={(e) => setAreaId(e.target.value)}
                >
                  {areas.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.name}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-xs text-gray-500" htmlFor="observation-strand">
                  Strand
                </label>
                <select
                  id="observation-strand"
                  className="input"
                  value={strandId}
                  onChange={(e) => setStrandId(e.target.value)}
                >
                  {strands.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="text-xs text-gray-500" htmlFor="observation-substrand">
                  Sub-strand <span className="text-red-500">*</span>
                </label>
                <select
                  id="observation-substrand"
                  className="input"
                  value={subStrandId}
                  onChange={(e) => setSubStrandId(e.target.value)}
                  onBlur={() => setTouched((t) => ({ ...t, subStrandId: true }))}
                  aria-invalid={Boolean(subStrandError)}
                >
                  {subStrands.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            {subStrandError && <p className="text-sm text-red-600 mt-1">{subStrandError}</p>}
            {noCurriculum && (
              <p className="text-sm text-amber-800 bg-amber-50 p-2 rounded mt-2">
                No sub-strands are set up yet. Add them under Academic → Curriculum first.
              </p>
            )}
          </fieldset>

          <fieldset>
            <legend className="label">Rubric level</legend>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
              {RUBRIC_LEVELS.map((r) => (
                <button
                  key={r.value}
                  type="button"
                  aria-pressed={level === r.value}
                  onClick={() => setLevel(r.value)}
                  className={`px-3 py-2 text-xs font-medium rounded border transition-colors ${
                    level === r.value
                      ? r.tone
                      : 'border-gray-300 text-gray-600 hover:bg-gray-50'
                  }`}
                >
                  {r.value}. {r.label}
                </button>
              ))}
            </div>
          </fieldset>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="label" htmlFor="observation-term">
                Term
              </label>
              <select
                id="observation-term"
                className="input"
                value={term}
                onChange={(e) => setTerm(Number(e.target.value))}
              >
                <option value={1}>Term 1</option>
                <option value={2}>Term 2</option>
                <option value={3}>Term 3</option>
              </select>
            </div>
            <div>
              <label className="label" htmlFor="observation-year">
                Year
              </label>
              <input
                id="observation-year"
                type="number"
                className="input"
                value={year}
                onChange={(e) => setYear(Number(e.target.value))}
              />
            </div>
          </div>

          <div>
            <label className="label" htmlFor="observation-note">
              What did the learner do?
            </label>
            <textarea
              id="observation-note"
              className="input"
              rows={3}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="A short, specific observation a parent would understand."
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
              {saving ? 'Saving…' : 'Record observation'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

/** Terms run Jan-Apr, May-Aug, Sep-Dec; the gaps fall back to the term before. */
function currentTerm(): number {
  const month = new Date().getMonth() + 1;
  if (month <= 4) return 1;
  if (month <= 8) return 2;
  return 3;
}
