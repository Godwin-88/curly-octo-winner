'use client';

import { useEffect, useState } from 'react';
import { Plus, Trash2, Wallet } from 'lucide-react';
import { api, FeeStructure, FeeItemInput } from '@/lib/api';

const ITEM_TYPES = ['tuition', 'caution', 'transport', 'activity', 'boarding', 'other'];

export default function FeesPage() {
  const token = ''; // TODO: Get from auth context
  const [structures, setStructures] = useState<FeeStructure[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    name: '',
    grade: 'Grade 4',
    term: 1,
    year: new Date().getFullYear(),
    notes: '',
  });
  const [items, setItems] = useState<FeeItemInput[]>([
    { name: 'Tuition', amount_cents: 0, item_type: 'tuition' },
  ]);

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const data = await api.listFeeStructures({}, token);
      setStructures(data);
    } catch (e: any) {
      setError(e.message || 'Failed to load fee structures');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await api.createFeeStructure({
        name: form.name,
        grade: form.grade,
        term: form.term,
        year: form.year,
        notes: form.notes || undefined,
        items: items.filter((i) => i.name && i.amount_cents > 0),
      }, token);
      setShowForm(false);
      setForm({ name: '', grade: 'Grade 4', term: 1, year: new Date().getFullYear(), notes: '' });
      setItems([{ name: 'Tuition', amount_cents: 0, item_type: 'tuition' }]);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to create fee structure');
    }
  };

  const handleDelete = async (id: string) => {
    setError('');
    try {
      await api.deleteFeeStructure(id, token);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to delete fee structure');
    }
  };

  const updateItem = (idx: number, field: keyof FeeItemInput, value: any) => {
    const next = [...items];
    next[idx] = { ...next[idx], [field]: value };
    setItems(next);
  };

  const total = items.reduce((sum, i) => sum + (i.amount_cents || 0), 0);

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Fee Structures</h1>
          <p className="text-gray-500">Per-grade fee schedules for each term</p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
        >
          <Plus size={18} /> New Fee Structure
        </button>
      </div>

      {error && <div className="mb-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {showForm && (
        <form onSubmit={handleCreate} className="mb-6 p-4 bg-white rounded-lg shadow border">
          <div className="grid grid-cols-2 gap-4 mb-4">
            <div>
              <label className="block text-sm font-medium mb-1">Name</label>
              <input
                required
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="w-full border rounded-md px-3 py-2"
                placeholder="e.g. Grade 4 Term 1 Fees"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Grade</label>
              <select
                value={form.grade}
                onChange={(e) => setForm({ ...form, grade: e.target.value })}
                className="w-full border rounded-md px-3 py-2"
              >
                {['Grade 4', 'Grade 5', 'Grade 6'].map((g) => (
                  <option key={g} value={g}>{g}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Term</label>
              <select
                value={form.term}
                onChange={(e) => setForm({ ...form, term: Number(e.target.value) })}
                className="w-full border rounded-md px-3 py-2"
              >
                {[1, 2, 3].map((t) => (
                  <option key={t} value={t}>Term {t}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Year</label>
              <input
                type="number"
                value={form.year}
                onChange={(e) => setForm({ ...form, year: Number(e.target.value) })}
                className="w-full border rounded-md px-3 py-2"
              />
            </div>
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-2">Fee Items</label>
            {items.map((item, idx) => (
              <div key={idx} className="flex gap-2 mb-2">
                <input
                  value={item.name}
                  onChange={(e) => updateItem(idx, 'name', e.target.value)}
                  className="flex-1 border rounded-md px-3 py-2"
                  placeholder="Item name"
                />
                <select
                  value={item.item_type}
                  onChange={(e) => updateItem(idx, 'item_type', e.target.value)}
                  className="border rounded-md px-3 py-2"
                >
                  {ITEM_TYPES.map((t) => (
                    <option key={t} value={t}>{t}</option>
                  ))}
                </select>
                <input
                  type="number"
                  value={item.amount_cents}
                  onChange={(e) => updateItem(idx, 'amount_cents', Number(e.target.value))}
                  className="w-32 border rounded-md px-3 py-2"
                  placeholder="Amount (KES)"
                />
                <button
                  type="button"
                  onClick={() => setItems(items.filter((_, i) => i !== idx))}
                  className="text-red-500 hover:text-red-700"
                >
                  <Trash2 size={18} />
                </button>
              </div>
            ))}
            <button
              type="button"
              onClick={() => setItems([...items, { name: '', amount_cents: 0, item_type: 'other' }])}
              className="text-blue-600 text-sm hover:underline"
            >
              + Add item
            </button>
          </div>

          <div className="flex items-center justify-between">
            <p className="text-sm text-gray-600">
              Total: <span className="font-semibold">KES {total.toLocaleString()}</span>
            </p>
            <button type="submit" className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">
              Create
            </button>
          </div>
        </form>
      )}

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : structures.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          <Wallet size={48} className="mx-auto mb-3 text-gray-300" />
          <p>No fee structures yet. Create one to get started.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {structures.map((s) => (
            <div key={s.id} className="bg-white rounded-lg shadow border p-4">
              <div className="flex items-start justify-between mb-2">
                <div>
                  <h3 className="font-semibold">{s.name}</h3>
                  <p className="text-sm text-gray-500">
                    {s.grade} · Term {s.term} · {s.year}
                  </p>
                </div>
                <button
                  onClick={() => handleDelete(s.id)}
                  className="text-red-500 hover:text-red-700"
                >
                  <Trash2 size={16} />
                </button>
              </div>
              <div className="text-2xl font-bold mb-3">
                KES {(s.total_cents / 100).toLocaleString()}
              </div>
              <div className="space-y-1">
                {s.items?.map((item) => (
                  <div key={item.id} className="flex justify-between text-sm">
                    <span className="text-gray-600">{item.name}</span>
                    <span>KES {(item.amount_cents / 100).toLocaleString()}</span>
                  </div>
                ))}
              </div>
              <div className="mt-3 pt-3 border-t text-xs">
                <span className={`px-2 py-1 rounded-full ${s.active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                  {s.active ? 'Active' : 'Inactive'}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}