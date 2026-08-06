'use client';

import { useEffect, useState } from 'react';
import { api, PurchaseRequisition, CreateRequisitionRequest, RequisitionItemInput } from '@/lib/api';

const token = ''; // TODO: Get from auth context

export default function RequisitionsPage() {
  const [requisitions, setRequisitions] = useState<PurchaseRequisition[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [status, setStatus] = useState('');
  const [selected, setSelected] = useState<PurchaseRequisition | null>(null);
  const [form, setForm] = useState<CreateRequisitionRequest>({ title: '', items: [] });
  const [rejectReason, setRejectReason] = useState('');

  async function load() {
    try {
      const data = await api.listRequisitions({ status: status || undefined }, token);
      setRequisitions(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status]);

  function addItem() {
    setForm({ ...form, items: [...form.items, { item_name: '', quantity: 1, estimated_unit_cost_cents: 0 }] });
  }

  function updateItem(idx: number, field: keyof RequisitionItemInput, value: string | number) {
    const items = form.items.map((it, i) => (i === idx ? { ...it, [field]: value } : it));
    setForm({ ...form, items });
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.createRequisition(form, token);
      setForm({ title: '', items: [] });
      setShowForm(false);
      load();
    } catch (err) {
      console.error(err);
    }
  }

  async function handleApprove(r: PurchaseRequisition) {
    try {
      await api.approveRequisition(r.id, {}, token);
      load();
    } catch (err) {
      console.error(err);
    }
  }

  async function handleReject(r: PurchaseRequisition) {
    if (!rejectReason) return;
    try {
      await api.rejectRequisition(r.id, { rejection_reason: rejectReason }, token);
      setRejectReason('');
      setSelected(null);
      load();
    } catch (err) {
      console.error(err);
    }
  }

  async function handleCancel(r: PurchaseRequisition) {
    try {
      await api.cancelRequisition(r.id, token);
      load();
    } catch (err) {
      console.error(err);
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Purchase Requisitions</h1>
          <p className="text-gray-500">Staff submits → HoD approves → Principal/Bursar approves</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">
          {showForm ? 'Cancel' : 'New Requisition'}
        </button>
      </div>

      <div className="flex gap-2 flex-wrap">
        {['', 'pending', 'hod_approved', 'approved', 'rejected', 'cancelled', 'ordered'].map((s) => (
          <button key={s} onClick={() => setStatus(s)} className={`px-3 py-1 rounded-full text-sm ${status === s ? 'bg-blue-600 text-white' : 'bg-gray-200'}`}>
            {s === '' ? 'All' : s}
          </button>
        ))}
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="bg-white p-4 rounded-lg shadow border border-gray-200 space-y-3">
          <h2 className="font-semibold">New Requisition</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <input required placeholder="Title *" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Department" value={form.department || ''} onChange={(e) => setForm({ ...form, department: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Required by (YYYY-MM-DD)" value={form.required_by || ''} onChange={(e) => setForm({ ...form, required_by: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Justification" value={form.justification || ''} onChange={(e) => setForm({ ...form, justification: e.target.value })} className="border rounded-md px-3 py-2" />
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <h3 className="font-medium text-sm">Items</h3>
              <button type="button" onClick={addItem} className="text-blue-600 text-sm hover:underline">+ Add Item</button>
            </div>
            {form.items.map((it, idx) => (
              <div key={idx} className="grid grid-cols-2 md:grid-cols-5 gap-2">
                <input required placeholder="Item name" value={it.item_name} onChange={(e) => updateItem(idx, 'item_name', e.target.value)} className="border rounded-md px-3 py-2" />
                <input type="number" min={1} placeholder="Qty" value={it.quantity} onChange={(e) => updateItem(idx, 'quantity', Number(e.target.value))} className="border rounded-md px-3 py-2" />
                <input placeholder="Unit" value={it.unit || ''} onChange={(e) => updateItem(idx, 'unit', e.target.value)} className="border rounded-md px-3 py-2" />
                <input type="number" min={0} placeholder="Unit cost (KSh)" value={it.estimated_unit_cost_cents / 100} onChange={(e) => updateItem(idx, 'estimated_unit_cost_cents', Math.round(Number(e.target.value) * 100))} className="border rounded-md px-3 py-2" />
                <button type="button" onClick={() => setForm({ ...form, items: form.items.filter((_, i) => i !== idx) })} className="text-red-600 text-sm">Remove</button>
              </div>
            ))}
          </div>

          <button type="submit" className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">Submit Requisition</button>
        </form>
      )}

      <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-left">
            <tr>
              <th className="px-4 py-3">Requisition</th>
              <th className="px-4 py-3">Department</th>
              <th className="px-4 py-3">Requested By</th>
              <th className="px-4 py-3">Estimate</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {requisitions.map((r) => (
              <tr key={r.id} className="hover:bg-gray-50">
                <td className="px-4 py-3">
                  <button onClick={() => setSelected(r)} className="font-medium text-blue-600 hover:underline">{r.requisition_no}</button>
                  <p className="text-xs text-gray-500">{r.title}</p>
                </td>
                <td className="px-4 py-3">{r.department || '-'}</td>
                <td className="px-4 py-3">{r.requested_by_name || '-'}</td>
                <td className="px-4 py-3">KSh {(r.total_estimate_cents / 100).toLocaleString()}</td>
                <td className="px-4 py-3"><span className={`text-xs px-2 py-1 rounded-full ${statusColor(r.status)}`}>{r.status}</span></td>
                <td className="px-4 py-3 space-x-2">
                  {(r.status === 'pending' || r.status === 'hod_approved') && (
                    <button onClick={() => handleApprove(r)} className="text-green-600 hover:underline text-xs">Approve</button>
                  )}
                  {(r.status === 'pending' || r.status === 'hod_approved') && (
                    <button onClick={() => { setSelected(r); setRejectReason(''); }} className="text-red-600 hover:underline text-xs">Reject</button>
                  )}
                  {(r.status === 'pending' || r.status === 'hod_approved') && (
                    <button onClick={() => handleCancel(r)} className="text-gray-600 hover:underline text-xs">Cancel</button>
                  )}
                </td>
              </tr>
            ))}
            {requisitions.length === 0 && !loading && (
              <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-400">No requisitions found</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {selected && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg shadow-lg max-w-2xl w-full p-6 max-h-[80vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-bold">{selected.requisition_no}</h2>
              <button onClick={() => setSelected(null)} className="text-gray-500 hover:text-gray-700">✕</button>
            </div>
            <p className="font-medium">{selected.title}</p>
            <p className="text-sm text-gray-500 mb-2">{selected.department || 'General'} · {selected.requested_by_name || 'Unknown'} · {new Date(selected.requested_at).toLocaleDateString()}</p>
            {selected.justification && <p className="text-sm mb-2"><strong>Justification:</strong> {selected.justification}</p>}
            {selected.rejection_reason && <p className="text-sm text-red-600 mb-2"><strong>Rejection:</strong> {selected.rejection_reason}</p>}

            <table className="w-full text-sm mb-4">
              <thead className="bg-gray-50 text-left">
                <tr>
                  <th className="px-3 py-2">Item</th>
                  <th className="px-3 py-2">Qty</th>
                  <th className="px-3 py-2">Unit</th>
                  <th className="px-3 py-2">Unit Cost</th>
                  <th className="px-3 py-2">Total</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {(selected.items || []).map((it) => (
                  <tr key={it.id}>
                    <td className="px-3 py-2">{it.item_name}</td>
                    <td className="px-3 py-2">{it.quantity}</td>
                    <td className="px-3 py-2">{it.unit || '-'}</td>
                    <td className="px-3 py-2">KSh {(it.estimated_unit_cost_cents / 100).toLocaleString()}</td>
                    <td className="px-3 py-2">KSh {(it.estimated_total_cents / 100).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>

            <p className="font-semibold text-right">Total: KSh {(selected.total_estimate_cents / 100).toLocaleString()}</p>

            {rejectReason !== undefined && (selected.status === 'pending' || selected.status === 'hod_approved') && (
              <div className="mt-4 space-y-2">
                <input placeholder="Rejection reason" value={rejectReason} onChange={(e) => setRejectReason(e.target.value)} className="border rounded-md px-3 py-2 w-full" />
                <button onClick={() => handleReject(selected)} className="bg-red-600 text-white px-4 py-2 rounded-md hover:bg-red-700">Confirm Reject</button>
              </div>
            )}
          </div>
        </div>
      )}

      {loading && <p className="text-center text-gray-400">Loading...</p>}
    </div>
  );
}

function statusColor(status: string) {
  const map: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-700',
    hod_approved: 'bg-blue-100 text-blue-700',
    approved: 'bg-green-100 text-green-700',
    rejected: 'bg-red-100 text-red-700',
    cancelled: 'bg-gray-100 text-gray-600',
    ordered: 'bg-purple-100 text-purple-700',
  };
  return map[status] || 'bg-gray-100 text-gray-600';
}