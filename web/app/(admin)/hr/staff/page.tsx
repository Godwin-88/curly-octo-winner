'use client';

import { useEffect, useState } from 'react';
import { Plus, Trash2, Eye } from 'lucide-react';
import { api, StaffProfile } from '@/lib/api';

export default function StaffPage() {
  const token = ''; // TODO: Get from auth context
  const [staff, setStaff] = useState<StaffProfile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [selected, setSelected] = useState<StaffProfile | null>(null);
  const [role, setRole] = useState('');
  const [department, setDepartment] = useState('');
  const [form, setForm] = useState({
    full_name: '', email: '', phone: '', role: 'teacher', tsc_number: '', national_id: '',
    kra_pin: '', department: '', job_title: '', employment_type: 'permanent', hire_date: '',
  });

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const data = await api.listStaff({ role, department, include_inactive: true }, token);
      setStaff(data);
    } catch (e: any) {
      setError(e.message || 'Failed to load staff');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, role, department]);

  const create = async () => {
    setError('');
    try {
      await api.createStaff(form, token);
      setShowForm(false);
      setForm({ full_name: '', email: '', phone: '', role: 'teacher', tsc_number: '', national_id: '', kra_pin: '', department: '', job_title: '', employment_type: 'permanent', hire_date: '' });
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to create staff');
    }
  };

  const remove = async (id: string) => {
    if (!confirm('Deactivate this staff member?')) return;
    try {
      await api.deleteStaff(id, token);
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to deactivate staff');
    }
  };

  const view = async (id: string) => {
    try {
      const s = await api.getStaff(id, token);
      setSelected(s);
    } catch (e: any) {
      setError(e.message || 'Failed to load staff');
    }
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Staff Directory</h1>
          <p className="text-gray-500">Manage staff profiles, TSC numbers & employment</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700 flex items-center gap-2">
          <Plus size={16} /> Add Staff
        </button>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {showForm && (
        <div className="mt-6 bg-white rounded-lg shadow border p-4">
          <h2 className="font-semibold mb-4">New Staff Member</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <input placeholder="Full name *" value={form.full_name} onChange={(e) => setForm({ ...form, full_name: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
            <input placeholder="Email *" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
            <input placeholder="Phone" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
            <select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })} className="border rounded-md px-3 py-2 text-sm">
              <option value="teacher">Teacher</option>
              <option value="principal">Principal</option>
              <option value="bursar">Bursar</option>
              <option value="transport_manager">Transport Manager</option>
              <option value="hr">HR</option>
              <option value="super_admin">Super Admin</option>
            </select>
            <input placeholder="TSC Number" value={form.tsc_number} onChange={(e) => setForm({ ...form, tsc_number: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
            <input placeholder="National ID" value={form.national_id} onChange={(e) => setForm({ ...form, national_id: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
            <input placeholder="KRA PIN" value={form.kra_pin} onChange={(e) => setForm({ ...form, kra_pin: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
            <input placeholder="Department" value={form.department} onChange={(e) => setForm({ ...form, department: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
            <input placeholder="Job Title" value={form.job_title} onChange={(e) => setForm({ ...form, job_title: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
            <select value={form.employment_type} onChange={(e) => setForm({ ...form, employment_type: e.target.value })} className="border rounded-md px-3 py-2 text-sm">
              <option value="permanent">Permanent</option>
              <option value="contract">Contract</option>
              <option value="temporary">Temporary</option>
              <option value="intern">Intern</option>
              <option value="volunteer">Volunteer</option>
            </select>
            <input type="date" value={form.hire_date} onChange={(e) => setForm({ ...form, hire_date: e.target.value })} className="border rounded-md px-3 py-2 text-sm" />
          </div>
          <div className="mt-4 flex gap-2">
            <button onClick={create} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm hover:bg-blue-700">Save</button>
            <button onClick={() => setShowForm(false)} className="bg-gray-100 text-gray-700 px-4 py-2 rounded-md text-sm hover:bg-gray-200">Cancel</button>
          </div>
        </div>
      )}

      <div className="mt-6 bg-white rounded-lg shadow border p-4 flex flex-wrap items-end gap-4">
        <div>
          <label className="block text-sm text-gray-500 mb-1">Role</label>
          <select value={role} onChange={(e) => setRole(e.target.value)} className="border rounded-md px-3 py-2 text-sm">
            <option value="">All roles</option>
            <option value="teacher">Teacher</option>
            <option value="principal">Principal</option>
            <option value="bursar">Bursar</option>
            <option value="transport_manager">Transport Manager</option>
            <option value="hr">HR</option>
          </select>
        </div>
        <div>
          <label className="block text-sm text-gray-500 mb-1">Department</label>
          <input value={department} onChange={(e) => setDepartment(e.target.value)} placeholder="Filter by department" className="border rounded-md px-3 py-2 text-sm" />
        </div>
      </div>

      {loading ? (
        <p className="text-gray-500 mt-4">Loading...</p>
      ) : (
        <div className="mt-6 bg-white rounded-lg shadow border overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-left text-gray-500">
              <tr>
                <th className="px-4 py-3">Name</th>
                <th className="px-4 py-3">Role</th>
                <th className="px-4 py-3">Department</th>
                <th className="px-4 py-3">TSC</th>
                <th className="px-4 py-3">Employment</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {staff.length === 0 ? (
                <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-400">No staff found.</td></tr>
              ) : (
                staff.map((s) => (
                  <tr key={s.id} className="border-t">
                    <td className="px-4 py-3 font-medium">{s.full_name}</td>
                    <td className="px-4 py-3 capitalize">{s.role}</td>
                    <td className="px-4 py-3">{s.department || '—'}</td>
                    <td className="px-4 py-3">{s.tsc_number || '—'}</td>
                    <td className="px-4 py-3 capitalize">{s.employment_type}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded-full text-xs ${s.is_active ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-600'}`}>
                        {s.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button onClick={() => view(s.id)} className="text-blue-600 hover:text-blue-800" title="View"><Eye size={16} /></button>
                        <button onClick={() => remove(s.id)} className="text-red-600 hover:text-red-800" title="Deactivate"><Trash2 size={16} /></button>
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
                <h2 className="text-xl font-bold">{selected.full_name}</h2>
                <p className="text-sm text-gray-500">{selected.job_title || selected.role} · {selected.email}</p>
              </div>
              <button onClick={() => setSelected(null)} className="text-gray-400 hover:text-gray-600 text-xl">×</button>
            </div>
            <div className="p-6 grid grid-cols-2 md:grid-cols-3 gap-4">
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Phone</p>
                <p className="font-medium">{selected.phone || '—'}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">TSC Number</p>
                <p className="font-medium">{selected.tsc_number || '—'}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">National ID</p>
                <p className="font-medium">{selected.national_id || '—'}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">KRA PIN</p>
                <p className="font-medium">{selected.kra_pin || '—'}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Department</p>
                <p className="font-medium">{selected.department || '—'}</p>
              </div>
              <div className="bg-gray-50 rounded-lg p-3">
                <p className="text-xs text-gray-500">Hire Date</p>
                <p className="font-medium">{selected.hire_date || '—'}</p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}