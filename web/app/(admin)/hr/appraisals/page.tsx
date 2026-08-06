'use client';

import { useEffect, useState } from 'react';
import { Plus, Trash2, Eye } from 'lucide-react';
import { api, StaffAppraisal, StaffProfile } from '@/lib/api';

export default function AppraisalsPage() {
  const token = ''; // TODO: Get from auth context
  const [appraisals, setAppraisals] = useState<StaffAppraisal[]>([]);
  const [staff, setStaff] = useState<StaffProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [selected, setSelected] = useState<StaffAppraisal | null>(null);
  const [year, setYear] = useState(2026);
  const [term, setTerm] = useState(0);
  const [form, setForm] = useState({
    staff_id: '', year: 2026, term: 1, comments: '',
  });

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [appraisalData, staffData] = await Promise.all([
        api.listAppraisals({ year, term: term || undefined }, token),
        api.listStaff({}, token),
      ]);
      setAppraisals(appraisalData);
      setStaff(staffData);
    } catch (e: any) {
      setError(e.message || 'Failed to load appraisals');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, year, term]);

  const create = async () => {
    setError('');
    try {
      await api.createAppraisal(form, token);
      setShowForm(false);
      setForm({ staff_id: '', year: 2026, term: 1, comments: '' });
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to create appraisal');
    }
  };

  const remove = async (id: string) => {
    if (!confirm('Delete this appraisal?')) return;
    try {
      await api.deleteAppraisal(id, token);
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to delete appraisal');
    }
  };

  const view = async (id: string) => {
    try {
      const a = await api.getAppraisal(id, token);
      setSelected(a);
    } catch (e: any) {
      setError(e.message || 'Failed to load appraisal');
    }
  };

  const statusColor = (s: string) => {
    if (s === 'approved') return 'bg-green-50 text-green-700';
    if (s === 'submitted') return 'bg-blue-50 text-blue-700';
    return 'bg-yellow-50 text-yellow-700';
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Staff Appraisals</h1>
          <p className="text-gray-500">TSC-aligned performance appraisal forms</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 flex items-center gap-2">
          <Plus size={16} /> New Appraisal
        </button>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {showForm && (
        <div className="mt-6 bg-white rounded-lg shadow border p-4">
          <h2 className="font-semibold mb-4">New Appraisal</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <select value={form.staff_id} onChange={(e) => setForm({ ...form, staff_id: e.target.value })} className="border rounded-md px-3 py-2 text-sm">
              <option value="">Select staff...</option>
              {staff.map((s) => <option key={s.id} value={s.id}>{s.full_name}</option>)}
            </select>
            <input type="number" value={form.year} onChange={(e) => setForm({ ...form, year: Number(e.target.value) })} className="border rounded-md px-3 py-2 text-sm" />
            <select value={form.term} onChange={(e) => setForm({ ...form, term: Number(e.target.value) })} className="border rounded-md px-3 py-2 text-sm">
              <option value={1}>Term 1</option>
              <option value={2}>Term 2</option>
              <option value={3}>Term 3</option>
            </select>
            <input placeholder="Comments" value={form.comments} onChange={(e) => setForm({ ...form, comments: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
          </div>
          <div className="mt-4 flex gap-2">
            <button onClick={create} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700">Save</button>
            <button onClick={() => setShowForm(false)} className="bg-gray-100 text-gray-700 px-4 py-2 rounded-md text-sm hover:bg-gray-200">Cancel</button>
          </div>
        </div>
      )}

      <div className="mt-6 bg-white rounded-lg shadow border p-4 flex flex-wrap items-end gap-4">
        <div>
          <label className="block text-sm text-gray-500 mb-1">Year</label>
          <input type="number" value={year} onChange={(e) => setYear(Number(e.target.value))} className="border rounded-md px-3 py-2 text-sm w-24" />
        </div>
        <div>
          <label className="block text-sm text-gray-500 mb-1">Term</label>
          <select value={term} onChange={(e) => setTerm(Number(e.target.value))} className="border rounded-md px-3 py-2 text-sm">
            <option value={0}>All terms</option>
            <option value={1}>Term 1</option>
            <option value={2}>Term 2</option>
            <option value={3}>Term 3</option>
          </select>
        </div>
      </div>

      {loading ? (
        <p className="text-gray-500 mt-4">Loading...</p>
      ) : (
        <div className="mt-6 bg-white rounded-lg shadow border overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-left text-gray-500">
              <tr>
                <th className="px-4 py-3">Staff</th>
                <th className="px-4 py-3">Period</th>
                <th className="px-4 py-3">Overall Score</th>
                <th className="px-4 py-3">Rating</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {appraisals.length === 0 ? (
                <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-400">No appraisals found.</td></tr>
              ) : (
                appraisals.map((a) => (
                  <tr key={a.id} className="border-t">
                    <td className="px-4 py-3 font-medium">{a.staff_name}</td>
                    <td className="px-4 py-3">{a.term ? `Term ${a.term}` : 'Annual'} {a.year}</td>
                    <td className="px-4 py-3">{a.overall_score ?? '—'}</td>
                    <td className="px-4 py-3">{a.rating || '—'}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded-full text-xs ${statusColor(a.status)}`}>{a.status}</span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button onClick={() => view(a.id)} className="text-blue-600 hover:text-blue-800" title="View"><Eye size={16} /></button>
                        <button onClick={() => remove(a.id)} className="text-red-600 hover:text-red-800" title="Delete"><Trash2 size={16} /></button>
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
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-lg w-full">
            <div className="p-6 border-b flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold">Appraisal</h2>
                <p className="text-sm text-gray-500">{selected.staff_name} · {selected.term ? `Term ${selected.term}` : 'Annual'} {selected.year}</p>
              </div>
              <button onClick={() => setSelected(null)} className="text-gray-400 hover:text-gray-600 text-xl">×</button>
            </div>
            <div className="p-6 grid grid-cols-2 gap-4">
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Overall Score</p>
                <p className="font-medium">{selected.overall_score ?? '—'}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Rating</p>
                <p className="font-medium">{selected.rating || '—'}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Status</p>
                <p className="font-medium capitalize">{selected.status}</p>
              </div>
              {selected.comments && (
                <div className="bg-gray-50 rounded-lg p-3 col-span-2">
                  <p className="text-xs text-gray-500">Comments</p>
                  <p className="font-medium">{selected.comments}</p>
                </div>
              )}
              {selected.scores && Object.keys(selected.scores).length > 0 && (
                <div className="col-span-2">
                  <h3 className="font-semibold mb-2">Scores</h3>
                  <div className="space-y-2">
                    {Object.entries(selected.scores).map(([key, val]) => (
                      <div key={key} className="bg-gray-50 rounded-lg p-3 flex justify-between">
                        <span className="text-sm">{key}</span>
                        <span className="font-medium">{String(val)}</span>
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