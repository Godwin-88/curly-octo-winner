'use client';

import { useEffect, useState } from 'react';
import { API_BASE } from '@/lib/api';
import { useAuth } from '@/lib/auth';

interface InvoiceBrief {
  id: string;
  learner_id: string;
  learner_name: string;
  invoice_number: string;
  term: number;
  year: number;
  total_cents: number;
  paid_cents: number;
  balance_cents: number;
  status: string;
  due_date?: string;
}

export default function ParentFeesPage() {
  const [invoices, setInvoices] = useState<InvoiceBrief[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const { guardianToken: token } = useAuth();

  useEffect(() => {
    if (!token) return;
    fetch(`${API_BASE}/parent/fees`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error('Failed to load invoices'))))
      .then(setInvoices)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [token]);

  const formatKES = (cents: number) => {
    return `KES ${(cents / 100).toFixed(2)}`;
  };

  if (!token || loading) {
    return <div className="text-center py-12 text-gray-500">Loading...</div>;
  }

  if (error) {
    return <div className="bg-red-50 text-red-700 p-3 rounded-md text-sm">{error}</div>;
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Fees & Payments</h1>
      {invoices.length === 0 ? (
        <p className="text-gray-500">No invoices found.</p>
      ) : (
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Invoice</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Learner</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Term</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Total</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Balance</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {invoices.map((inv) => (
                <tr key={inv.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm">{inv.invoice_number}</td>
                  <td className="px-4 py-3 text-sm">{inv.learner_name}</td>
                  <td className="px-4 py-3 text-sm">T{inv.term} {inv.year}</td>
                  <td className="px-4 py-3 text-sm">{formatKES(inv.total_cents)}</td>
                  <td className="px-4 py-3 text-sm font-medium text-red-600">{formatKES(inv.balance_cents)}</td>
                  <td className="px-4 py-3 text-sm">
                    <span className={`px-2 py-1 rounded-full text-xs ${
                      inv.status === 'paid' ? 'bg-green-100 text-green-800' :
                      inv.status === 'overdue' ? 'bg-red-100 text-red-800' :
                      'bg-yellow-100 text-yellow-800'
                    }`}>
                      {inv.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
