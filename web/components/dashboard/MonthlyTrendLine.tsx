'use client';

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { MonthlyCollectionTrend } from '@/lib/api';

interface Props {
  data: MonthlyCollectionTrend[];
}

export default function MonthlyTrendLine({ data }: Props) {
  if (!data || data.length === 0) {
    return <p className="text-gray-400 text-sm">No collection trend data for this filter.</p>;
  }

  const chartData = data.map((d) => ({
    month: d.month,
    amount: d.total_cents / 100,
    count: d.payment_count,
  }));

  return (
    <ResponsiveContainer width="100%" height={280}>
      <LineChart data={chartData}>
        <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
        <XAxis dataKey="month" tick={{ fontSize: 12 }} />
        <YAxis
          tick={{ fontSize: 12 }}
          tickFormatter={(v) => `KES ${(v / 1000).toFixed(0)}k`}
        />
        <Tooltip formatter={(value, name) => [
          name === 'amount' ? `KES ${Number(value || 0).toLocaleString()}` : String(value),
          name === 'amount' ? 'Amount' : 'Payments',
        ]} />
        <Line type="monotone" dataKey="amount" name="Amount" stroke="#2563eb" strokeWidth={2} dot={{ r: 4 }} />
      </LineChart>
    </ResponsiveContainer>
  );
}