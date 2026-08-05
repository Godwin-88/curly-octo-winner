'use client';

import { useEffect, useState } from 'react';
import { Plus, Bus, AlertTriangle } from 'lucide-react';
import { api, Vehicle } from '@/lib/api';

const STATUS_STYLES: Record<string, string> = {
  active: 'bg-green-100 text-green-700',
  maintenance: 'bg-yellow-100 text-yellow-700',
  retired: 'bg-gray-100 text-gray-500',
};

export default function VehiclesPage() {
  const token = ''; // TODO: Get from auth context
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [statusFilter, setStatusFilter] = useState('');
  const [form, setForm] = useState({
    registration: '',
    make: '',
    model: '',
    capacity: 14,
    year: '',
    status: 'active',
    driver_name: '',
    driver_phone: '',
    notes: '',
  });

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const data = await api.listVehicles({ status: statusFilter || undefined }, token);
      setVehicles(data);
    } catch (e: any) {
      setError(e.message || 'Failed to load vehicles');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, statusFilter]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await api.createVehicle({
        registration: form.registration,
        make: form.make,
        model: form.model,
        capacity: form.capacity,
        year: form.year ? Number(form.year) : undefined,
        status: form.status,
        driver_name: form.driver_name || undefined,
        driver_phone: form.driver_phone || undefined,
        notes: form.notes || undefined,
      }, token);
      setShowForm(false);
      setForm({ registration: '', make: '', model: '', capacity: 14, year: '', status: 'active', driver_name: '', driver_phone: '', notes: '' });
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to create vehicle');
    }
  };

  const handleStatusChange = async (v: Vehicle, status: string) => {
    try {
      await api.updateVehicle(v.id, { status }, token);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to update vehicle status');
    }
  };

  const stats = {
    active: vehicles.filter((v) => v.status === 'active').length,
    maintenance: vehicles.filter((v) => v.status === 'maintenance').length,
    totalCapacity: vehicles.reduce((sum, v) => sum + (v.status !== 'retired' ? v.capacity : 0), 0),
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Vehicles</h1>
          <p className="text-sm text-gray-500">Manage your school transport fleet</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="btn-primary flex items-center gap-2">
          <Plus size={16} /> Add Vehicle
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
        <div className="card p-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-blue-100 text-blue-600 flex items-center justify-center">
              <Bus size={20} />
            </div>
            <div>
              <p className="text-sm text-gray-500">Active Fleet</p>
              <p className="text-xl font-bold">{stats.active}</p>
            </div>
          </div>
        </div>
        <div className="card p-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-yellow-100 text-yellow-600 flex items-center justify-center">
              <AlertTriangle size={20} />
            </div>
            <div>
              <p className="text-sm text-gray-500">In Maintenance</p>
              <p className="text-xl font-bold">{stats.maintenance}</p>
            </div>
          </div>
        </div>
        <div className="card p-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-green-100 text-green-600 flex items-center justify-center">
              <Bus size={20} />
            </div>
            <div>
              <p className="text-sm text-gray-500">Total Seats</p>
              <p className="text-xl font-bold">{stats.totalCapacity}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Add form */}
      {showForm && (
        <div className="card p-5 mb-6">
          <h2 className="font-semibold mb-4">Add New Vehicle</h2>
          <form onSubmit={handleSubmit} className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Registration *</label>
              <input
                required
                value={form.registration}
                onChange={(e) => setForm({ ...form, registration: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
                placeholder="KDE 123A"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Make *</label>
              <input
                required
                value={form.make}
                onChange={(e) => setForm({ ...form, make: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
                placeholder="Toyota"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Model</label>
              <input
                value={form.model}
                onChange={(e) => setForm({ ...form, model: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
                placeholder="HiAce"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Capacity</label>
              <input
                type="number"
                value={form.capacity}
                onChange={(e) => setForm({ ...form, capacity: Number(e.target.value) })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Year</label>
              <input
                type="number"
                value={form.year}
                onChange={(e) => setForm({ ...form, year: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Status</label>
              <select
                value={form.status}
                onChange={(e) => setForm({ ...form, status: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              >
                <option value="active">Active</option>
                <option value="maintenance">Maintenance</option>
                <option value="retired">Retired</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Driver Name</label>
              <input
                value={form.driver_name}
                onChange={(e) => setForm({ ...form, driver_name: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Driver Phone</label>
              <input
                value={form.driver_phone}
                onChange={(e) => setForm({ ...form, driver_phone: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-600 mb-1">Notes</label>
              <input
                value={form.notes}
                onChange={(e) => setForm({ ...form, notes: e.target.value })}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div className="sm:col-span-2 lg:col-span-3 flex gap-2 pt-2">
              <button type="submit" className="btn-primary text-sm">Save Vehicle</button>
              <button type="button" onClick={() => setShowForm(false)} className="btn-secondary text-sm">Cancel</button>
            </div>
          </form>
        </div>
      )}

      {/* Status filter */}
      <div className="card p-4 mb-6">
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3 py-2 border rounded-md text-sm"
        >
          <option value="">All Statuses</option>
          <option value="active">Active</option>
          <option value="maintenance">Maintenance</option>
          <option value="retired">Retired</option>
        </select>
      </div>

      {error && <div className="bg-red-50 text-red-700 p-3 rounded-md mb-4 text-sm">{error}</div>}

      {/* Table */}
      <div className="card overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-400">Loading vehicles...</div>
        ) : vehicles.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            <Bus className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No vehicles found</p>
            <p className="text-xs mt-2">Add your first vehicle to get started</p>
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-gray-50 text-left text-gray-500">
                <th className="px-4 py-3 font-medium">Registration</th>
                <th className="px-4 py-3 font-medium">Make / Model</th>
                <th className="px-4 py-3 font-medium">Capacity</th>
                <th className="px-4 py-3 font-medium">Driver</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Insurance Expiry</th>
                <th className="px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {vehicles.map((v) => (
                <tr key={v.id} className="border-t hover:bg-gray-50">
                  <td className="px-4 py-3 font-medium">{v.registration}</td>
                  <td className="px-4 py-3">{v.make} {v.model}</td>
                  <td className="px-4 py-3">{v.capacity} seats</td>
                  <td className="px-4 py-3">{v.driver_name || '—'}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-0.5 rounded-full text-xs ${STATUS_STYLES[v.status] || 'bg-gray-100 text-gray-500'}`}>
                      {v.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-600">
                    {v.insurance_expiry ? new Date(v.insurance_expiry).toLocaleDateString() : '—'}
                  </td>
                  <td className="px-4 py-3">
                    <select
                      value={v.status}
                      onChange={(e) => handleStatusChange(v, e.target.value)}
                      className="text-xs border rounded px-1 py-0.5"
                    >
                      <option value="active">Active</option>
                      <option value="maintenance">Maintenance</option>
                      <option value="retired">Retired</option>
                    </select>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}