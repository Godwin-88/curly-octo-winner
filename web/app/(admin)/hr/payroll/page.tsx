'use client';

import { useEffect, useState } from 'react';
import { Plus, Trash2, Eye } from 'lucide-react';
import { api, PayrollRun, StaffProfile } from '@/lib/api';

export default function PayrollPage() {
  const token = ''; // TODO: Get from auth context
  const [runs, setRuns] = useState<PayrollRun[]>([]);
  const [staff, setStaff] = useState<StaffProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [selected, setSelected] = useState<PayrollRun | null>(null);
  const [month, setMonth] = useState(1);
  const [year, setYear] = useState(2026);
  const [status, setStatus] = useState('');
  const [form, setForm] = useState({
    staff_id: '', month: 1, year: 2026, basic_salary_cents: 0, allowances_cents: 0,
    paye_cents: 0, nhif_cents: 0, nssf_cents: 0, other_deductions_cents: 0,
  });

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [runData, staffData] = await Promise.all([
        api.listPayrollRuns({ month, year, status }, token),
        api.listStaff({}, token),
      ]);
      setRuns(runData);
      setStaff(staffData);
    } catch (e: any) {
      setError(e.message || 'Failed to load payroll');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, month, year, status]);

  const create = async () => {
    setError('');
    try {
      await api.createPayrollRun(form, token);
      setShowForm(false);
      setForm({ staff_id: '', month: 1, year: 2026, basic_salary_cents: 0, allowances_cents: 0, paye_cents: 0, nhif_cents: 0, nssf_cents: 0, other_deductions_cents: 0 });
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to create payroll run');
    }
  };

  const remove = async (id: string) => {
    if (!confirm('Delete this payroll run?')) return;
    try {
      await api.deletePayrollRun(id, token);
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to delete payroll run');
    }
  };

  const view = async (id: string) => {
    try {
      const r = await api.getPayrollRun(id, token);
      setSelected(r);
    } catch (e: any) {
      setError(e.message || 'Failed to load payroll run');
    }
  };

  const statusColor = (s: string) => {
    if (s === 'paid') return 'bg-green-50 text-green-700';
    if (s === 'approved') return 'bg-blue-50 text-blue-700';
    return 'bg-yellow-50 text-yellow-700';
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Payroll</h1>
          <p className="text-gray-500">Monthly payroll runs with PAYE, NHIF & NSSF deductions</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 flex items-center gap-2">
          <Plus size={16} /> New Payroll Run
        </button>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {showForm && (
        <div className="mt-6 bg-white rounded-lg shadow border p-4">
          <h2 className="font-semibold mb-4">New Payroll Run</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <select value={form.staff_id} onChange={(e) => setForm({ ...form, staff_id: e.target.value })} className="border rounded-md px-3 py-2 text-sm">
              <option value="">Select staff...</option>
              {staff.map((s) => <option key={s.id} value={s.id}>{s.full_name}</option>)}
            </select>
            <select value={form.month} onChange={(e) => setForm({ ...form, month: Number(e.target.value) })} className="border rounded-md px-3 py-2 text-sm">
              {Array.from({ length: 12 }, (_, i) => i + 1).map((m) => (
                <option key={m} value={m}>{new Date(2026, m - 1).toLocaleString('default', { month: 'long' })}</option>
              ))}
            </select>
            <input type="number" value={form.year} onChange={(e) => setForm({ ...form, year: Number(e.target.value) })} className="border rounded-md px-3 py-2 text-sm" />
            <input type="number" placeholder="Basic salary (cents)" value={form.basic_salary_cents} onChange={(e) => setForm({ ...form, basic_salary_cents: Number(e.target.value) })} className="border rounded-md px-3 py-2 text-sm" />
            <input type="number" placeholder="Allowances (cents)" value={form.allowances_cents} onChange={(e) => setForm({ ...form, allowances_cents: Number(e.target.value) })} className="border rounded-md px-3 py-2 text-sm" />
            <input type="number" placeholder="PAYE (cents)" value={form.paye_cents} onChange={(e) => setForm({ ...form, paye_cents: Number(e.target.value) })} className="border rounded-md px-3 py-2 text-sm" />
            <input type="number" placeholder="NHIF (cents)" value={form.nhif_cents} onChange={(e) => setForm({ ...form, nhif_cents: Number(e.target.value) })} className="border rounded-md px-3 py-2 text-sm" />
            <input type="number" placeholder="NSSF (cents)" value={form.nssf_cents} onChange={(e) => setForm({ ...form, nssf_cents: Number(e.target.value) })} className="border rounded-md px-3 py-2 text-sm" />
            <input type="number" placeholder="Other deductions (cents)" value={form.other_deductions_cents} onChange={(e) => setForm({ ...form, other_deductions_cents: Number(e.target.value) })} className="border rounded-md px-3 py-2 text-sm" />
          </div>
          <div className="mt-4 flex gap-2">
            <button onClick={create} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700">Save</button>
            <button onClick={() => setShowForm(false)} className="bg-gray-100 text-gray-700 px-4 py-2 rounded-md text-sm hover:bg-gray-200">Cancel</button>
          </div>
        </div>
      )}

      <div className="mt-6 bg-white rounded-lg shadow border p-4 flex flex-wrap items-end gap-4">
        <div>
          <label className="block text-sm text-gray-500 mb-1">Month</label>
          <select value={month} onChange={(e) => setMonth(Number(e.target.value))} className="border rounded-md px-3 py-2 text-sm">
            {Array.from({ length: 12 }, (_, i) => i + 1).map((m) => (
              <option key={m} value={m}>{new Date(2026, m - 1).toLocaleString('default', { month: 'long' })}</option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm text-gray-500 mb-1">Year</label>
          <input type="number" value={year} onChange={(e) => setYear(Number(e.target.value))} className="border rounded-md px-3 py-2 text-sm w-24" />
        </div>
        <div>
          <label className="block text-sm text-gray-500 mb-1">Status</label>
          <select value={status} onChange={(e) => setStatus(e.target.value)} className="border rounded-md px-3 py-2 text-sm">
            <option value="">All</option>
            <option value="draft">Draft</option>
            <option value="approved">Approved</option>
            <option value="paid">Paid</option>
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
                <th className="px-4 py-3">Gross (KES)</th>
                <th className="px-4 py-3">Deductions (KES)</th>
                <th className="px-4 py-3">Net (KES)</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {runs.length === 0 ? (
                <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-400">No payroll runs found.</td></tr>
              ) : (
                runs.map((r) => (
                  <tr key={r.id} className="border-t">
                    <td className="px-4 py-3 font-medium">{r.staff_name}</td>
                    <td className="px-4 py-3">{new Date(r.year, r.month - 1).toLocaleString('default', { month: 'short' })} {r.year}</td>
                    <td className="px-4 py-3">{(r.gross_cents / 100).toLocaleString()}</td>
                    <td className="px-4 py-3">{((r.paye_cents + r.nhif_cents + r.nssf_cents + r.other_deductions_cents) / 100).toLocaleString()}</td>
                    <td className="px-4 py-3 font-medium">{(r.net_cents / 100).toLocaleString()}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded-full text-xs ${statusColor(r.status)}`}>{r.status}</span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button onClick={() => view(r.id)} className="text-blue-600 hover:text-blue-800" title="View"><Eye size={16} /></button>
                        <button onClick={() => remove(r.id)} className="text-red-600 hover:text-red-800" title="Delete"><Trash2 size={16} /></button>
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
          <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold">Payroll Run</h2>
                <p className="text-sm text-gray-500">{selected.staff_name} · {new Date(selected.year, selected.month - 1).toLocaleString('default', { month: 'long' })} {selected.year}</p>
              </div>
              <button onClick={() => setSelected(null)} className="text-gray-400 hover:text-gray-600 text-xl">×</button>
            </div>
            <div className="p-6 grid grid-cols-2 md:grid-cols-4 gap-4">
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Basic Salary</p>
                <p className="font-medium">KES {(selected.basic_salary_cents / 100).toLocaleString()}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Allowances</p>
                <p className="font-medium">KES {(selected.allowances_cents / 100).toLocaleString()}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Gross</p>
                <p className="font-medium">KES {(selected.gross_cents / 100).toLocaleString()}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Net</p>
                <p className="font-bold">KES {(selected.net_cents / 100).toLocaleString()}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">PAYE</p>
                <p className="font-medium">KES {(selected.paye_cents / 100).toLocaleString()}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">NHIF</p>
                <p className="font-medium">KES {(selected.nhif_cents / 100).toLocaleString()}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">NSSF</p>
                <p className="font-medium">KES {(selected.nssf_cents / 100).toLocaleString()}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Other Deductions</p>
                <p className="font-medium">KES {(selected.other_deductions_cents / 100).toLocaleString()}</p>
              </div>
            </div>
            {selected.items && selected.items.length > 0 && (
              <div className="p-6 pt-0">
                <h3 className="font-semibold mb-2">Line Items</h3>
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-left text-gray-500">
                    <tr>
                      <th className="px-3 py-2">Type</th>
                      <th className="px-3 py-2">Name</th>
                      <th className="px-3 py-2">Amount (KES)</th>
                    </tr>
                  </thead>
                  <tbody>
                    {selected.items.map((it) => (
                      <tr key={it.id} className="border-t">
                        <td className="px-3 py-2 capitalize">{it.item_type}</td>
                        <td className="px-3 py-2">{it.name}</td>
                        <td className="px-3 py-2">{(it.amount_cents / 100).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}