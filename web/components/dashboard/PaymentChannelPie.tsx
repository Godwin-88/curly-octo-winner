'use client';

import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import { PaymentChannelBreakdown } from '@/lib/api';

const COLORS = ['#2563eb', '#16a34a', '#f59e0b', '#dc2626', '#7c3aed'];

interface Props {
  data: PaymentChannelBreakdown[];
}

export default function PaymentChannelPie({ data }: Props) {
  if (!data || data.length === 0) {
    return <p className="text-gray-400 text-sm">No payment data for this filter.</p>;
  }

  const chartData = data.map((d) => ({
    name: d.channel.charAt(0).toUpperCase() + d.channel.slice(1),
    value: d.total_cents / 100,
  }));

  return (
    <ResponsiveContainer width="100%" height={280}>
      <PieChart>
        <Pie
          data={chartData}
          dataKey="value"
          nameKey="name"
          cx="50%"
          cy="50%"
          outerRadius={100}
          label={(entry) => `${entry.name}`}
        >
          {chartData.map((_, i) => (
            <Cell key={i} fill={COLORS[i % COLORS.length]} />
          ))}
        </Pie>
        <Tooltip formatter={(value) => `KES ${Number(value || 0).toLocaleString()}`} />
        <Legend />
      </PieChart>
    </ResponsiveContainer>
  );
}