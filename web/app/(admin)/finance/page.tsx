'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { Wallet, FileText, Receipt, BadgeDollarSign, ArrowRight } from 'lucide-react';
import { api, Invoice, Payment, FeeStructure } from '@/lib/api';

export default function FinanceOverviewPage() {
  const token = ''; // TODO: Get from auth context
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [payments, setPayments] = useState<Payment[]>([]);
  const [structures, setStructures] = useState<FeeStructure[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [invData, payData, feeData] = await Promise.all([
        api.listInvoices({}, token),
        api.listPayments({}, token),
        api.listFeeStructures({}, token),
      ]);
      setInvoices(invData);
      setPayments(payData);
      setStructures(feeData);
    } catch (e: any) {
      setError(e.message || 'Failed to load finance data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const completedPayments = payments.filter((p) => p.status === 'completed');
  const collected = completedPayments.reduce((s, p) => s + p.amount_cents, 0);
  const outstanding = invoices.reduce((s, i) => s + (i.balance_cents || 0), 0);
  const overdue = invoices.filter((i) => i.status === 'overdue').length;

  const stats = [
    { label: 'Collected (Term)', value: `KES ${(collected / 100).toLocaleString()}`, icon: BadgeDollarSign, color: 'text-green-600 bg-green-50' },
    { label: 'Outstanding', value: `KES ${(outstanding / 100).toLocaleString()}`, icon: Wallet, color: 'text-red-600 bg-red-50' },
    { label: 'Active Fee Structures', value: String(structures.length), icon: FileText, color: 'text-blue-600 bg-blue-50' },
    { label: 'Overdue Invoices', value: String(overdue), icon: Receipt, color: 'text-orange-600 bg-orange-50' },
  ];

  return (
    <div className="p-6">
      <div>
        <h1 className="text-2xl font-bold">Finance</h1>
        <p className="text-gray-500">Fee structures, invoices, and collections</p>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {loading ? (
        <p className="text-gray-500 mt-4">Loading...</p>
      ) : (
        <div className="mt-6 grid grid-cols-1 md:grid-cols-4 gap-4">
          {stats.map((s) => {
            const Icon = s.icon;
            return (
              <div key={s.label} className="bg-white rounded-lg shadow border p-4">
                <div className={`w-10 h-10 rounded-lg flex items-center justify-center mb-3 ${s.color}`}>
                  <Icon size={20} />
                </div>
                <p className="text-sm text-gray-500">{s.label}</p>
                <p className="text-2xl font-bold">{s.value}</p>
              </div>
            );
          })}
        </div>
      )}

      <div className="mt-8 grid grid-cols-1 md:grid-cols-3 gap-4">
        <Link
          href="/finance/fees"
          className="bg-white rounded-lg shadow border p-5 hover:border-blue-400 transition-colors"
        >
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold">Fee Structures</h3>
            <ArrowRight size={18} className="text-gray-400" />
          </div>
          <p className="text-sm text-gray-500">Per-grade fee schedules with tuition, transport, and activity items.</p>
        </Link>

        <Link
          href="/finance/invoices"
          className="bg-white rounded-lg shadow border p-5 hover:border-blue-400 transition-colors"
        >
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold">Invoices</h3>
            <ArrowRight size={18} className="text-gray-400" />
          </div>
          <p className="text-sm text-gray-500">Issue learner bills, apply discounts, and track fee balances.</p>
        </Link>

        <Link
          href="/finance/payments"
          className="bg-white rounded-lg shadow border p-5 hover:border-blue-400 transition-colors"
        >
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold">Payments</h3>
            <ArrowRight size={18} className="text-gray-400" />
          </div>
          <p className="text-sm text-gray-500">M-Pesa (Daraja), bank, cash, and cheque collections with reversal.</p>
        </Link>
      </div>
    </div>
  );
}