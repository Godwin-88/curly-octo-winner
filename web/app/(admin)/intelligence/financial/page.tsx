'use client';

import { useEffect, useState } from 'react';
import { Wallet, TrendingUp, AlertTriangle, CreditCard } from 'lucide-react';
import { api, FeeCollectionSummary, PaymentChannelBreakdown, FeeDefaulter, MonthlyCollectionTrend } from '@/lib/api';

export default function FinancialAnalyticsPage() {
  const token = ''; // TODO: Get from auth context
  const [feeSummary, setFeeSummary] = useState<FeeCollectionSummary[]>([]);
  const [channels, setChannels] = useState<PaymentChannelBreakdown[]>([]);
  const [defaulters, setDefaulters] = useState<FeeDefaulter[]>([]);
  const [trend, setTrend] = useState<MonthlyCollectionTrend[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [term, setTerm] = useState(1);
  const [year, setYear] = useState(2026);

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [fee, ch, def, tr] = await Promise.all([
        api.getFeeCollectionSummary({ term, year }, token),
        api.getPaymentChannelBreakdown({ term, year }, token),
        api.getFeeDefaulters({ term, year }, token),
        api.getMonthlyCollectionTrend(token),
      ]);
      setFeeSummary(fee);
      setChannels(ch);
      setDefaulters(def);
      setTrend(tr);
    } catch (e: any) {
      setError(e.message || 'Failed to load financial analytics');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, term, year]);

  const current = feeSummary.find((f) => f.term === term && f.year === year);
  const totalCollected = current?.total_collected_cents || 0;
  const totalOutstanding = current?.outstanding_cents || 0;
  const collectionRate = current?.collection_rate || 0;
  const totalBilled = current?.total_billed_cents || 0;

  const stats = [
    { label: 'Total Billed', value: `KES ${(totalBilled / 100).toLocaleString()}`, icon: Wallet, color: 'text-blue-600 bg-blue-50' },
    { label: 'Collected', value: `KES ${(totalCollected / 100).toLocaleString()}`, icon: CreditCard, color: 'text-green-600 bg-green-50' },
    { label: 'Outstanding', value: `KES ${(totalOutstanding / 100).toLocaleString()}`, icon: TrendingUp, color: 'text-red-600 bg-red-50' },
    { label: 'Collection Rate', value: `${collectionRate.toFixed(1)}%`, icon: AlertTriangle, color: 'text-indigo-600 bg-indigo-50' },
  ];

  const maxTrend = Math.max(...trend.map((t) => t.total_cents), 1);

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Financial Analytics</h1>
          <p className="text-gray-500">Fee collection, payment channels, and defaulters</p>
        </div>
      </div>

      {/* Filters */}
      <div className="mt-6 bg-white rounded-lg shadow border p-4 flex flex-wrap items-end gap-4">
        <div>
          <label className="block text-sm text-gray-500 mb-1">Term</label>
          <select value={term} onChange={(e) => setTerm(Number(e.target.value))} className="border rounded-md px-3 py-2 text-sm">
            <option value={1}>Term 1</option>
            <option value={2}>Term 2</option>
            <option value={3}>Term 3</option>
          </select>
        </div>
        <div>
          <label className="block text-sm text-gray-500 mb-1">Year</label>
          <input type="number" value={year} onChange={(e) => setYear(Number(e.target.value))} className="border rounded-md px-3 py-2 text-sm w-24" />
        </div>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {loading ? (
        <p className="text-gray-500 mt-4">Loading...</p>
      ) : (
        <>
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

          {/* Monthly collection trend */}
          <div className="mt-8 bg-white rounded-lg shadow border p-6">
            <h2 className="text-lg font-semibold mb-4">Monthly Collection Trend</h2>
            {trend.length === 0 ? (
              <p className="text-gray-400 text-sm">No collection data available.</p>
            ) : (
              <div className="flex items-end gap-2 h-40">
                {trend.map((t) => (
                  <div key={t.month} className="flex-1 flex flex-col items-center gap-1">
                    <span className="text-xs text-gray-500">{(t.total_cents / 100).toLocaleString()}</span>
                    <div
                      className="w-full bg-blue-500 rounded-t"
                      style={{ height: `${(t.total_cents / maxTrend) * 100}%`, minHeight: '4px' }}
                    />
                    <span className="text-xs text-gray-500">{new Date(t.month).toLocaleDateString(undefined, { month: 'short' })}</span>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Payment channels */}
          <div className="mt-8 bg-white rounded-lg shadow border overflow-hidden">
            <div className="p-6 pb-0">
              <h2 className="text-lg font-semibold">Payment Channel Breakdown</h2>
              <p className="text-sm text-gray-500">Amount collected by channel</p>
            </div>
            <div className="p-6">
              {channels.length === 0 ? (
                <p className="text-gray-400 text-sm">No payment channel data for this filter.</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-left text-gray-500">
                    <tr>
                      <th className="px-3 py-2">Channel</th>
                      <th className="px-3 py-2">Payments</th>
                      <th className="px-3 py-2">Total</th>
                    </tr>
                  </thead>
                  <tbody>
                    {channels.map((c) => (
                      <tr key={c.channel} className="border-t">
                        <td className="px-3 py-2 uppercase font-medium">{c.channel}</td>
                        <td className="px-3 py-2">{c.payment_count}</td>
                        <td className="px-3 py-2">KES {(c.total_cents / 100).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>

          {/* Fee defaulters */}
          <div className="mt-8 bg-white rounded-lg shadow border overflow-hidden">
            <div className="p-6 pb-0">
              <h2 className="text-lg font-semibold">Fee Defaulters</h2>
              <p className="text-sm text-gray-500">Learners with outstanding balances</p>
            </div>
            <div className="p-6">
              {defaulters.length === 0 ? (
                <p className="text-gray-400 text-sm">No defaulters for this filter.</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-left text-gray-500">
                    <tr>
                      <th className="px-3 py-2">Learner</th>
                      <th className="px-3 py-2">Grade</th>
                      <th className="px-3 py-2">Invoice</th>
                      <th className="px-3 py-2">Balance</th>
                      <th className="px-3 py-2">Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {defaulters.slice(0, 20).map((d) => (
                      <tr key={d.invoice_number} className="border-t">
                        <td className="px-3 py-2 font-medium">{d.learner_name}</td>
                        <td className="px-3 py-2">{d.grade} {d.stream}</td>
                        <td className="px-3 py-2">{d.invoice_number}</td>
                        <td className="px-3 py-2 text-red-600 font-medium">KES {(d.balance_cents / 100).toLocaleString()}</td>
                        <td className="px-3 py-2">
                          <span className={`px-2 py-1 rounded-full text-xs ${d.status === 'overdue' ? 'bg-red-50 text-red-700' : 'bg-yellow-50 text-yellow-700'}`}>
                            {d.status}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}