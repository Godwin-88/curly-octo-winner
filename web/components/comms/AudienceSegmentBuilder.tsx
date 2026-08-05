'use client';

import { ReachEstimate } from '@/lib/api';

interface Props {
  audienceType: string;
  setAudienceType: (type: string) => void;
  audienceFilter: Record<string, unknown>;
  setAudienceFilter: (filter: Record<string, unknown>) => void;
  onEstimate: () => void;
  estimate: ReachEstimate | null;
}

const AUDIENCE_TYPES = [
  { value: 'all_parents', label: 'All Parents' },
  { value: 'grade', label: 'By Grade' },
  { value: 'stream', label: 'By Grade & Stream' },
  { value: 'transport', label: 'Transport Enrolled' },
  { value: 'fee_defaulters', label: 'Fee Defaulters' },
  { value: 'custom', label: 'Custom Selection' },
];

const GRADES = ['Grade 4', 'Grade 5', 'Grade 6'];
const STREAMS = ['North', 'South'];

export default function AudienceSegmentBuilder({
  audienceType,
  setAudienceType,
  audienceFilter,
  setAudienceFilter,
  onEstimate,
  estimate,
}: Props) {
  return (
    <div className="card p-6 max-w-2xl">
      <h2 className="text-lg font-semibold mb-4">Select Audience</h2>

      <div className="space-y-4">
        <div>
          <label className="label">Audience Type</label>
          <select
            className="input"
            value={audienceType}
            onChange={(e) => {
              setAudienceType(e.target.value);
              setAudienceFilter({});
            }}
          >
            {AUDIENCE_TYPES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </div>

        {audienceType === 'grade' && (
          <div>
            <label className="label">Grade</label>
            <select
              className="input"
              value={(audienceFilter.grade as string) || ''}
              onChange={(e) => setAudienceFilter({ ...audienceFilter, grade: e.target.value })}
            >
              <option value="">Select grade</option>
              {GRADES.map((g) => (
                <option key={g} value={g}>
                  {g}
                </option>
              ))}
            </select>
          </div>
        )}

        {audienceType === 'stream' && (
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="label">Grade</label>
              <select
                className="input"
                value={(audienceFilter.grade as string) || ''}
                onChange={(e) => setAudienceFilter({ ...audienceFilter, grade: e.target.value })}
              >
                <option value="">Select grade</option>
                {GRADES.map((g) => (
                  <option key={g} value={g}>
                    {g}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="label">Stream</label>
              <select
                className="input"
                value={(audienceFilter.stream as string) || ''}
                onChange={(e) => setAudienceFilter({ ...audienceFilter, stream: e.target.value })}
              >
                <option value="">Select stream</option>
                {STREAMS.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </select>
            </div>
          </div>
        )}

        {audienceType === 'custom' && (
          <div>
            <label className="label">Guardian IDs (comma-separated)</label>
            <textarea
              className="input"
              rows={3}
              placeholder="c0000000-0000-0000-0000-000000000001, c0000000-0000-0000-0000-000000000002"
              onChange={(e) => {
                const ids = e.target.value
                  .split(',')
                  .map((s) => s.trim())
                  .filter(Boolean);
                setAudienceFilter({ ...audienceFilter, guardian_ids: ids });
              }}
            />
          </div>
        )}

        <button className="btn-secondary" onClick={onEstimate}>
          Estimate Reach
        </button>

        {estimate && (
          <div className="bg-blue-50 border border-blue-200 rounded-md p-4">
            <div className="flex justify-between">
              <span className="text-sm text-gray-600">Recipients</span>
              <span className="font-semibold">{estimate.recipient_count}</span>
            </div>
            <div className="flex justify-between mt-2">
              <span className="text-sm text-gray-600">Estimated Cost</span>
              <span className="font-semibold">KES {estimate.estimated_kes.toFixed(2)}</span>
            </div>
            <div className="flex justify-between mt-2">
              <span className="text-sm text-gray-600">SMS Units</span>
              <span className="font-semibold">{estimate.sms_units}</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}