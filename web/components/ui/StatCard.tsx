// Standard KPI stat card — replaces the per-page icon-card copy-paste.

import type { LucideIcon } from 'lucide-react';

const TONES = {
  blue: 'bg-blue-100 text-blue-700',
  green: 'bg-green-100 text-green-700',
  yellow: 'bg-yellow-100 text-yellow-700',
  red: 'bg-red-100 text-red-700',
  indigo: 'bg-indigo-100 text-indigo-700',
  gray: 'bg-gray-100 text-gray-700',
} as const;

export function StatCard({
  label,
  value,
  icon: Icon,
  tone = 'blue',
  hint,
}: {
  label: string;
  value: string | number;
  icon: LucideIcon;
  tone?: keyof typeof TONES;
  hint?: string;
}) {
  return (
    <div className="card p-4">
      <div className="flex items-center gap-3">
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${TONES[tone]}`} aria-hidden="true">
          <Icon size={20} />
        </div>
        <div className="min-w-0">
          <p className="text-sm text-gray-600">{label}</p>
          <p className="text-xl font-bold text-gray-900 truncate">{value}</p>
          {hint && <p className="text-xs text-gray-500">{hint}</p>}
        </div>
      </div>
    </div>
  );
}
