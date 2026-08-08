'use client';

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { CompetencyDistribution } from '@/lib/api';

const LEVEL_COLORS = ['#dc2626', '#f59e0b', '#16a34a', '#2563eb'];

interface Props {
  data: CompetencyDistribution[];
}

export default function CompetencyBar({ data }: Props) {
  if (!data || data.length === 0) {
    return <p className="text-gray-400 text-sm">No competency data for this filter.</p>;
  }

  // Aggregate by rubric level 1-4
  const chartData = [1, 2, 3, 4].map((level) => ({
    level: `Level ${level}`,
    learners: data
      .filter((d) => d.rubric_level === level)
      .reduce((s, d) => s + d.learner_count, 0),
  }));

  return (
    <ResponsiveContainer width="100%" height={280}>
      <BarChart data={chartData}>
        <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
        <XAxis dataKey="level" tick={{ fontSize: 12 }} />
        <YAxis tick={{ fontSize: 12 }} allowDecimals={false} />
        <Tooltip formatter={(value) => [`${value} learners`, '']} />
        <Bar dataKey="learners" name="Learners" radius={[4, 4, 0, 0]}>
          {chartData.map((_, i) => (
            <Cell key={i} fill={LEVEL_COLORS[i]} />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  );
}