'use client';

import { RadarChart, PolarGrid, PolarAngleAxis, PolarRadiusAxis, Radar, ResponsiveContainer, Tooltip } from 'recharts';

interface Props {
  data: { learning_area: string; avg_rubric_level: number; assessment_count: number }[];
}

export default function LearningAreaRadar({ data }: Props) {
  if (!data || data.length === 0) {
    return <p className="text-gray-400 text-sm">No learning area performance data for this filter.</p>;
  }

  const chartData = data.map((d) => ({
    subject: d.learning_area,
    average: d.avg_rubric_level,
    fullMark: 4,
  }));

  return (
    <ResponsiveContainer width="100%" height={300}>
      <RadarChart data={chartData}>
        <PolarGrid stroke="#e5e7eb" />
        <PolarAngleAxis dataKey="subject" tick={{ fontSize: 11 }} />
        <PolarRadiusAxis angle={30} domain={[0, 4]} tick={{ fontSize: 10 }} />
        <Radar
          name="Avg Rubric Level"
          dataKey="average"
          stroke="#2563eb"
          fill="#2563eb"
          fillOpacity={0.2}
        />
        <Tooltip formatter={(value) => [Number(value).toFixed(2), 'Average Rubric Level']} />
      </RadarChart>
    </ResponsiveContainer>
  );
}