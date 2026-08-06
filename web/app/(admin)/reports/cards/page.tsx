'use client';

import { useEffect, useState } from 'react';
import { RefreshCw, Trash2, Eye } from 'lucide-react';
import { api, ReportCard, Learner } from '@/lib/api';

export default function ReportCardsPage() {
  const token = ''; // TODO: Get from auth context
  const [cards, setCards] = useState<ReportCard[]>([]);
  const [learners, setLearners] = useState<Learner[]>([]);
  const [selected, setSelected] = useState<ReportCard | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [term, setTerm] = useState(1);
  const [year, setYear] = useState(2026);
  const [learnerId, setLearnerId] = useState('');
  const [generating, setGenerating] = useState(false);

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [cardData, learnerData] = await Promise.all([
        api.listReportCards({ term, year }, token),
        api.listLearners({}, token),
      ]);
      setCards(cardData);
      setLearners(learnerData);
    } catch (e: any) {
      setError(e.message || 'Failed to load report cards');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, term, year]);

  const generate = async () => {
    if (!learnerId) {
      setError('Select a learner to generate a report card');
      return;
    }
    setGenerating(true);
    setError('');
    try {
      const card = await api.generateReportCard({ learner_id: learnerId, term, year }, {}, token);
      setSelected(card);
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to generate report card');
    } finally {
      setGenerating(false);
    }
  };

  const viewCard = async (id: string) => {
    try {
      const card = await api.getReportCard(id, token);
      setSelected(card);
    } catch (e: any) {
      setError(e.message || 'Failed to load report card');
    }
  };

  const removeCard = async (id: string) => {
    if (!confirm('Delete this report card?')) return;
    try {
      await api.deleteReportCard(id, token);
      setSelected(null);
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to delete report card');
    }
  };

  const rubricColor = (level?: number) => {
    if (!level) return 'bg-gray-100 text-gray-600';
    if (level === 1) return 'bg-red-50 text-red-700';
    if (level === 2) return 'bg-yellow-50 text-yellow-700';
    if (level === 3) return 'bg-green-50 text-green-700';
    return 'bg-blue-50 text-blue-700';
  };

  return (
    <div className="p-6">
      <div>
        <h1 className="text-2xl font-bold">Report Cards</h1>
        <p className="text-gray-500">CBC-compliant term report cards</p>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {/* Filters */}
      <div className="mt-6 bg-white rounded-lg shadow border p-4 flex flex-wrap items-end gap-4">
        <div>
          <label className="block text-sm text-gray-500 mb-1">Term</label>
          <select value={term} onChange={(e) => setTerm(Number(e.target.value))} className="border rounded-md px-3 py-2 text-sm">
            <option value={1}>Term 1</option>
            <option value={2}>Term 2</option>
            <option value={3}>Term 3</option>
          </select>
        </div>
        <div>
          <label className="block text-sm text-gray-500 mb-1">Year</label>
          <input type="number" value={year} onChange={(e) => setYear(Number(e.target.value))} className="border rounded-md px-3 py-2 text-sm w-24" />
        </div>
        <div className="flex-1 min-w-[200px]">
          <label className="block text-sm text-gray-500 mb-1">Learner</label>
          <select value={learnerId} onChange={(e) => setLearnerId(e.target.value)} className="border rounded-md px-3 py-2 text-sm w-full">
            <option value="">Select learner...</option>
            {learners.map((l) => (
              <option key={l.id} value={l.id}>{l.full_name} — {l.grade} {l.stream}</option>
            ))}
          </select>
        </div>
        <button
          onClick={generate}
          disabled={generating}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"
        >
          <RefreshCw size={16} className={generating ? 'animate-spin' : ''} />
          Generate
        </button>
      </div>

      {loading ? (
        <p className="text-gray-500 mt-4">Loading...</p>
      ) : (
        <div className="mt-6 bg-white rounded-lg shadow border overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-left text-gray-500">
              <tr>
                <th className="px-4 py-3">Learner</th>
                <th className="px-4 py-3">Grade</th>
                <th className="px-4 py-3">Stream</th>
                <th className="px-4 py-3">Term</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Overall</th>
                <th className="px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {cards.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-4 py-8 text-center text-gray-400">
                    No report cards generated yet. Select a learner and click Generate.
                  </td>
                </tr>
              ) : (
                cards.map((c) => (
                  <tr key={c.id} className="border-t">
                    <td className="px-4 py-3 font-medium">{c.learner_name}</td>
                    <td className="px-4 py-3">{c.grade}</td>
                    <td className="px-4 py-3">{c.stream}</td>
                    <td className="px-4 py-3">Term {c.term} {c.year}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded-full text-xs ${c.status === 'final' ? 'bg-green-50 text-green-700' : 'bg-yellow-50 text-yellow-700'}`}>
                        {c.status}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      {c.overall_rating ? (
                        <span className={`px-2 py-1 rounded-full text-xs ${rubricColor(c.overall_rating)}`}>{c.overall_rating}</span>
                      ) : '—'}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button onClick={() => viewCard(c.id)} className="text-blue-600 hover:text-blue-800" title="View"><Eye size={16} /></button>
                        <button onClick={() => removeCard(c.id)} className="text-red-600 hover:text-red-800" title="Delete"><Trash2 size={16} /></button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* Detail modal */}
      {selected && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-3xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold">Report Card</h2>
                <p className="text-sm text-gray-500">
                  {selected.learner_name} — {selected.grade} {selected.stream} · Term {selected.term} {selected.year}
                </p>
              </div>
              <button onClick={() => setSelected(null)} className="text-gray-400 hover:text-gray-600 text-xl">×</button>
            </div>

            <div className="p-6 space-y-6">
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="bg-gray-50 rounded-lg p-3">
                  <p className="text-xs text-gray-500">UPI</p>
                  <p className="font-medium">{selected.upi || '—'}</p>
                </div>
                <div className="bg-gray-50 rounded-lg p-3">
                  <p className="text-xs text-gray-500">Status</p>
                  <p className="font-medium capitalize">{selected.status}</p>
                </div>
                <div className="bg-gray-50 rounded-lg p-3">
                  <p className="text-xs text-gray-500">Overall Rating</p>
                  <p className="font-medium">{selected.overall_rating || '—'}</p>
                </div>
                <div className="bg-gray-50 rounded-lg p-3">
                  <p className="text-xs text-gray-500">Generated</p>
                  <p className="font-medium">{new Date(selected.generated_at).toLocaleDateString()}</p>
                </div>
              </div>

              {selected.attendance_summary && (
                <div>
                  <h3 className="font-semibold mb-2">Attendance</h3>
                  <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
                    <div className="bg-gray-50 rounded-lg p-3 text-center">
                      <p className="text-xs text-gray-500">Rate</p>
                      <p className="font-bold">{Number(selected.attendance_summary.attendance_rate || 0).toFixed(1)}%</p>
                    </div>
                    <div className="bg-gray-50 rounded-lg p-3 text-center">
                      <p className="text-xs text-gray-500">Present</p>
                      <p className="font-bold">{selected.attendance_summary.present_days || 0}</p>
                    </div>
                    <div className="bg-gray-50 rounded-lg p-3 text-center">
                      <p className="text-xs text-gray-500">Absent</p>
                      <p className="font-bold">{selected.attendance_summary.absent_days || 0}</p>
                    </div>
                    <div className="bg-gray-50 rounded-lg p-3 text-center">
                      <p className="text-xs text-gray-500">Late</p>
                      <p className="font-bold">{selected.attendance_summary.late_days || 0}</p>
                    </div>
                    <div className="bg-gray-50 rounded-lg p-3 text-center">
                      <p className="text-xs text-gray-500">Excused</p>
                      <p className="font-bold">{selected.attendance_summary.excused_days || 0}</p>
                    </div>
                  </div>
                </div>
              )}

              <div>
                <h3 className="font-semibold mb-2">Learning Areas & Strands</h3>
                {selected.items && selected.items.length > 0 ? (
                  <table className="w-full text-sm">
                    <thead className="bg-gray-50 text-left text-gray-500">
                      <tr>
                        <th className="px-3 py-2">Learning Area</th>
                        <th className="px-3 py-2">Strand</th>
                        <th className="px-3 py-2">Sub-Strand</th>
                        <th className="px-3 py-2">Rating</th>
                        <th className="px-3 py-2">Comment</th>
                      </tr>
                    </thead>
                    <tbody>
                      {selected.items.map((item) => (
                        <tr key={item.id} className="border-t">
                          <td className="px-3 py-2">{item.learning_area}</td>
                          <td className="px-3 py-2">{item.strand_name}</td>
                          <td className="px-3 py-2">{item.sub_strand_name}</td>
                          <td className="px-3 py-2">
                            {item.rubric_level ? (
                              <span className={`px-2 py-1 rounded-full text-xs ${rubricColor(item.rubric_level)}`}>
                                {item.rubric_level} · {item.rubric_label}
                              </span>
                            ) : '—'}
                          </td>
                          <td className="px-3 py-2 text-gray-600">{item.comment || '—'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                ) : (
                  <p className="text-gray-400 text-sm">No assessment items yet.</p>
                )}
              </div>

              {selected.core_competency_remarks && Object.keys(selected.core_competency_remarks).length > 0 && (
                <div>
                  <h3 className="font-semibold mb-2">Core Competency Remarks</h3>
                  <div className="space-y-2">
                    {Object.entries(selected.core_competency_remarks).map(([key, val]) => (
                      <div key={key} className="bg-gray-50 rounded-lg p-3">
                        <p className="text-xs text-gray-500 font-medium">{key}</p>
                        <p className="text-sm">{val}</p>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {selected.teacher_comments && Object.keys(selected.teacher_comments).length > 0 && (
                <div>
                  <h3 className="font-semibold mb-2">Teacher Comments</h3>
                  <div className="space-y-2">
                    {Object.entries(selected.teacher_comments).map(([key, val]) => (
                      <div key={key} className="bg-gray-50 rounded-lg p-3">
                        <p className="text-xs text-gray-500 font-medium">{key}</p>
                        <p className="text-sm">{val}</p>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}