'use client';

// Report cards for a term.
//
// A card is a draft until it is published. A draft is built from the term's
// observations and can be rebuilt, commented on or deleted; a published card is
// what the parent sees, so nothing here changes it until a principal reopens it.

import { useCallback, useEffect, useMemo, useState } from 'react';
import { Download, Eye, FileCheck2, RefreshCw, Trash2, Undo2 } from 'lucide-react';

import { APIError, ClassReportCardsResult, Learner, ReportCard, api, asList } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';

const LEVELS: Record<number, string> = {
  1: 'bg-red-50 text-red-700',
  2: 'bg-yellow-50 text-yellow-800',
  3: 'bg-green-50 text-green-700',
  4: 'bg-blue-50 text-blue-700',
};

const message = (e: unknown, fallback: string) => (e instanceof APIError ? e.message : fallback);

/** Kenyan school terms: January–April, May–August, September–December. */
const currentTerm = () => Math.min(3, Math.floor(new Date().getMonth() / 4) + 1);

const COMMENT_HEADING = 'Class teacher';

export default function ReportCardsPage() {
  const { token, staff } = useAuth();
  const canReopen = staff?.role === 'principal' || staff?.role === 'super_admin';

  const [cards, setCards] = useState<ReportCard[]>([]);
  const [learners, setLearners] = useState<Learner[]>([]);
  const [selected, setSelected] = useState<ReportCard | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [term, setTerm] = useState(currentTerm);
  const [year, setYear] = useState(() => new Date().getFullYear());
  const [learnerId, setLearnerId] = useState('');
  const [grade, setGrade] = useState('');
  const [busy, setBusy] = useState('');
  const [comment, setComment] = useState('');
  const [confirm, setConfirm] = useState<{ kind: 'publish' | 'reopen' | 'delete'; card: ReportCard } | null>(null);

  const load = useCallback(async () => {
    if (!staff) return;
    setLoading(true);
    try {
      const [cardData, learnerData] = await Promise.all([
        api.listReportCards({ term, year }, token),
        api.listLearners({}, token),
      ]);
      setCards(asList(cardData));
      setLearners(asList(learnerData));
    } catch (e) {
      setError(message(e, 'Could not load the report cards.'));
    } finally {
      setLoading(false);
    }
  }, [staff, token, term, year]);

  useEffect(() => {
    load();
  }, [load]);

  const grades = useMemo(
    () => Array.from(new Set(learners.map((l) => l.grade).filter(Boolean))).sort((a, b) => a.localeCompare(b, undefined, { numeric: true })),
    [learners]
  );

  const run = async (label: string, work: () => Promise<void>) => {
    setBusy(label);
    setError('');
    setNotice('');
    try {
      await work();
    } catch (e) {
      setError(message(e, 'That did not work. Please try again.'));
    } finally {
      setBusy('');
    }
  };

  const open = (card: ReportCard) => {
    setSelected(card);
    setComment(card.teacher_comments?.[COMMENT_HEADING] ?? '');
  };

  const generateOne = () =>
    run('one', async () => {
      if (!learnerId) throw new APIError('Choose a learner first.', 'INVALID', 400);
      open(await api.generateReportCard({ learner_id: learnerId, term, year }, {}, token));
      await load();
    });

  const generateClass = () =>
    run('class', async () => {
      if (!grade) throw new APIError('Choose a grade first.', 'INVALID', 400);
      const r: ClassReportCardsResult = await api.generateClassReportCards({ grade, term, year }, token);
      const parts = [`${r.generated} draft report card${r.generated === 1 ? '' : 's'} made for ${grade}.`];
      if (r.skipped_published) parts.push(`${r.skipped_published} already published and left as they are.`);
      if (r.no_observations.length) {
        parts.push(`No observations this term for: ${r.no_observations.join(', ')}.`);
      }
      setNotice(parts.join(' '));
      await load();
    });

  const view = (id: string) => run('view', async () => open(await api.getReportCard(id, token)));

  const saveComment = () =>
    run('comment', async () => {
      if (!selected) return;
      open(await api.updateReportCard(selected.id, {
        teacher_comments: { ...selected.teacher_comments, [COMMENT_HEADING]: comment.trim() },
      }, token));
      setNotice('Comment saved.');
    });

  const download = (card: ReportCard) =>
    run('pdf', async () => {
      const { blob, name } = await api.downloadReportCardPDF(card.id, token);
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = name;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(url);
    });

  const confirmed = () => {
    if (!confirm) return;
    const { kind, card } = confirm;
    return run(kind, async () => {
      if (kind === 'delete') {
        await api.deleteReportCard(card.id, token);
        setSelected(null);
        setNotice('Draft deleted.');
      } else {
        const updated = kind === 'publish'
          ? await api.publishReportCard(card.id, token)
          : await api.reopenReportCard(card.id, token);
        if (selected?.id === card.id) open(updated);
        setNotice(kind === 'publish' ? 'Report card published. Parents can now see it.' : 'Report card reopened as a draft.');
      }
      setConfirm(null);
      await load();
    });
  };

  const status = (c: ReportCard) => (
    <span className={`px-2 py-1 rounded-full text-xs ${c.status === 'final' ? 'bg-green-50 text-green-700' : 'bg-yellow-50 text-yellow-800'}`}>
      {c.status === 'final' ? 'Published' : 'Draft'}
    </span>
  );

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold">Report cards</h1>
      <p className="text-gray-600">
        Drafts are built from the term&apos;s observations. Publish a card when it is ready for the parent.
      </p>

      {error && <div role="alert" className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}
      {notice && <div role="status" className="mt-4 p-3 bg-green-50 text-green-800 rounded-md">{notice}</div>}

      <div className="mt-6 card p-4 flex flex-wrap items-end gap-4">
        <div>
          <label htmlFor="rc-term" className="block text-sm text-gray-600 mb-1">Term</label>
          <select id="rc-term" value={term} onChange={(e) => setTerm(Number(e.target.value))} className="input">
            <option value={1}>Term 1</option>
            <option value={2}>Term 2</option>
            <option value={3}>Term 3</option>
          </select>
        </div>
        <div>
          <label htmlFor="rc-year" className="block text-sm text-gray-600 mb-1">Year</label>
          <input id="rc-year" type="number" value={year} onChange={(e) => setYear(Number(e.target.value))} className="input w-24" />
        </div>
        <div className="flex items-end gap-2">
          <div>
            <label htmlFor="rc-grade" className="block text-sm text-gray-600 mb-1">Whole grade</label>
            <select id="rc-grade" value={grade} onChange={(e) => setGrade(e.target.value)} className="input">
              <option value="">Choose a grade…</option>
              {grades.map((g) => <option key={g} value={g}>{g}</option>)}
            </select>
          </div>
          <button type="button" onClick={generateClass} disabled={busy !== ''} className="btn-primary flex items-center gap-2">
            <RefreshCw size={16} className={busy === 'class' ? 'animate-spin' : ''} aria-hidden="true" />
            Make drafts for the grade
          </button>
        </div>
        <div className="flex items-end gap-2 flex-1 min-w-[260px]">
          <div className="flex-1">
            <label htmlFor="rc-learner" className="block text-sm text-gray-600 mb-1">One learner</label>
            <select id="rc-learner" value={learnerId} onChange={(e) => setLearnerId(e.target.value)} className="input w-full">
              <option value="">Choose a learner…</option>
              {learners.map((l) => (
                <option key={l.id} value={l.id}>{l.full_name} — {[l.grade, l.stream].filter(Boolean).join(' ')}</option>
              ))}
            </select>
          </div>
          <button type="button" onClick={generateOne} disabled={busy !== ''} className="btn-secondary">
            Make draft
          </button>
        </div>
      </div>

      {loading ? (
        <p className="text-gray-600 mt-4" aria-busy="true">Loading…</p>
      ) : (
        <div className="mt-6 card overflow-x-auto">
          <table className="w-full text-sm">
            <caption className="sr-only">Report cards for Term {term} {year}</caption>
            <thead className="bg-gray-50 text-left text-gray-600">
              <tr>
                <th scope="col" className="px-4 py-3">Learner</th>
                <th scope="col" className="px-4 py-3">Class</th>
                <th scope="col" className="px-4 py-3">Status</th>
                <th scope="col" className="px-4 py-3">Overall</th>
                <th scope="col" className="px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {cards.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-gray-600">
                    No report cards for Term {term} {year} yet. Record observations, then make drafts for a grade.
                  </td>
                </tr>
              ) : (
                cards.map((c) => (
                  <tr key={c.id} className="border-t">
                    <td className="px-4 py-3 font-medium">{c.learner_name}</td>
                    <td className="px-4 py-3">{[c.grade, c.stream].filter(Boolean).join(' ')}</td>
                    <td className="px-4 py-3">{status(c)}</td>
                    <td className="px-4 py-3">
                      {c.overall_rating ? (
                        <span className={`px-2 py-1 rounded-full text-xs ${LEVELS[c.overall_rating]}`}>
                          {c.overall_rating} · {c.overall_label}
                        </span>
                      ) : '—'}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-3">
                        <button type="button" onClick={() => view(c.id)} className="text-blue-700 hover:text-blue-900" aria-label={`View ${c.learner_name}'s report card`}><Eye size={16} aria-hidden="true" /></button>
                        <button type="button" onClick={() => download(c)} className="text-gray-700 hover:text-gray-900" aria-label={`Download ${c.learner_name}'s report card as PDF`}><Download size={16} aria-hidden="true" /></button>
                        {c.status === 'draft' && (
                          <button type="button" onClick={() => setConfirm({ kind: 'delete', card: c })} className="text-red-700 hover:text-red-900" aria-label={`Delete ${c.learner_name}'s draft`}><Trash2 size={16} aria-hidden="true" /></button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {selected && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-40 p-4" role="dialog" aria-modal="true" aria-labelledby="rc-title">
          <div className="bg-white rounded-lg shadow-xl max-w-3xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b flex items-start justify-between gap-4">
              <div>
                <h2 id="rc-title" className="text-xl font-bold">{selected.learner_name}</h2>
                <p className="text-sm text-gray-600">
                  {[selected.grade, selected.stream].filter(Boolean).join(' ')} · Term {selected.term} {selected.year} · {status(selected)}
                </p>
              </div>
              <button type="button" onClick={() => setSelected(null)} className="text-gray-600 hover:text-gray-900 text-xl" aria-label="Close">×</button>
            </div>

            <div className="p-6 space-y-6">
              <div className="flex flex-wrap gap-2">
                <button type="button" className="btn-secondary flex items-center gap-2" onClick={() => download(selected)} disabled={busy !== ''}>
                  <Download size={16} aria-hidden="true" /> Download PDF
                </button>
                {selected.status === 'draft' ? (
                  <button type="button" className="btn-primary flex items-center gap-2" onClick={() => setConfirm({ kind: 'publish', card: selected })} disabled={busy !== ''}>
                    <FileCheck2 size={16} aria-hidden="true" /> Publish
                  </button>
                ) : canReopen ? (
                  <button type="button" className="btn-secondary flex items-center gap-2" onClick={() => setConfirm({ kind: 'reopen', card: selected })} disabled={busy !== ''}>
                    <Undo2 size={16} aria-hidden="true" /> Reopen as draft
                  </button>
                ) : (
                  <p className="text-sm text-gray-600 self-center">Published. A principal can reopen it if it needs to change.</p>
                )}
              </div>

              <dl className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                <div className="bg-gray-50 rounded-lg p-3">
                  <dt className="text-xs text-gray-600">UPI</dt>
                  <dd className="font-medium">{selected.upi || '—'}</dd>
                </div>
                <div className="bg-gray-50 rounded-lg p-3">
                  <dt className="text-xs text-gray-600">Overall</dt>
                  <dd className="font-medium">{selected.overall_rating ? `${selected.overall_rating} · ${selected.overall_label}` : '—'}</dd>
                </div>
                <div className="bg-gray-50 rounded-lg p-3">
                  <dt className="text-xs text-gray-600">Attendance</dt>
                  <dd className="font-medium">
                    {selected.attendance_summary?.total_days
                      ? `${Number(selected.attendance_summary.attendance_rate || 0).toFixed(0)}% of ${selected.attendance_summary.total_days} days`
                      : 'No register marked'}
                  </dd>
                </div>
                <div className="bg-gray-50 rounded-lg p-3">
                  <dt className="text-xs text-gray-600">{selected.published_at ? 'Published' : 'Drafted'}</dt>
                  <dd className="font-medium">{new Date(selected.published_at || selected.generated_at).toLocaleDateString('en-KE')}</dd>
                </div>
              </dl>

              <div>
                <h3 className="font-semibold mb-2">Learning areas</h3>
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-left text-gray-600">
                    <tr>
                      <th scope="col" className="px-3 py-2">Learning area</th>
                      <th scope="col" className="px-3 py-2">Strand</th>
                      <th scope="col" className="px-3 py-2">Sub-strand</th>
                      <th scope="col" className="px-3 py-2">Level</th>
                      <th scope="col" className="px-3 py-2">Note</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(selected.items ?? []).map((item) => (
                      <tr key={item.id} className="border-t">
                        <td className="px-3 py-2">{item.learning_area}</td>
                        <td className="px-3 py-2">{item.strand_name}</td>
                        <td className="px-3 py-2">{item.sub_strand_name}</td>
                        <td className="px-3 py-2">
                          {item.rubric_level ? (
                            <span className={`px-2 py-1 rounded-full text-xs ${LEVELS[item.rubric_level]}`}>
                              {item.rubric_level} · {item.rubric_label}
                            </span>
                          ) : '—'}
                        </td>
                        <td className="px-3 py-2 text-gray-700">{item.comment || '—'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div>
                <label htmlFor="rc-comment" className="font-semibold block mb-2">Class teacher&apos;s comment</label>
                {selected.status === 'draft' ? (
                  <>
                    <textarea
                      id="rc-comment"
                      className="input w-full"
                      rows={3}
                      maxLength={1000}
                      value={comment}
                      onChange={(e) => setComment(e.target.value)}
                    />
                    <button type="button" className="btn-secondary mt-2" onClick={saveComment} disabled={busy !== ''}>
                      Save comment
                    </button>
                  </>
                ) : (
                  <p id="rc-comment" className="text-sm text-gray-800">{selected.teacher_comments?.[COMMENT_HEADING] || 'None written.'}</p>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {confirm && (
        <ConfirmDialog
          title={
            confirm.kind === 'publish' ? 'Publish this report card?' :
            confirm.kind === 'reopen' ? 'Reopen this report card?' : 'Delete this draft?'
          }
          message={
            confirm.kind === 'publish'
              ? `${confirm.card.learner_name}'s parent will be able to see it, and it can no longer be edited, rebuilt or deleted.`
              : confirm.kind === 'reopen'
                ? `${confirm.card.learner_name}'s report card becomes a draft again and parents stop seeing it until it is published.`
                : `${confirm.card.learner_name}'s draft for Term ${confirm.card.term} ${confirm.card.year} will be removed. The observations it was built from are kept.`
          }
          confirmLabel={confirm.kind === 'publish' ? 'Publish' : confirm.kind === 'reopen' ? 'Reopen' : 'Delete'}
          busy={busy !== ''}
          onConfirm={confirmed}
          onCancel={() => setConfirm(null)}
        />
      )}
    </div>
  );
}
