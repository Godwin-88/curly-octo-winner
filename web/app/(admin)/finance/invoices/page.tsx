'use client';

import { useEffect, useState } from 'react';
import { Plus, Trash2, Smartphone, FileText, Wallet } from 'lucide-react';
import { api, Invoice, FeeStructure, Learner, Payment, Discount } from '@/lib/api';

const STATUS_STYLES: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-500',
  unpaid: 'bg-red-100 text-red-700',
  partially_paid: 'bg-yellow-100 text-yellow-700',
  paid: 'bg-green-100 text-green-700',
  overdue: 'bg-orange-100 text-orange-700',
  void: 'bg-gray-100 text-gray-400',
};

const DISCOUNT_TYPES = ['scholarship', 'sibling', 'waiver', 'other'];

export default function InvoicesPage() {
  const token = ''; // TODO: Get from auth context
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [learners, setLearners] = useState<Learner[]>([]);
  const [feeStructures, setFeeStructures] = useState<FeeStructure[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    learner_id: '',
    fee_structure_id: '',
    term: 1,
    year: new Date().getFullYear(),
    due_date: '',
  });

  const [detail, setDetail] = useState<Invoice | null>(null);
  const [payments, setPayments] = useState<Payment[]>([]);
  const [discounts, setDiscounts] = useState<Discount[]>([]);
  const [discountForm, setDiscountForm] = useState({ amount_cents: 0, discount_type: 'scholarship', reason: '' });
  const [paymentForm, setPaymentForm] = useState({ channel: 'cash', amount_cents: 0, reference: '', paid_by: '' });
  const [stkForm, setStkForm] = useState({ phone: '', amount_cents: 0 });

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [invoiceData, learnerData, feeData] = await Promise.all([
        api.listInvoices({ status: statusFilter || undefined }, token),
        api.listLearners({}, token),
        api.listFeeStructures({}, token),
      ]);
      setInvoices(invoiceData);
      setLearners(learnerData);
      setFeeStructures(feeData);
    } catch (e: any) {
      setError(e.message || 'Failed to load invoices');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, statusFilter]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await api.createInvoice({
        learner_id: form.learner_id,
        fee_structure_id: form.fee_structure_id || undefined,
        term: form.term,
        year: form.year,
        due_date: form.due_date || undefined,
      }, token);
      setShowForm(false);
      setForm({ learner_id: '', fee_structure_id: '', term: 1, year: new Date().getFullYear(), due_date: '' });
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to create invoice');
    }
  };

  const openDetail = async (invoice: Invoice) => {
    setDetail(invoice);
    setError('');
    try {
      const [payData, discData] = await Promise.all([
        api.listInvoicePayments(invoice.id, token),
        api.listInvoiceDiscounts(invoice.id, token),
      ]);
      setPayments(payData);
      setDiscounts(discData);
    } catch (e: any) {
      setError(e.message || 'Failed to load invoice details');
    }
  };

  const handleAddDiscount = async () => {
    if (!detail || !discountForm.amount_cents) return;
    setError('');
    try {
      await api.createInvoiceDiscount(detail.id, {
        amount_cents: discountForm.amount_cents,
        discount_type: discountForm.discount_type,
        reason: discountForm.reason || undefined,
      }, token);
      setDiscountForm({ amount_cents: 0, discount_type: 'scholarship', reason: '' });
      openDetail(detail);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to add discount');
    }
  };

  const handleDeleteDiscount = async (discountId: string) => {
    if (!detail) return;
    setError('');
    try {
      await api.deleteInvoiceDiscount(discountId, token);
      openDetail(detail);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to remove discount');
    }
  };

  const handleCreatePayment = async () => {
    if (!detail || !paymentForm.amount_cents) return;
    setError('');
    try {
      await api.createPayment({
        invoice_id: detail.id,
        amount_cents: paymentForm.amount_cents,
        channel: paymentForm.channel,
        reference: paymentForm.reference || undefined,
        paid_by: paymentForm.paid_by || undefined,
      }, token);
      setPaymentForm({ channel: 'cash', amount_cents: 0, reference: '', paid_by: '' });
      openDetail(detail);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to record payment');
    }
  };

  const handleReversePayment = async (paymentId: string) => {
    if (!detail) return;
    setError('');
    try {
      await api.reversePayment(paymentId, token);
      openDetail(detail);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to reverse payment');
    }
  };

  const handleMpesaStk = async () => {
    if (!detail || !stkForm.phone || !stkForm.amount_cents) return;
    setError('');
    try {
      await api.initiateMpesaStk({
        invoice_id: detail.id,
        phone: stkForm.phone,
        amount_cents: stkForm.amount_cents,
      }, token);
      setStkForm({ phone: '', amount_cents: 0 });
      window.alert('M-Pesa STK push sent. Ask the payer to enter their M-Pesa PIN.');
    } catch (e: any) {
      setError(e.message || 'Failed to initiate M-Pesa payment');
    }
  };

  const stats = {
    total: invoices.length,
    paid: invoices.filter((i) => i.status === 'paid').length,
    unpaid: invoices.filter((i) => i.status === 'unpaid' || i.status === 'overdue').length,
    collected: invoices.reduce((sum, i) => sum + i.paid_cents, 0),
    outstanding: invoices.reduce((sum, i) => sum + (i.balance_cents || 0), 0),
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Invoices</h1>
          <p className="text-gray-500">Learner fee bills and collections</p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
        >
          <Plus size={18} /> New Invoice
        </button>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-white rounded-lg shadow border p-4">
          <p className="text-sm text-gray-500">Total Invoices</p>
          <p className="text-2xl font-bold">{stats.total}</p>
        </div>
        <div className="bg-white rounded-lg shadow border p-4">
          <p className="text-sm text-gray-500">Paid</p>
          <p className="text-2xl font-bold text-green-600">{stats.paid}</p>
        </div>
        <div className="bg-white rounded-lg shadow border p-4">
          <p className="text-sm text-gray-500">Collected</p>
          <p className="text-2xl font-bold text-blue-600">KES {(stats.collected / 100).toLocaleString()}</p>
        </div>
        <div className="bg-white rounded-lg shadow border p-4">
          <p className="text-sm text-gray-500">Outstanding</p>
          <p className="text-2xl font-bold text-red-600">KES {(stats.outstanding / 100).toLocaleString()}</p>
        </div>
      </div>

      {error && <div className="mb-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      <div className="mb-4">
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="border rounded-md px-3 py-2"
        >
          <option value="">All statuses</option>
          {Object.keys(STATUS_STYLES).map((s) => (
            <option key={s} value={s}>{s.replace('_', ' ')}</option>
          ))}
        </select>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="mb-6 p-4 bg-white rounded-lg shadow border">
          <div className="grid grid-cols-2 gap-4 mb-4">
            <div>
              <label className="block text-sm font-medium mb-1">Learner</label>
              <select
                required
                value={form.learner_id}
                onChange={(e) => setForm({ ...form, learner_id: e.target.value })}
                className="w-full border rounded-md px-3 py-2"
              >
                <option value="">Select learner</option>
                {learners.map((l) => (
                  <option key={l.id} value={l.id}>{l.full_name} ({l.grade})</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Fee Structure</label>
              <select
                value={form.fee_structure_id}
                onChange={(e) => setForm({ ...form, fee_structure_id: e.target.value })}
                className="w-full border rounded-md px-3 py-2"
              >
                <option value="">Custom items (none)</option>
                {feeStructures.map((f) => (
                  <option key={f.id} value={f.id}>{f.name} ({f.grade})</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Term</label>
              <select
                value={form.term}
                onChange={(e) => setForm({ ...form, term: Number(e.target.value) })}
                className="w-full border rounded-md px-3 py-2"
              >
                {[1, 2, 3].map((t) => (
                  <option key={t} value={t}>Term {t}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Due Date</label>
              <input
                type="date"
                value={form.due_date}
                onChange={(e) => setForm({ ...form, due_date: e.target.value })}
                className="w-full border rounded-md px-3 py-2"
              />
            </div>
          </div>
          <button type="submit" className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">
            Create Invoice
          </button>
        </form>
      )}

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : invoices.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          <FileText size={48} className="mx-auto mb-3 text-gray-300" />
          <p>No invoices yet. Create one to get started.</p>
        </div>
      ) : (
        <div className="space-y-2">
          {invoices.map((inv) => (
            <div
              key={inv.id}
              className="bg-white rounded-lg shadow border p-4 flex items-center justify-between cursor-pointer hover:border-blue-400"
              onClick={() => openDetail(inv)}
            >
              <div>
                <p className="font-semibold">{inv.learner_name}</p>
                <p className="text-sm text-gray-500">
                  {inv.invoice_number} · Term {inv.term} {inv.year} · {inv.grade}
                </p>
              </div>
              <div className="text-right">
                <p className="font-bold">KES {(inv.total_cents / 100).toLocaleString()}</p>
                <p className="text-sm text-gray-500">Balance: KES {((inv.balance_cents || 0) / 100).toLocaleString()}</p>
              </div>
              <span className={`px-2 py-1 rounded-full text-xs ${STATUS_STYLES[inv.status] || 'bg-gray-100'}`}>
                {inv.status.replace('_', ' ')}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Detail modal */}
      {detail && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setDetail(null)}>
          <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto p-6" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-start justify-between mb-4">
              <div>
                <h2 className="text-xl font-bold">{detail.learner_name}</h2>
                <p className="text-sm text-gray-500">{detail.invoice_number} · Term {detail.term} {detail.year}</p>
              </div>
              <button onClick={() => setDetail(null)} className="text-gray-500 hover:text-gray-700">✕</button>
            </div>

            {/* Summary */}
            <div className="grid grid-cols-4 gap-2 mb-4 text-center">
              <div className="p-2 bg-gray-50 rounded">
                <p className="text-xs text-gray-500">Total</p>
                <p className="font-bold">KES {(detail.total_cents / 100).toLocaleString()}</p>
              </div>
              <div className="p-2 bg-green-50 rounded">
                <p className="text-xs text-gray-500">Paid</p>
                <p className="font-bold text-green-600">KES {(detail.paid_cents / 100).toLocaleString()}</p>
              </div>
              <div className="p-2 bg-yellow-50 rounded">
                <p className="text-xs text-gray-500">Discount</p>
                <p className="font-bold text-yellow-600">KES {(detail.discount_cents / 100).toLocaleString()}</p>
              </div>
              <div className="p-2 bg-red-50 rounded">
                <p className="text-xs text-gray-500">Balance</p>
                <p className="font-bold text-red-600">KES {((detail.balance_cents || 0) / 100).toLocaleString()}</p>
              </div>
            </div>

            {/* Line items */}
            <div className="mb-4">
              <h3 className="font-semibold mb-2">Line Items</h3>
              <div className="space-y-1">
                {detail.items?.map((item) => (
                  <div key={item.id} className="flex justify-between text-sm">
                    <span className="text-gray-600">{item.name}</span>
                    <span>KES {(item.amount_cents / 100).toLocaleString()}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Discounts */}
            <div className="mb-4">
              <h3 className="font-semibold mb-2">Discounts</h3>
              {discounts.length === 0 && <p className="text-sm text-gray-400">No discounts</p>}
              <div className="space-y-1 mb-2">
                {discounts.map((d) => (
                  <div key={d.id} className="flex justify-between items-center text-sm">
                    <span>
                      {d.discount_type} - KES {(d.amount_cents / 100).toLocaleString()}
                      {d.reason && <span className="text-gray-400"> ({d.reason})</span>}
                    </span>
                    <button onClick={() => handleDeleteDiscount(d.id)} className="text-red-500 hover:text-red-700">
                      <Trash2 size={14} />
                    </button>
                  </div>
                ))}
              </div>
              <div className="flex gap-2">
                <input
                  type="number"
                  value={discountForm.amount_cents}
                  onChange={(e) => setDiscountForm({ ...discountForm, amount_cents: Number(e.target.value) })}
                  className="border rounded-md px-3 py-1 w-32"
                  placeholder="Amount"
                />
                <select
                  value={discountForm.discount_type}
                  onChange={(e) => setDiscountForm({ ...discountForm, discount_type: e.target.value })}
                  className="border rounded-md px-3 py-1"
                >
                  {DISCOUNT_TYPES.map((t) => (
                    <option key={t} value={t}>{t}</option>
                  ))}
                </select>
                <input
                  value={discountForm.reason}
                  onChange={(e) => setDiscountForm({ ...discountForm, reason: e.target.value })}
                  className="border rounded-md px-3 py-1 flex-1"
                  placeholder="Reason"
                />
                <button onClick={handleAddDiscount} className="bg-yellow-600 text-white px-3 py-1 rounded-md hover:bg-yellow-700">
                  Add
                </button>
              </div>
            </div>

            {/* Payments */}
            <div className="mb-4">
              <h3 className="font-semibold mb-2">Payments</h3>
              {payments.length === 0 && <p className="text-sm text-gray-400">No payments</p>}
              <div className="space-y-1 mb-2">
                {payments.map((p) => (
                  <div key={p.id} className="flex justify-between items-center text-sm">
                    <span>
                      {p.channel} - KES {(p.amount_cents / 100).toLocaleString()}
                      {p.mpesa_receipt && <span className="text-gray-400"> ({p.mpesa_receipt})</span>}
                      <span className={`ml-2 px-2 py-0.5 rounded-full text-xs ${p.status === 'completed' ? 'bg-green-100 text-green-700' : p.status === 'pending' ? 'bg-yellow-100 text-yellow-700' : 'bg-red-100 text-red-700'}`}>
                        {p.status}
                      </span>
                    </span>
                    {p.status === 'completed' && (
                      <button onClick={() => handleReversePayment(p.id)} className="text-red-500 hover:text-red-700">
                        Reverse
                      </button>
                    )}
                  </div>
                ))}
              </div>

              {/* M-Pesa STK */}
              <div className="p-3 bg-blue-50 rounded-md mb-2">
                <p className="text-sm font-medium mb-2 flex items-center gap-1">
                  <Smartphone size={14} /> M-Pesa STK Push
                </p>
                <div className="flex gap-2">
                  <input
                    value={stkForm.phone}
                    onChange={(e) => setStkForm({ ...stkForm, phone: e.target.value })}
                    className="border rounded-md px-3 py-1 flex-1"
                    placeholder="Phone (07XXXXXXXX)"
                  />
                  <input
                    type="number"
                    value={stkForm.amount_cents}
                    onChange={(e) => setStkForm({ ...stkForm, amount_cents: Number(e.target.value) })}
                    className="border rounded-md px-3 py-1 w-32"
                    placeholder="Amount"
                  />
                  <button onClick={handleMpesaStk} className="bg-green-600 text-white px-3 py-1 rounded-md hover:bg-green-700">
                    Send
                  </button>
                </div>
              </div>

              {/* Manual payment */}
              <div className="flex gap-2">
                <select
                  value={paymentForm.channel}
                  onChange={(e) => setPaymentForm({ ...paymentForm, channel: e.target.value })}
                  className="border rounded-md px-3 py-1"
                >
                  {['cash', 'bank', 'cheque', 'mpesa'].map((c) => (
                    <option key={c} value={c}>{c}</option>
                  ))}
                </select>
                <input
                  type="number"
                  value={paymentForm.amount_cents}
                  onChange={(e) => setPaymentForm({ ...paymentForm, amount_cents: Number(e.target.value) })}
                  className="border rounded-md px-3 py-1 w-32"
                  placeholder="Amount"
                />
                <input
                  value={paymentForm.reference}
                  onChange={(e) => setPaymentForm({ ...paymentForm, reference: e.target.value })}
                  className="border rounded-md px-3 py-1 flex-1"
                  placeholder="Reference (optional)"
                />
                <button onClick={handleCreatePayment} className="bg-blue-600 text-white px-3 py-1 rounded-md hover:bg-blue-700">
                  <Wallet size={16} />
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}