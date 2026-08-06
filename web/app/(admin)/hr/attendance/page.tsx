'use client';

import { useEffect, useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { api, StaffAttendance, StaffProfile } from '@/lib/api';

export default function StaffAttendancePage() {
  const token = ''; // TODO: Get from auth context
  const [records, setRecords] = useState<StaffAttendance[]>([]);
  const [staff, setStaff] = useState<StaffProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10));
  const [staffId, setStaffId] = useState('');
  const [status, setStatus] = useState('');
  const [form, setForm] = useState({
    staff_id: '', date: new Date().toISOString().slice(0, 10), status: 'present', notes: '',
  });

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [recordData, staffData] = await Promise.all([
        api.listStaffAttendance({ date, staff_id: staffId, status }, token),
        api.listStaff({}, token),
      ]);
      setRecords(recordData);
      setStaff(staffData);
    } catch (e: any) {
      setError(e.message || 'Failed to load staff attendance');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, date, staffId, status]);

  const create = async () => {
    setError('');
    try {
      await api.createStaffAttendance(form, token);
      setShowForm(false);
      setForm({ staff_id: '', date: new Date().toISOString().slice(0, 10), status: 'present', notes: '' });
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to record attendance');
    }
  };

  const remove = async (id: string) => {
    if (!confirm('Delete this attendance record?')) return;
    try {
      await api.deleteStaffAttendance(id, token);
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to delete attendance record');
    }
  };

  const statusColor = (s: string) => {
    if (s === 'present') return 'bg-green-50 text-green-700';
    if (s === 'late') return 'bg-yellow-50 text-yellow-700';
    if (s === 'absent') return 'bg-red-50 text-red-700';
    if (s === 'leave') return 'bg-blue-50 text-blue-700';
    if (s === 'holiday') return 'bg-purple-50 text-purple-700';
    return 'bg-gray-100 text-gray-600';
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Staff Attendance</h1>
          <p className="text-gray-500">Daily clock-in/out records for staff</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 flex items-center gap-2">
          <Plus size={16} /> Record Attendance
        </button>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {showForm && (
        <div className="mt-6 bg-white rounded-lg shadow border p-4">
          <h2 className="font-semibold mb-4">Record Attendance</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <select value={form.staff_id} onChange={(e) => setForm({ ...form, staff_id: e.target.value })} className="border rounded-md px-3 py-2 text-sm">
              <option value="">Select staff...</option>
              {staff.map((s) => <option key={s.id} value={s.id}>{s.full_name}</option>)}
            </select>
            <input type="date" value={form.date} onChange={(e) => setForm({ ...form, date: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
            <select value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })} className="border rounded-md px-3 py-2 text-sm">
              <option value="present">Present</option>
              <option value="absent">Absent</option>
              <option value="late">Late</option>
              <option value="half_day">Half Day</option>
              <option value="leave">Leave</option>
              <option value="holiday">Holiday</option>
            </select>
            <input placeholder="Notes" value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
          </div>
          <div className="mt-4 flex gap-2">
            <button onClick={create} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700">Save</button>
            <button onClick={() => setShowForm(false)} className="bg-gray-100 text-gray-700 px-4 py-2 rounded-md text-sm hover:bg-gray-200">Cancel</button>
          </div>
        </div>
      )}

      <div className="mt-6 bg-white rounded-lg shadow border p-4 flex flex-wrap items-end gap-4">
        <div>
          <label className="block text-sm text-gray-500 mb-1">Date</label>
          <input type="date" value={date} onChange={(e) => setDate(e.target.value)} className="border rounded-md px-3 py-2 text-sm" />
        </div>
        <div>
          <label className="block text-sm text-gray-500 mb-1">Staff</label>
          <select value={staffId} onChange={(e) => setStaffId(e.target.value)} className="border rounded-md px-3 py-2 text-sm">
            <option value="">All staff</option>
            {staff.map((s) => <option key={s.id} value={s.id}>{s.full_name}</option>)}
          </select>
        </div>
        <div>
          <label className="block text-sm text-gray-500 mb-1">Status</label>
          <select value={status} onChange={(e) => setStatus(e.target.value)} className="border rounded-md px-3 py-2 text-sm">
            <option value="">All</option>
            <option value="present">Present</option>
            <option value="absent">Absent</option>
            <option value="late">Late</option>
            <option value="leave">Leave</option>
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
                <th className="px-4 py-3">Date</th>
                <th className="px-4 py-3">Clock In</th>
                <th className="px-4 py-3">Clock Out</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {records.length === 0 ? (
                <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-400">No attendance records found.</td></tr>
              ) : (
                records.map((r) => (
                  <tr key={r.id} className="border-t">
                    <td className="px-4 py-3 font-medium">{r.staff_name}</td>
                    <td className="px-4 py-3">{r.date}</td>
                    <td className="px-4 py-3">{r.clock_in ? new Date(r.clock_in).toLocaleTimeString() : '—'}</td>
                    <td className="px-4 py-3">{r.clock_out ? new Date(r.clock_out).toLocaleTimeString() : '—'}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded-full text-xs ${statusColor(r.status)}`}>{r.status}</span>
                    </td>
                    <td className="px-4 py-3">
                      <button onClick={() => remove(r.id)} className="text-red-600 hover:text-red-800" title="Delete"><Trash2 size={16} /></button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}