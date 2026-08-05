'use client';

import { useEffect, useState } from 'react';
import { Smartphone, CreditCard, CheckCircle, XCircle } from 'lucide-react';
import { api, Payment } from '@/lib/api';

const CHANNEL_STYLES: Record<string, string> = {
  mpesa: 'bg-green-100 text-green-700',
  bank: 'bg-blue-100 text-blue-700',
  cash: 'bg-yellow-100 text-yellow-700',
  cheque: 'bg-purple-100 text-purple-700',
};

const STATUS_STYLES: Record<string, string> = {
  pending: 'bg-yellow-100 text-yellow-700',
  completed: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
  reversed: 'bg-gray-100 text-gray-500',
};

export default function PaymentsPage() {
  const token = ''; // TODO: Get from auth context
  const [payments, setPayments] = useState<Payment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [channelFilter, setChannelFilter] = useState('');

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const data = await api.listPayments({
        status: statusFilter || undefined,
        channel: channelFilter || undefined,
      }, token);
      setPayments(data);
    } catch (e: any) {
      setError(e.message || 'Failed to load payments');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, statusFilter, channelFilter]);

  const handleReverse = async (id: string) => {
    setError('');
    try {
      await api.reversePayment(id, token);
      load();
    } catch (e: any) {
      setError(e.message || 'Failed to reverse payment');
    }
  };

  const completed = payments.filter((p) => p.status === 'completed');
  const stats = {
    completed: completed.length,
    pending: payments.filter((p) => p.status === 'pending').length,
    failed: payments.filter((p) => p.status === 'failed').length,
    collected: completed.reduce((s, p) => s + p.amount_cents, 0),
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Payments</h1>
          <p className="text-gray-500">All fee collections across channels</p>
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-white rounded-lg shadow border p-4">
          <p className="text-sm text-gray-500">Completed</p>
          <p className="text-2xl font-bold text-green-600">{stats.completed}</p>
        </div>
        <div className="bg-white rounded-lg shadow border p-4">
          <p className="text-sm text-gray-500">Pending (STK)</p>
          <p className="text-2xl font-bold text-yellow-600">{stats.pending}</p>
        </div>
        <div className="bg-white rounded-lg shadow border p-4">
          <p className="text-sm text-gray-500">Failed</p>
          <p className="text-2xl font-bold text-red-600">{stats.failed}</p>
        </div>
        <div className="bg-white rounded-lg shadow border p-4">
          <p className="text-sm text-gray-500">Collected</p>
          <p className="text-2xl font-bold text-blue-600">KES {(stats.collected / 100).toLocaleString()}</p>
        </div>
      </div>

      {error && <div className="mb-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      <div className="flex gap-2 mb-4">
        <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className="border rounded-md px-3 py-2">
          <option value="">All statuses</option>
          {Object.keys(STATUS_STYLES).map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
        <select value={channelFilter} onChange={(e) => setChannelFilter(e.target.value)} className="border rounded-md px-3 py-2">
          <option value="">All channels</option>
          {Object.keys(CHANNEL_STYLES).map((c) => (
            <option key={c} value={c}>{c}</option>
          ))}
        </select>
      </div>

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : payments.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          <CreditCard size={48} className="mx-auto mb-3 text-gray-300" />
          <p>No payments yet.</p>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow border overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-left">
              <tr>
                <th className="px-4 py-3">Learner</th>
                <th className="px-4 py-3">Invoice</th>
                <th className="px-4 py-3">Channel</th>
                <th className="px-4 py-3">Amount</th>
                <th className="px-4 py-3">Reference</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3">Date</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody>
              {payments.map((p) => (
                <tr key={p.id} className="border-t hover:bg-gray-50">
                  <td className="px-4 py-3 font-medium">{p.learner_name}</td>
                  <td className="px-4 py-3">{p.invoice_number}</td>
                  <td className="px-4 py-3">
                    <span className={`flex items-center gap-1 px-2 py-1 rounded-full text-xs inline-flex ${CHANNEL_STYLES[p.channel] || 'bg-gray-100'}`}>
                      {p.channel === 'mpesa' && <Smartphone size={12} />}
                      {p.channel}
                    </span>
                  </td>
                  <td className="px-4 py-3 font-semibold">KES {(p.amount_cents / 100).toLocaleString()}</td>
                  <td className="px-4 py-3 text-gray-500">{p.mpesa_receipt || p.reference || '-'}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-1 rounded-full text-xs ${STATUS_STYLES[p.status] || 'bg-gray-100'}`}>
                      {p.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-500">
                    {p.paid_at ? new Date(p.paid_at).toLocaleDateString() : new Date(p.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3">
                    {p.status === 'completed' && (
                      <button onClick={() => handleReverse(p.id)} className="text-red-500 hover:text-red-700 text-xs">
                        Reverse
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="mt-4 flex items-center gap-4 text-sm text-gray-500">
        <span className="flex items-center gap-1"><CheckCircle size={14} className="text-green-500" /> Completed = confirmed in ledger</span>
        <span className="flex items-center gap-1"><XCircle size={14} className="text-red-500" /> Failed = M-Pesa declined</span>
      </div>
    </div>
  );
}