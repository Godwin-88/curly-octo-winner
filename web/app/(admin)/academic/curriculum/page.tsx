'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { BookOpen, Pencil, Plus, Trash2 } from 'lucide-react';

import {
  APIError,
  CoreCompetency,
  LearningArea,
  Strand,
  SubStrand,
  Value,
  asList,
  api,
} from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { EmptyState } from '@/components/ui/EmptyState';
import { ErrorState } from '@/components/ui/ErrorState';
import { Skeleton } from '@/components/ui/Skeleton';
import CurriculumItemModal, {
  CURRICULUM_LABELS,
  type AnyCurriculumItem,
  type CurriculumKind,
} from '@/components/academic/CurriculumItemModal';

const TABS: { id: CurriculumKind; blurb: string }[] = [
  { id: 'learning-area', blurb: 'The subjects your school teaches, e.g. Mathematics, Kiswahili.' },
  { id: 'strand', blurb: 'The main threads inside a learning area, e.g. Numbers.' },
  { id: 'sub-strand', blurb: 'The topics inside a strand that carry the learning outcomes.' },
  { id: 'competency', blurb: 'The KICD core competencies every learner develops.' },
  { id: 'value', blurb: 'The KICD values and how they show up in daily school life.' },
];

/**
 * CBC curriculum management.
 *
 * The curriculum is a tree — area → strand → sub-strand — so each tab shows one
 * level with its parent chosen above it. This replaced a placeholder page whose
 * "Add Item" button did nothing and which listed nothing at all.
 */
