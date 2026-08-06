'use client';

import { useEffect, useState } from 'react';
import { api, GoodsReceipt, PurchaseOrder, CreateGoodsReceiptRequest, GoodsReceiptItemInput } from '@/lib/api';

const token = ''; // TODO: Get from auth context

export default function ReceiptsPage() {
  const [receipts, setReceipts] = useState<GoodsReceipt[]>([]);
  const [orders, setOrders] = useState<PurchaseOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [status, setStatus] = useState('');
  const [selected, setSelected] = useState<GoodsReceipt | null>(null);
  const [form, setForm] = useState<CreateGoodsReceiptRequest>({ purchase_order_id: '', items: [] });

  async function load() {
    try {
      const [grn, po] = await Promise.all([
        api.listGoodsReceipts({ status: status || undefined }, token),
        api.listPurchaseOrders({}, token),
      ]);
      setReceipts(grn);
      setOrders(po);
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
    setForm({ ...form, items: [...form.items, { item_name: '', quantity_received: 1, quantity_rejected: 0, unit_cost_cents: 0 }] });
  }

  function updateItem(idx: number, field: keyof GoodsReceiptItemInput, value: string | number) {
    const items = form.items.map((it, i) => (i === idx ? { ...it, [field]: value } : it));
    setForm({ ...form, items });
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.createGoodsReceipt(form, token);
      setForm({ purchase_order_id: '', items: [] });
      setShowForm(false);
      load();
    } catch (err) {
      console.error(err);
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Goods Receipt Notes</h1>
          <p className="text-gray-500">Receiving officer confirms delivery; quantity verified</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">
          {showForm ? 'Cancel' : 'New Goods Receipt'}
        </button>
      </div>

      <div className="flex gap-2 flex-wrap">
        {['', 'received', 'partial', 'rejected'].map((s) => (
          <button key={s} onClick={() => setStatus(s)} className={`px-3 py-1 rounded-full text-sm ${status === s ? 'bg-blue-600 text-white' : 'bg-gray-200'}`}>
            {s === '' ? 'All' : s}
          </button>
        ))}
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="bg-white p-4 rounded-lg shadow border border-gray-200 space-y-3">
          <h2 className="font-semibold">New Goods Receipt</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <select required value={form.purchase_order_id} onChange={(e) => setForm({ ...form, purchase_order_id: e.target.value })} className="border rounded-md px-3 py-2">
              <option value="">Select purchase order *</option>
              {orders.filter((o) => o.status === 'sent' || o.status === 'partially_received').map((o) => (
                <option key={o.id} value={o.id}>{o.po_number} · {o.supplier_name}</option>
              ))}
            </select>
            <input placeholder="Received date (YYYY-MM-DD)" value={form.received_date || ''} onChange={(e) => setForm({ ...form, received_date: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Notes" value={form.notes || ''} onChange={(e) => setForm({ ...form, notes: e.target.value })} className="border rounded-md px-3 py-2" />
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <h3 className="font-medium text-sm">Items</h3>
              <button type="button" onClick={addItem} className="text-blue-600 text-sm hover:underline">+ Add Item</button>
            </div>
            {form.items.map((it, idx) => (
              <div key={idx} className="grid grid-cols-2 md:grid-cols-6 gap-2">
                <input required placeholder="Item name" value={it.item_name} onChange={(e) => updateItem(idx, 'item_name', e.target.value)} className="border rounded-md px-3 py-2" />
                <input type="number" min={1} placeholder="Received" value={it.quantity_received} onChange={(e) => updateItem(idx, 'quantity_received', Number(e.target.value))} className="border rounded-md px-3 py-2" />
                <input type="number" min={0} placeholder="Rejected" value={it.quantity_rejected} onChange={(e) => updateItem(idx, 'quantity_rejected', Number(e.target.value))} className="border rounded-md px-3 py-2" />
                <input placeholder="Unit" value={it.unit || ''} onChange={(e) => updateItem(idx, 'unit', e.target.value)} className="border rounded-md px-3 py-2" />
                <input type="number" min={0} placeholder="Unit cost (KSh)" value={it.unit_cost_cents / 100} onChange={(e) => updateItem(idx, 'unit_cost_cents', Math.round(Number(e.target.value) * 100))} className="border rounded-md px-3 py-2" />
                <button type="button" onClick={() => setForm({ ...form, items: form.items.filter((_, i) => i !== idx) })} className="text-red-600 text-sm">Remove</button>
              </div>
            ))}
          </div>

          <button type="submit" className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">Create Goods Receipt</button>
        </form>
      )}

      <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-left">
            <tr>
              <th className="px-4 py-3">GRN Number</th>
              <th className="px-4 py-3">PO</th>
              <th className="px-4 py-3">Supplier</th>
              <th className="px-4 py-3">Received Date</th>
              <th className="px-4 py-3">Received By</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {receipts.map((g) => (
              <tr key={g.id} className="hover:bg-gray-50">
                <td className="px-4 py-3">
                  <button onClick={() => setSelected(g)} className="font-medium text-blue-600 hover:underline">{g.grn_number}</button>
                </td>
                <td className="px-4 py-3">{g.po_number}</td>
                <td className="px-4 py-3">{g.supplier_name}</td>
                <td className="px-4 py-3">{g.received_date}</td>
                <td className="px-4 py-3">{g.received_by_name || '-'}</td>
                <td className="px-4 py-3"><span className={`text-xs px-2 py-1 rounded-full ${statusColor(g.status)}`}>{g.status}</span></td>
                <td className="px-4 py-3">
                  <button onClick={() => setSelected(g)} className="text-blue-600 hover:underline text-xs">View</button>
                </td>
              </tr>
            ))}
            {receipts.length === 0 && !loading && (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-400">No goods receipts found</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {selected && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg shadow-lg max-w-2xl w-full p-6 max-h-[80vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-bold">{selected.grn_number}</h2>
              <button onClick={() => setSelected(null)} className="text-gray-500 hover:text-gray-700">✕</button>
            </div>
            <p className="text-sm text-gray-500 mb-2">{selected.supplier_name} · PO {selected.po_number} · Received {selected.received_date}</p>
            {selected.notes && <p className="text-sm mb-2">{selected.notes}</p>}

            <table className="w-full text-sm mb-4">
              <thead className="bg-gray-50 text-left">
                <tr>
                  <th className="px-3 py-2">Item</th>
                  <th className="px-3 py-2">Received</th>
                  <th className="px-3 py-2">Rejected</th>
                  <th className="px-3 py-2">Unit Cost</th>
                  <th className="px-3 py-2">Total</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {(selected.items || []).map((it) => (
                  <tr key={it.id}>
                    <td className="px-3 py-2">{it.item_name}</td>
                    <td className="px-3 py-2">{it.quantity_received}</td>
                    <td className="px-3 py-2">{it.quantity_rejected}</td>
                    <td className="px-3 py-2">KSh {(it.unit_cost_cents / 100).toLocaleString()}</td>
                    <td className="px-3 py-2">KSh {(it.total_cost_cents / 100).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {loading && <p className="text-center text-gray-400">Loading...</p>}
    </div>
  );
}

function statusColor(status: string) {
  const map: Record<string, string> = {
    received: 'bg-green-100 text-green-700',
    partial: 'bg-yellow-100 text-yellow-700',
    rejected: 'bg-red-100 text-red-700',
  };
  return map[status] || 'bg-gray-100 text-gray-600';
}