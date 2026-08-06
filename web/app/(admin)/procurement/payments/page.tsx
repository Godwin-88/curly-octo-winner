'use client';

import { useEffect, useState } from 'react';
import { api, SupplierPayment, Supplier, PurchaseOrder, GoodsReceipt, CreateSupplierPaymentRequest } from '@/lib/api';

const token = ''; // TODO: Get from auth context

export default function PaymentsPage() {
  const [payments, setPayments] = useState<SupplierPayment[]>([]);
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [orders, setOrders] = useState<PurchaseOrder[]>([]);
  const [receipts, setReceipts] = useState<GoodsReceipt[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [status, setStatus] = useState('');
  const [form, setForm] = useState<CreateSupplierPaymentRequest>({ supplier_id: '', amount_cents: 0 });

  async function load() {
    try {
      const [pay, sup, po, grn] = await Promise.all([
        api.listSupplierPayments({ status: status || undefined }, token),
        api.listSuppliers({}, token),
        api.listPurchaseOrders({}, token),
        api.listGoodsReceipts({}, token),
      ]);
      setPayments(pay);
      setSuppliers(sup);
      setOrders(po);
      setReceipts(grn);
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

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    try {
      await api.createSupplierPayment(form, token);
      setForm({ supplier_id: '', amount_cents: 0 });
      setShowForm(false);
      load();
    } catch (err) {
      console.error(err);
    }
  }

  async function handleStatusChange(p: SupplierPayment, newStatus: string) {
    try {
      await api.updateSupplierPayment(p.id, { status: newStatus }, token);
      load();
    } catch (err) {
      console.error(err);
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Supplier Payments</h1>
          <p className="text-gray-500">Three-way match (PO → GRN → Invoice) before payment authorisation</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">
          {showForm ? 'Cancel' : 'New Payment'}
        </button>
      </div>

      <div className="flex gap-2 flex-wrap">
        {['', 'pending', 'authorised', 'paid', 'cancelled'].map((s) => (
          <button key={s} onClick={() => setStatus(s)} className={`px-3 py-1 rounded-full text-sm ${status === s ? 'bg-blue-600 text-white' : 'bg-gray-200'}`}>
            {s === '' ? 'All' : s}
          </button>
        ))}
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="bg-white p-4 rounded-lg shadow border border-gray-200 space-y-3">
          <h2 className="font-semibold">New Supplier Payment</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <select required value={form.supplier_id} onChange={(e) => setForm({ ...form, supplier_id: e.target.value })} className="border rounded-md px-3 py-2">
              <option value="">Select supplier *</option>
              {suppliers.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
            <input type="number" min={1} required placeholder="Amount (KSh) *" value={form.amount_cents / 100} onChange={(e) => setForm({ ...form, amount_cents: Math.round(Number(e.target.value) * 100) })} className="border rounded-md px-3 py-2" />
            <select value={form.payment_method || 'bank'} onChange={(e) => setForm({ ...form, payment_method: e.target.value })} className="border rounded-md px-3 py-2">
              <option value="bank">Bank</option>
              <option value="mpesa">M-Pesa</option>
              <option value="cash">Cash</option>
              <option value="cheque">Cheque</option>
            </select>
            <select value={form.purchase_order_id || ''} onChange={(e) => setForm({ ...form, purchase_order_id: e.target.value || undefined })} className="border rounded-md px-3 py-2">
              <option value="">Select PO (optional)</option>
              {orders.map((o) => <option key={o.id} value={o.id}>{o.po_number} · {o.supplier_name}</option>)}
            </select>
            <select value={form.goods_receipt_id || ''} onChange={(e) => setForm({ ...form, goods_receipt_id: e.target.value || undefined })} className="border rounded-md px-3 py-2">
              <option value="">Select GRN (optional)</option>
              {receipts.map((g) => <option key={g.id} value={g.id}>{g.grn_number} · {g.supplier_name}</option>)}
            </select>
            <input placeholder="Invoice number" value={form.invoice_number || ''} onChange={(e) => setForm({ ...form, invoice_number: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Invoice date (YYYY-MM-DD)" value={form.invoice_date || ''} onChange={(e) => setForm({ ...form, invoice_date: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Reference" value={form.reference || ''} onChange={(e) => setForm({ ...form, reference: e.target.value })} className="border rounded-md px-3 py-2" />
            <input placeholder="Notes" value={form.notes || ''} onChange={(e) => setForm({ ...form, notes: e.target.value })} className="border rounded-md px-3 py-2" />
          </div>
          <button type="submit" className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">Create Payment</button>
        </form>
      )}

      <div className="bg-white rounded-lg shadow border border-gray-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-left">
            <tr>
              <th className="px-4 py-3">Payment No</th>
              <th className="px-4 py-3">Supplier</th>
              <th className="px-4 py-3">PO / GRN</th>
              <th className="px-4 py-3">Invoice</th>
              <th className="px-4 py-3">Amount</th>
              <th className="px-4 py-3">Method</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {payments.map((p) => (
              <tr key={p.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-medium">{p.payment_no}</td>
                <td className="px-4 py-3">{p.supplier_name}</td>
                <td className="px-4 py-3">
                  <p>{p.po_number || '-'}</p>
                  <p className="text-xs text-gray-500">{p.grn_number || ''}</p>
                </td>
                <td className="px-4 py-3">{p.invoice_number || '-'}</td>
                <td className="px-4 py-3 font-medium">KSh {(p.amount_cents / 100).toLocaleString()}</td>
                <td className="px-4 py-3">{p.payment_method}</td>
                <td className="px-4 py-3"><span className={`text-xs px-2 py-1 rounded-full ${statusColor(p.status)}`}>{p.status}</span></td>
                <td className="px-4 py-3 space-x-2">
                  {p.status === 'pending' && (
                    <button onClick={() => handleStatusChange(p, 'authorised')} className="text-blue-600 hover:underline text-xs">Authorise</button>
                  )}
                  {p.status === 'authorised' && (
                    <button onClick={() => handleStatusChange(p, 'paid')} className="text-green-600 hover:underline text-xs">Mark Paid</button>
                  )}
                  {(p.status === 'pending' || p.status === 'authorised') && (
                    <button onClick={() => handleStatusChange(p, 'cancelled')} className="text-red-600 hover:underline text-xs">Cancel</button>
                  )}
                </td>
              </tr>
            ))}
            {payments.length === 0 && !loading && (
              <tr><td colSpan={8} className="px-4 py-8 text-center text-gray-400">No supplier payments found</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {loading && <p className="text-center text-gray-400">Loading...</p>}
    </div>
  );
}

function statusColor(status: string) {
  const map: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-700',
    authorised: 'bg-blue-100 text-blue-700',
    paid: 'bg-green-100 text-green-700',
    cancelled: 'bg-red-100 text-red-700',
  };
  return map[status] || 'bg-gray-100 text-gray-600';
}