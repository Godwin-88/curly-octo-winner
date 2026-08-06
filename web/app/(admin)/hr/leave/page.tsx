'use client';

import { useEffect, useState } from 'react';
import { Plus, Check, X, Eye } from 'lucide-react';
import { api, LeaveRequest, StaffProfile } from '@/lib/api';

export default function LeavePage() {
  const token = ''; // TODO: Get from auth context
  const [leaves, setLeaves] = useState<LeaveRequest[]>([]);
  const [staff, setStaff] = useState<StaffProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [selected, setSelected] = useState<LeaveRequest | null>(null);
  const [status, setStatus] = useState('');
  const [form, setForm] = useState({
    staff_id: '', leave_type: 'annual', start_date: '', end_date: '', reason: '',
  });

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [leaveData, staffData] = await Promise.all([
        api.listLeaveRequests({ status }, token),
        api.listStaff({}, token),
      ]);
      setLeaves(leaveData);
      setStaff(staffData);
    } catch (e: any) {
      setError(e.message || 'Failed to load leave requests');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, status]);

  const create = async () => {
    setError('');
    try {
      await api.createLeaveRequest(form, token);
      setShowForm(false);
      setForm({ staff_id: '', leave_type: 'annual', start_date: '', end_date: '', reason: '' });
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to create leave request');
    }
  };

  const approve = async (id: string) => {
    try {
      await api.approveLeaveRequest(id, {}, token);
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to approve leave');
    }
  };

  const deny = async (id: string) => {
    const reason = prompt('Reason for denial:');
    if (reason === null) return;
    try {
      await api.denyLeaveRequest(id, { denial_reason: reason }, token);
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to deny leave');
    }
  };

  const view = async (id: string) => {
    try {
      const l = await api.getLeaveRequest(id, token);
      setSelected(l);
    } catch (e: any) {
      setError(e.message || 'Failed to load leave request');
    }
  };

  const statusColor = (s: string) => {
    if (s === 'approved') return 'bg-green-50 text-green-700';
    if (s === 'pending') return 'bg-yellow-50 text-yellow-700';
    if (s === 'denied') return 'bg-red-50 text-red-700';
    return 'bg-gray-100 text-gray-600';
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Leave Management</h1>
          <p className="text-gray-500">Application → approval workflow with substitute tracking</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 flex items-center gap-2">
          <Plus size={16} /> New Leave Request
        </button>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {showForm && (
        <div className="mt-6 bg-white rounded-lg shadow border p-4">
          <h2 className="font-semibold mb-4">New Leave Request</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <select value={form.staff_id} onChange={(e) => setForm({ ...form, staff_id: e.target.value })} className="border rounded-md px-3 py-2 text-sm">
              <option value="">Select staff...</option>
              {staff.map((s) => <option key={s.id} value={s.id}>{s.full_name}</option>)}
            </select>
            <select value={form.leave_type} onChange={(e) => setForm({ ...form, leave_type: e.target.value })} className="border rounded-md px-3 py-2 text-sm">
              <option value="annual">Annual</option>
              <option value="sick">Sick</option>
              <option value="maternity">Maternity</option>
              <option value="paternity">Paternity</option>
              <option value="compassionate">Compassionate</option>
              <option value="study">Study</option>
              <option value="unpaid">Unpaid</option>
              <option value="other">Other</option>
            </select>
            <input type="date" value={form.start_date} onChange={(e) => setForm({ ...form, start_date: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
            <input type="date" value={form.end_date} onChange={(e) => setForm({ ...form, end_date: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
            <input placeholder="Reason" value={form.reason} onChange={(e) => setForm({ ...form, reason: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
          </div>
          <div className="mt-4 flex gap-2">
            <button onClick={create} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700">Save</button>
            <button onClick={() => setShowForm(false)} className="bg-gray-100 text-gray-700 px-4 py-2 rounded-md text-sm hover:bg-gray-200">Cancel</button>
          </div>
        </div>
      )}

      <div className="mt-6 bg-white rounded-lg shadow border p-4 flex flex-wrap items-end gap-4">
        <div>
          <label className="block text-sm text-gray-500 mb-1">Status</label>
          <select value={status} onChange={(e) => setStatus(e.target.value)} className="border rounded-md px-3 py-2 text-sm">
            <option value="">All</option>
            <option value="pending">Pending</option>
            <option value="approved">Approved</option>
            <option value="denied">Denied</option>
            <option value="cancelled">Cancelled</option>
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
                <th className="px-4 py-3">Type</th>
                <th className="px-4 py-3">Dates</th>
                <th className="px-4 py-3">Days</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {leaves.length === 0 ? (
                <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-400">No leave requests found.</td></tr>
              ) : (
                leaves.map((l) => (
                  <tr key={l.id} className="border-t">
                    <td className="px-4 py-3 font-medium">{l.staff_name}</td>
                    <td className="px-4 py-3 capitalize">{l.leave_type}</td>
                    <td className="px-4 py-3">{l.start_date} → {l.end_date}</td>
                    <td className="px-4 py-3">{l.days}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded-full text-xs ${statusColor(l.status)}`}>{l.status}</span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button onClick={() => view(l.id)} className="text-blue-600 hover:text-blue-800" title="View"><Eye size={16} /></button>
                        {l.status === 'pending' && (
                          <>
                            <button onClick={() => approve(l.id)} className="text-green-600 hover:text-green-800" title="Approve"><Check size={16} /></button>
                            <button onClick={() => deny(l.id)} className="text-red-600 hover:text-red-800" title="Deny"><X size={16} /></button>
                          </>
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
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-xl max-w-lg w-full">
            <div className="p-6 border-b flex items-center justify-between">
              <div>
                <h2 className="text-xl font-bold">Leave Request</h2>
                <p className="text-sm text-gray-500">{selected.staff_name} · {selected.leave_type}</p>
              </div>
              <button onClick={() => setSelected(null)} className="text-gray-400 hover:text-gray-600 text-xl">×</button>
            </div>
            <div className="p-6 grid grid-cols-2 gap-4">
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Start Date</p>
                <p className="font-medium">{selected.start_date}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">End Date</p>
                <p className="font-medium">{selected.end_date}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Days</p>
                <p className="font-medium">{selected.days}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Status</p>
                <p className="font-medium capitalize">{selected.status}</p>
              </div>
              {selected.reason && (
                <div className="bg-gray-50 rounded-lg p-3 col-span-2">
                  <p className="text-xs text-gray-500">Reason</p>
                  <p className="font-medium">{selected.reason}</p>
                </div>
              )}
              {selected.denial_reason && (
                <div className="bg-red-50 rounded-lg p-3 col-span-2">
                  <p className="text-xs text-red-500">Denial Reason</p>
                  <p className="font-medium">{selected.denial_reason}</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}