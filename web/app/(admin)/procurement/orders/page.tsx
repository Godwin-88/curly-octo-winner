'use client';

import { useEffect, useState } from 'react';
import { api, PurchaseOrder, Supplier, CreatePurchaseOrderRequest, PurchaseOrderItemInput } from '@/lib/api';

const token = ''; // TODO: Get from auth context

export default function OrdersPage() {
  const [orders, setOrders] = useState<PurchaseOrder[]>([]);
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [status, setStatus] = useState('');
  const [selected, setSelected] = useState<PurchaseOrder | null>(null);
  const [form, setForm] = useState<CreatePurchaseOrderRequest>({ supplier_id: '', items: [] });

  async function load() {
    try {
      const [po, sup] = await Promise.all([
        api.listPurchaseOrders({ status: status || undefined }, token),
        api.listSuppliers({}, token),
      ]);
      setOrders(po);
      setSuppliers(sup);
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
    setForm({ ...form, items: [...form.items, { item_name: '', quantity: 1, unit_cost_cents: 0 }] });
  }

  function updateItem(idx: number, field: keyof PurchaseOrderItemInput, value: string | number) {
    const items = form.items.map((it, i) => (i === idx ? { ...it, [field]: value } : it));
    setForm({ ...form, items });
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.createPurchaseOrder(form, token);
      setForm({ supplier_id: '', items: [] });
      setShowForm(false);
      load();
    } catch (err) {
      console.error(err);
    }
  }

  async function handleStatusChange(o: PurchaseOrder, newStatus: string) {
    try {
      await api.updatePurchaseOrder(o.id, { status: newStatus }, token);
      load();
    } catch (err) {
      console.error(err);
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Purchase Orders</h1>
          <p className="text-gray-500">PO with school letterhead; PDF sent to supplier via WhatsApp</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">
          {showForm ? 'Cancel' : 'New Purchase Order'}
        </button>
      </div>

      <div className="flex gap-2 flex-wrap">
        {['', 'draft', 'sent', 'partially_received', 'received', 'cancelled'].map((s) => (
          <button key={s} onClick={() => setStatus(s)} className={`px-3 py-1 rounded-full text-sm ${status === s ? 'bg-blue-600 text-white' : 'bg-gray-200'}`}>
            {s === '' ? 'All' : s}
          </button>
        ))}
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="bg-white p-4 rounded-lg shadow border border-gray-200 space-y-3">
          <h2 className="font-semibold">New Purchase Order</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <select required value={form.supplier_id} onChange={(e) => setForm({ ...form, supplier_id: e.target.value })} className="border rounded-md px-3 py-2">
              <option value="">Select supplier *</option>
              {suppliers.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
            <input placeholder="Expected delivery (YYYY-MM-DD)" value={form.expected_delivery || ''} onChange={(e) => setForm({ ...form, expected_delivery: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Notes" value={form.notes || ''} onChange={(e) => setForm({ ...form, notes: e.target.value })} className="border rounded-md px-3 py-2" />
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
                <input type="number" min={0} placeholder="Unit cost (KSh)" value={it.unit_cost_cents / 100} onChange={(e) => updateItem(idx, 'unit_cost_cents', Math.round(Number(e.target.value) * 100))} className="border rounded-md px-3 py-2" />
                <button type="button" onClick={() => setForm({ ...form, items: form.items.filter((_, i) => i !== idx) })} className="text-red-600 text-sm">Remove</button>
              </div>
            ))}
          </div>

          <button type="submit" className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">Create Purchase Order</button>
        </form>
      )}

      <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-left">
            <tr>
              <th className="px-4 py-3">PO Number</th>
              <th className="px-4 py-3">Supplier</th>
              <th className="px-4 py-3">Order Date</th>
              <th className="px-4 py-3">Expected</th>
              <th className="px-4 py-3">Total</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {orders.map((o) => (
              <tr key={o.id} className="hover:bg-gray-50">
                <td className="px-4 py-3">
                  <button onClick={() => setSelected(o)} className="font-medium text-blue-600 hover:underline">{o.po_number}</button>
                </td>
                <td className="px-4 py-3">{o.supplier_name}</td>
                <td className="px-4 py-3">{o.order_date}</td>
                <td className="px-4 py-3">{o.expected_delivery || '-'}</td>
                <td className="px-4 py-3">KSh {(o.total_amount_cents / 100).toLocaleString()}</td>
                <td className="px-4 py-3"><span className={`text-xs px-2 py-1 rounded-full ${statusColor(o.status)}`}>{o.status}</span></td>
                <td className="px-4 py-3 space-x-2">
                  {o.status === 'draft' && (
                    <button onClick={() => handleStatusChange(o, 'sent')} className="text-blue-600 hover:underline text-xs">Send</button>
                  )}
                  {o.status === 'sent' && (
                    <button onClick={() => handleStatusChange(o, 'cancelled')} className="text-red-600 hover:underline text-xs">Cancel</button>
                  )}
                </td>
              </tr>
            ))}
            {orders.length === 0 && !loading && (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-400">No purchase orders found</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {selected && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg shadow-lg max-w-2xl w-full p-6 max-h-[80vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-bold">{selected.po_number}</h2>
              <button onClick={() => setSelected(null)} className="text-gray-500 hover:text-gray-700">✕</button>
            </div>
            <p className="text-sm text-gray-500 mb-2">{selected.supplier_name} · Ordered {selected.order_date}</p>
            {selected.notes && <p className="text-sm mb-2">{selected.notes}</p>}

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
                    <td className="px-3 py-2">KSh {(it.unit_cost_cents / 100).toLocaleString()}</td>
                    <td className="px-3 py-2">KSh {(it.total_cost_cents / 100).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>

            <p className="font-semibold text-right">Total: KSh {(selected.total_amount_cents / 100).toLocaleString()}</p>
          </div>
        </div>
      )}

      {loading && <p className="text-center text-gray-400">Loading...</p>}
    </div>
  );
}

function statusColor(status: string) {
  const map: Record<string, string> = {
    draft: 'bg-gray-100 text-gray-600',
    sent: 'bg-blue-100 text-blue-700',
    partially_received: 'bg-yellow-100 text-yellow-700',
    received: 'bg-green-100 text-green-700',
    cancelled: 'bg-red-100 text-red-700',
  };
  return map[status] || 'bg-gray-100 text-gray-600';
}