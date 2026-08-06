'use client';

import { useEffect, useState } from 'react';
import { api, Supplier, CreateSupplierRequest } from '@/lib/api';

const token = ''; // TODO: Get from auth context

const categories = ['textbooks', 'stationery', 'furniture', 'ict', 'uniforms', 'food', 'lab', 'construction', 'transport', 'general'];

export default function SuppliersPage() {
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [category, setCategory] = useState('');
  const [form, setForm] = useState<CreateSupplierRequest>({ name: '' });

  async function load() {
    try {
      const data = await api.listSuppliers({ category: category || undefined }, token);
      setSuppliers(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [category]);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.createSupplier(form, token);
      setForm({ name: '' });
      setShowForm(false);
      load();
    } catch (err) {
      console.error(err);
    }
  }

  async function handleToggleActive(s: Supplier) {
    try {
      await api.updateSupplier(s.id, { is_active: !s.is_active }, token);
      load();
    } catch (err) {
      console.error(err);
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Supplier Registry</h1>
          <p className="text-gray-500">KYC: business registration, KRA PIN, bank details & contact</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">
          {showForm ? 'Cancel' : 'Add Supplier'}
        </button>
      </div>

      <div className="flex gap-2 flex-wrap">
        <button onClick={() => setCategory('')} className={`px-3 py-1 rounded-full text-sm ${category === '' ? 'bg-blue-600 text-white' : 'bg-gray-200'}`}>
          All
        </button>
        {categories.map((c) => (
          <button key={c} onClick={() => setCategory(c)} className={`px-3 py-1 rounded-full text-sm ${category === c ? 'bg-blue-600 text-white' : 'bg-gray-200'}`}>
            {c}
          </button>
        ))}
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="bg-white p-4 rounded-lg shadow border border-gray-200 space-y-3">
          <h2 className="font-semibold">New Supplier</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <input required placeholder="Business name *" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Business registration" value={form.business_registration || ''} onChange={(e) => setForm({ ...form, business_registration: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="KRA PIN" value={form.kra_pin || ''} onChange={(e) => setForm({ ...form, kra_pin: e.target.value })} className="border rounded-md px-3 py-2" />
            <select value={form.category || 'general'} onChange={(e) => setForm({ ...form, category: e.target.value })} className="border rounded-md px-3 py-2">
              {categories.map((c) => <option key={c} value={c}>{c}</option>)}
            </select>
            <input placeholder="Contact person" value={form.contact_person || ''} onChange={(e) => setForm({ ...form, contact_person: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Phone" value={form.phone || ''} onChange={(e) => setForm({ ...form, phone: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Email" value={form.email || ''} onChange={(e) => setForm({ ...form, email: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="WhatsApp phone" value={form.whatsapp_phone || ''} onChange={(e) => setForm({ ...form, whatsapp_phone: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Bank account name" value={form.bank_account_name || ''} onChange={(e) => setForm({ ...form, bank_account_name: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Bank account number" value={form.bank_account_number || ''} onChange={(e) => setForm({ ...form, bank_account_number: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Bank branch" value={form.bank_branch || ''} onChange={(e) => setForm({ ...form, bank_branch: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Physical address" value={form.physical_address || ''} onChange={(e) => setForm({ ...form, physical_address: e.target.value })} className="border rounded-md px-3 py-2" />
          </div>
          <button type="submit" className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">Save Supplier</button>
        </form>
      )}

      <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-left">
            <tr>
              <th className="px-4 py-3">Name</th>
              <th className="px-4 py-3">Category</th>
              <th className="px-4 py-3">KRA PIN</th>
              <th className="px-4 py-3">Contact</th>
              <th className="px-4 py-3">Bank</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {suppliers.map((s) => (
              <tr key={s.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-medium">{s.name}</td>
                <td className="px-4 py-3"><span className="text-xs px-2 py-1 rounded-full bg-gray-100">{s.category}</span></td>
                <td className="px-4 py-3">{s.kra_pin || '-'}</td>
                <td className="px-4 py-3">
                  <p>{s.contact_person || '-'}</p>
                  <p className="text-xs text-gray-500">{s.phone || s.whatsapp_phone || ''}</p>
                </td>
                <td className="px-4 py-3">
                  <p>{s.bank_account_name || '-'}</p>
                  <p className="text-xs text-gray-500">{s.bank_account_number || ''}</p>
                </td>
                <td className="px-4 py-3">
                  <span className={`text-xs px-2 py-1 rounded-full ${s.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'}`}>
                    {s.is_active ? 'Active' : 'Inactive'}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <button onClick={() => handleToggleActive(s)} className="text-blue-600 hover:underline text-xs">
                    {s.is_active ? 'Deactivate' : 'Activate'}
                  </button>
                </td>
              </tr>
            ))}
            {suppliers.length === 0 && !loading && (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-400">No suppliers found</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {loading && <p className="text-center text-gray-400">Loading...</p>}
    </div>
  );
}