export default function CurriculumPage() {
  const { token } = useAuth();

  const [tab, setTab] = useState<CurriculumKind>('learning-area');

  const [areas, setAreas] = useState<LearningArea[]>([]);
  const [competencies, setCompetencies] = useState<CoreCompetency[]>([]);
  const [values, setValues] = useState<Value[]>([]);
  const [strands, setStrands] = useState<Strand[]>([]);
  const [subStrands, setSubStrands] = useState<SubStrand[]>([]);

  const [areaId, setAreaId] = useState('');
  const [strandId, setStrandId] = useState('');

  const [loading, setLoading] = useState(true);
  const [childLoading, setChildLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<AnyCurriculumItem | null>(null);
  const [pendingDelete, setPendingDelete] = useState<AnyCurriculumItem | null>(null);
  const [busy, setBusy] = useState(false);

  // The top level is needed by every tab (it names the strands), so it loads
  // once here instead of being refetched on each tab change.
  const loadTopLevel = useCallback(async () => {
    try {
      setError(null);
      const [areaRows, competencyRows, valueRows] = await Promise.all([
        api.listLearningAreas(token),
        api.listCoreCompetencies(token),
        api.listValues(token),
      ]);
      setAreas(asList(areaRows));
      setCompetencies(asList(competencyRows));
      setValues(asList(valueRows));
      setAreaId((current) => current || (areaRows[0]?.id ?? ''));
    } catch (err) {
      setError(err instanceof APIError ? err.message : 'Could not load the curriculum.');
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    loadTopLevel();
  }, [loadTopLevel]);

  useEffect(() => {
    if (tab !== 'strand' && tab !== 'sub-strand') return;
    if (!areaId) {
      setStrands([]);
      return;
    }
    let cancelled = false;
    setChildLoading(true);
    api
      .listStrands(areaId, token)
      .then((rows) => {
        if (cancelled) return;
        const safe = asList(rows);
        setStrands(safe);
        setStrandId((current) => (safe.some((s) => s.id === current) ? current : safe[0]?.id ?? ''));
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof APIError ? err.message : 'Could not load the strands.');
        }
      })
      .finally(() => {
        if (!cancelled) setChildLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [tab, areaId, token]);

  useEffect(() => {
    if (tab !== 'sub-strand') return;
    if (!strandId) {
      setSubStrands([]);
      return;
    }
    let cancelled = false;
    setChildLoading(true);
    api
      .listSubStrands(strandId, token)
      .then((rows) => {
        if (!cancelled) setSubStrands(asList(rows));
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof APIError ? err.message : 'Could not load the sub-strands.');
        }
      })
      .finally(() => {
        if (!cancelled) setChildLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [tab, strandId, token]);

  const rows = useMemo<AnyCurriculumItem[]>(() => {
    if (tab === 'learning-area') return areas;
    if (tab === 'strand') return strands;
    if (tab === 'sub-strand') return subStrands;
    if (tab === 'competency') return competencies;
    return values;
  }, [tab, areas, strands, subStrands, competencies, values]);

  const parentOptions = useMemo(() => {
    if (tab === 'strand') return areas.map((a) => ({ id: a.id, label: a.name }));
    if (tab === 'sub-strand') return strands.map((s) => ({ id: s.id, label: s.name }));
    return [];
  }, [tab, areas, strands]);

  const meta = CURRICULUM_LABELS[tab];
  const activeBlurb = TABS.find((t) => t.id === tab)?.blurb ?? '';

  // A level that hangs off a parent cannot be added until that parent exists.
  const missingParent =
    (tab === 'strand' && areas.length === 0) || (tab === 'sub-strand' && strands.length === 0);

  // Each level keeps its own state, so writes go through the matching setter.
  // A union of setState functions would lose the element type and silently allow
  // a strand to be spliced into the competencies list.
  function onSaved(item: AnyCurriculumItem, mode: 'created' | 'updated') {
    setModalOpen(false);
    setEditing(null);
    setNotice(`${capitalise(meta.singular)} ${mode === 'created' ? 'added' : 'updated'}.`);

    if (tab === 'learning-area') setAreas((prev) => upsert(prev, item as LearningArea));
    else if (tab === 'strand') setStrands((prev) => upsert(prev, item as Strand));
    else if (tab === 'sub-strand') setSubStrands((prev) => upsert(prev, item as SubStrand));
    else if (tab === 'competency') setCompetencies((prev) => upsert(prev, item as CoreCompetency));
    else setValues((prev) => upsert(prev, item as Value));
  }

  async function onDeleteConfirmed() {
    if (!pendingDelete) return;
    const id = pendingDelete.id;
    setBusy(true);
    try {
      if (tab === 'learning-area') await api.deleteLearningArea(id, token);
      else if (tab === 'strand') await api.deleteStrand(id, token);
      else if (tab === 'sub-strand') await api.deleteSubStrand(id, token);
      else if (tab === 'competency') await api.deleteCoreCompetency(id, token);
      else await api.deleteValue(id, token);

      const name = (pendingDelete as { name: string }).name;
      setNotice(`${capitalise(meta.singular)} “${name}” deleted.`);
      if (tab === 'learning-area') setAreas((prev) => prev.filter((r) => r.id !== id));
      else if (tab === 'strand') setStrands((prev) => prev.filter((r) => r.id !== id));
      else if (tab === 'sub-strand') setSubStrands((prev) => prev.filter((r) => r.id !== id));
      else if (tab === 'competency') setCompetencies((prev) => prev.filter((r) => r.id !== id));
      else setValues((prev) => prev.filter((r) => r.id !== id));
    } catch (err) {
      // A blocked delete (children still exist) arrives as a 409 with a sentence
      // explaining what to remove first — show it rather than a generic failure.
      setNotice(
        err instanceof APIError
          ? err.message
          : 'Could not delete. Check your connection and try again.'
      );
    } finally {
      setBusy(false);
      setPendingDelete(null);
    }
  }

  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error && rows.length === 0) {
    return <ErrorState title="Curriculum unavailable" message={error} onRetry={loadTopLevel} />;
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6 gap-4">
        <div>
          <h1 className="text-2xl font-bold">CBC Curriculum Structure</h1>
          <p className="text-sm text-gray-500 mt-1">{activeBlurb}</p>
        </div>
        <button
          type="button"
          className="btn-primary flex items-center gap-2 whitespace-nowrap"
          onClick={() => {
            setEditing(null);
            setModalOpen(true);
          }}
          disabled={missingParent}
          title={missingParent ? (tab === 'strand' ? 'Add a learning area first' : 'Add a strand first') : undefined}
        >
          <Plus size={16} /> Add {meta.singular}
        </button>
      </div>

      <div className="flex flex-wrap gap-2 mb-4 border-b" role="tablist" aria-label="Curriculum levels">
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
            {CURRICULUM_LABELS[t.id].plural}
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

      {tab === 'strand' && (
        <ParentPicker
          label="Learning area"
          value={areaId}
          options={areas.map((a) => ({ id: a.id, label: `${a.name} (${a.grade_level})` }))}
          onChange={setAreaId}
          emptyLabel="No learning areas yet — add one on the Learning areas tab."
        />
      )}

      {tab === 'sub-strand' && (
        <>
          <ParentPicker
            label="Learning area"
            value={areaId}
            options={areas.map((a) => ({ id: a.id, label: `${a.name} (${a.grade_level})` }))}
            onChange={(id) => {
              setAreaId(id);
              setStrandId('');
            }}
            emptyLabel="No learning areas yet — add one on the Learning areas tab."
          />
          {strands.length > 0 && (
            <ParentPicker
              label="Strand"
              value={strandId}
              options={strands.map((s) => ({ id: s.id, label: s.name }))}
              onChange={setStrandId}
            />
          )}
        </>
      )}

      <div className="card">
        {childLoading ? (
          <div className="p-4 space-y-3" aria-busy="true">
            {[0, 1, 2].map((i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : rows.length === 0 ? (
          <div className="p-4">
            <EmptyState
              title={`No ${meta.plural} yet`}
              message={
                missingParent
                  ? tab === 'strand'
                    ? 'Add a learning area first, then you can add strands inside it.'
                    : 'Add a strand first, then you can add sub-strands inside it.'
                  : `Add your first ${meta.singular} to start building this part of the curriculum.`
              }
              action={
                <button
                  type="button"
                  className="btn-primary"
                  onClick={() => {
                    setEditing(null);
                    setModalOpen(true);
                  }}
                  disabled={missingParent}
                >
                  <Plus size={16} /> Add {meta.singular}
                </button>
              }
            />
          </div>
        ) : (
          <ul className="divide-y">
            {rows.map((row) => {
              const anyRow = row as unknown as Record<string, string | undefined>;
              return (
                <li key={row.id} className="flex items-center gap-3 p-4 hover:bg-gray-50">
                  <BookOpen size={16} className="text-gray-400 shrink-0" aria-hidden="true" />
                  <div className="min-w-0 flex-1">
                    <p className="font-medium text-gray-900 truncate">
                      {anyRow.name}
                      {anyRow.kicd_code && (
                        <span className="ml-2 text-xs font-mono text-gray-600 bg-gray-100 px-1.5 py-0.5 rounded">
                          {anyRow.kicd_code}
                        </span>
                      )}
                    </p>
                    <p className="text-sm text-gray-500 truncate">
                      {anyRow.grade_level || parentLabelFor(tab, anyRow, areas, strands) || (anyRow.description ?? '')}
                    </p>
                  </div>
                  <button
                    type="button"
                    className="p-2 text-gray-500 hover:text-blue-600"
                    aria-label={`Edit ${anyRow.name}`}
                    onClick={() => {
                      setEditing(row);
                      setModalOpen(true);
                    }}
                  >
                    <Pencil size={16} />
                  </button>
                  <button
                    type="button"
                    className="p-2 text-gray-500 hover:text-red-600"
                    aria-label={`Delete ${anyRow.name}`}
                    onClick={() => setPendingDelete(row)}
                  >
                    <Trash2 size={16} />
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>

      {modalOpen && (
        <CurriculumItemModal
          kind={tab}
          item={editing}
          parents={parentOptions}
          onClose={() => {
            setModalOpen(false);
            setEditing(null);
          }}
          onSaved={onSaved}
        />
      )}

      {pendingDelete && (
        <ConfirmDialog
          title={`Delete this ${meta.singular}?`}
          message={`“${(pendingDelete as { name: string }).name}” will be removed permanently. If something still depends on it, the school will be told what to remove first.`}
          busy={busy}
          onCancel={() => setPendingDelete(null)}
          onConfirm={onDeleteConfirmed}
        />
      )}
    </div>
  );
}

function ParentPicker({
  label,
  value,
  options,
  onChange,
  emptyLabel,
}: {
  label: string;
  value: string;
  options: { id: string; label: string }[];
  onChange: (id: string) => void;
  emptyLabel?: string;
}) {
  if (options.length === 0 && emptyLabel) {
    return <p className="mb-4 text-sm text-gray-500 bg-gray-50 p-3 rounded">{emptyLabel}</p>;
  }
  return (
    <div className="mb-4 max-w-sm">
      <label className="label" htmlFor={`parent-${label.toLowerCase().replace(/\s+/g, '-')}`}>
        {label}
      </label>
      <select
        id={`parent-${label.toLowerCase().replace(/\s+/g, '-')}`}
        className="input"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      >
        {options.length === 0 && <option value="">None yet</option>}
        {options.map((o) => (
          <option key={o.id} value={o.id}>
            {o.label}
          </option>
        ))}
      </select>
    </div>
  );
}

/** Replaces a row in place, or appends it when it is new. */
function upsert<T extends { id: string }>(rows: T[], item: T): T[] {
  const index = rows.findIndex((r) => r.id === item.id);
  if (index === -1) return [...rows, item];
  const copy = rows.slice();
  copy[index] = item;
  return copy;
}

/** Names the parent of a row so a strand says which area it belongs to. */
function parentLabelFor(
  tab: CurriculumKind,
  row: Record<string, string | undefined>,
  areas: LearningArea[],
  strands: Strand[]
): string {
  if (tab === 'strand') {
    return areas.find((a) => a.id === row.learning_area_id)?.name ?? '';
  }
  if (tab === 'sub-strand') {
    const strand = strands.find((s) => s.id === row.strand_id);
    const area = areas.find((a) => a.id === strand?.learning_area_id);
    return [area?.name, strand?.name].filter(Boolean).join(' › ');
  }
  return '';
}

function capitalise(word: string): string {
  return word.charAt(0).toUpperCase() + word.slice(1);
}
