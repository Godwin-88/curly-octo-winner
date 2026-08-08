'use client';

import { useMemo, useState } from 'react';
import { NAIROBI_BOUNDARY, NAIROBI_SUB_COUNTIES, PILOT_SCHOOL } from '@/lib/nairobi';

interface Props {
  selectedSubCounty?: string;
  onSelect: (subCounty: string) => void;
}

// Equirectangular projection: lng -> x, lat -> y, scaled to fit a viewBox.
function project(boundary: [number, number][]) {
  const lngs = boundary.map((p) => p[0]);
  const lats = boundary.map((p) => p[1]);
  const minLng = Math.min(...lngs);
  const maxLng = Math.max(...lngs);
  const minLat = Math.min(...lats);
  const maxLat = Math.max(...lats);
  const W = 600;
  const H = 500;
  const pad = 30;
  const scale = Math.min((W - pad * 2) / (maxLng - minLng), (H - pad * 2) / (maxLat - minLat));
  const cx = (minLng + maxLng) / 2;
  const cy = (minLat + maxLat) / 2;
  const toXY = (lng: number, lat: number) => ({
    x: (lng - cx) * scale + W / 2,
    y: (cy - lat) * scale + H / 2,
  });
  return { toXY, W, H };
}

export default function KenyaMap({ selectedSubCounty, onSelect }: Props) {
  const [hovered, setHovered] = useState<string | null>(null);
  const { toXY, W, H } = useMemo(() => project(NAIROBI_BOUNDARY), []);

  const pathD = useMemo(() => {
    return (
      NAIROBI_BOUNDARY.map((p, i) => {
        const { x, y } = toXY(p[0], p[1]);
        return `${i === 0 ? 'M' : 'L'} ${x.toFixed(1)} ${y.toFixed(1)}`;
      }).join(' ') + ' Z'
    );
  }, [toXY]);

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <span className="text-sm font-medium">Nairobi County</span>
        <span className="text-xs text-gray-500">Pilot school: {PILOT_SCHOOL.subCounty}</span>
      </div>

      <svg viewBox={`0 0 ${W} ${H}`} className="w-full h-auto max-h-[420px]">
        <path
          d={pathD}
          fill="#dbeafe"
          stroke="#2563eb"
          strokeWidth={2}
          className="transition-opacity"
        />

        {NAIROBI_SUB_COUNTIES.map((sub) => {
          const { x, y } = toXY(sub.lng, sub.lat);
          const isPilot = sub.name === PILOT_SCHOOL.subCounty;
          const isSel = selectedSubCounty === sub.name;
          return (
            <g
              key={sub.name}
              className="cursor-pointer"
              onMouseEnter={() => setHovered(sub.name)}
              onMouseLeave={() => setHovered(null)}
              onClick={() => onSelect(sub.name)}
            >
              <circle
                cx={x}
                cy={y}
                r={isPilot ? 8 : 6}
                fill={isSel ? '#2563eb' : isPilot ? '#dc2626' : '#3b82f6'}
                stroke="#ffffff"
                strokeWidth={2}
              />
              <text
                x={x}
                y={y - 10}
                textAnchor="middle"
                fontSize={10}
                fill="#1f2937"
                fontWeight={isPilot ? 700 : 500}
              >
                {sub.name}
              </text>
            </g>
          );
        })}

        {hovered && (
          <g>
            <rect x={W / 2 - 60} y={H - 40} width={120} height={24} rx={4} fill="#111827" opacity={0.9} />
            <text x={W / 2} y={H - 24} textAnchor="middle" fontSize={12} fill="#ffffff">
              {hovered}
            </text>
          </g>
        )}
      </svg>

      <p className="text-xs text-gray-500 mt-2">
        Click a sub-county to focus the dashboard on that locale. {PILOT_SCHOOL.name} is in{' '}
        {PILOT_SCHOOL.subCounty}, {PILOT_SCHOOL.county}.
      </p>
    </div>
  );
}
