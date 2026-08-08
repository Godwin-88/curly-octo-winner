'use client';

import { useEffect, useMemo, useState } from 'react';
import {
  Users,
  GraduationCap,
  MessageSquare,
  AlertTriangle,
  Activity,
  MapPin,
} from 'lucide-react';
import {
  api,
  SchoolOverview,
  AlertLearner,
  PaymentChannelBreakdown,
  MonthlyCollectionTrend,
  CompetencyDistribution,
  StrandCoverage,
  ChannelReach,
} from '@/lib/api';
import ChartCard from '@/components/dashboard/ChartCard';
import PaymentChannelPie from '@/components/dashboard/PaymentChannelPie';
import MonthlyTrendLine from '@/components/dashboard/MonthlyTrendLine';
import CompetencyBar from '@/components/dashboard/CompetencyBar';
import LearningAreaRadar from '@/components/dashboard/LearningAreaRadar';
import KenyaMap from '@/components/dashboard/KenyaMap';
import { PILOT_SCHOOL } from '@/lib/nairobi';

export default function DashboardPage() {
  const [token, setToken] = useState('');
  const [loading, setLoading] = useState(true);

  const [overview, setOverview] = useState<SchoolOverview | null>(null);
  const [atRisk, setAtRisk] = useState<AlertLearner[]>([]);
  const [channels, setChannels] = useState<PaymentChannelBreakdown[]>([]);
  const [trend, setTrend] = useState<MonthlyCollectionTrend[]>([]);
  const [distribution, setDistribution] = useState<CompetencyDistribution[]>([]);
  const [coverage, setCoverage] = useState<StrandCoverage[]>([]);
  const [channelReach, setChannelReach] = useState<ChannelReach[]>([]);
  const [failedEndpoints, setFailedEndpoints] = useState<string[]>([]);

  // Filters
  const [term, setTerm] = useState(1);
  const [year, setYear] = useState(2026);
  const [grade, setGrade] = useState('');
  const [stream, setStream] = useState('');
  const [county, setCounty] = useState('');
  const [subCounty, setSubCounty] = useState('');

  useEffect(() => {
    // Read token from localStorage (set on login)
    const t = typeof window !== 'undefined' ? localStorage.getItem('token') || '' : '';
    setToken(t);
  }, []);

  const load = async () => {
    if (!token) return;
    setLoading(true);

    const results = await Promise.allSettled([
      api.getSchoolOverview(token).then((v) => ({ k: 'overview' as const, v })),
      api.getAtRiskLearners({ term, year }, token).then((v) => ({ k: 'atRisk' as const, v })),
      api.getPaymentChannelBreakdown({ term, year }, token).then((v) => ({ k: 'channels' as const, v })),
      api.getMonthlyCollectionTrend(token).then((v) => ({ k: 'trend' as const, v })),
      api.getCompetencyDistribution({ grade, stream, term, year }, token).then((v) => ({ k: 'distribution' as const, v })),
      api.getStrandCoverage({ grade, stream, term, year }, token).then((v) => ({ k: 'coverage' as const, v })),
      api.getChannelReach(token).then((v) => ({ k: 'reach' as const, v })),
    ]);

    const failures: string[] = [];
    let ov: SchoolOverview | null = null;
    let risk: AlertLearner[] = [];
    let ch: PaymentChannelBreakdown[] = [];
    let tr: MonthlyCollectionTrend[] = [];
    let dist: CompetencyDistribution[] = [];
    let cov: StrandCoverage[] = [];
    let reach: ChannelReach[] = [];

    results.forEach((res) => {
      if (res.status === 'rejected') {
        failures.push(res.reason?.label ?? 'widget');
        return;
      }
      const data = res.value;
      switch (data.k) {
        case 'overview': ov = data.v; break;
        case 'atRisk': risk = data.v ?? []; break;
        case 'channels': ch = data.v ?? []; break;
        case 'trend': tr = data.v ?? []; break;
        case 'distribution': dist = data.v ?? []; break;
        case 'coverage': cov = data.v ?? []; break;
        case 'reach': reach = data.v ?? []; break;
      }
    });

    setOverview(ov);
    setAtRisk(risk);
    setChannels(ch);
    setTrend(tr);
    setDistribution(dist);
    setCoverage(cov);
    setChannelReach(reach);
    setFailedEndpoints(failures);
    setLoading(false);
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, term, year, grade, stream]);

  // Fallback: if school overview has no learner count but coverage implies learners, use it
  const learnerCount = overview?.learner_count ?? 0;

  // Aggregate learning areas for radar from strand coverage
  const radarData = useMemo(() => {
    const map = new Map<string, { total: number; count: number }>();
    coverage.forEach((c) => {
      const cur = map.get(c.learning_area) || { total: 0, count: 0 };
      cur.total += c.sub_strands_assessed > 0 ? 1 : 0;
      cur.count += 1;
      map.set(c.learning_area, cur);
    });
    return Array.from(map.entries()).map(([learning_area, v]) => ({
      learning_area,
      avg_rubric_level: v.count ? Number((v.total / v.count).toFixed(2)) : 0,
      assessment_count: v.total,
    }));
  }, [coverage]);

  const stats = [
    { label: 'Total Learners', value: learnerCount ? String(learnerCount) : '—', icon: Users, color: 'text-blue-600 bg-blue-50' },
    { label: 'At-Risk Learners', value: String(atRisk.length), icon: AlertTriangle, color: 'text-red-600 bg-red-50' },
    { label: 'Channels Reached', value: String(channelReach.length), icon: MessageSquare, color: 'text-green-600 bg-green-50' },
    { label: 'Strands Assessed', value: String(coverage.length), icon: GraduationCap, color: 'text-indigo-600 bg-indigo-50' },
  ];

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Dashboard</h1>
          <p className="text-gray-500">
            School overview · {PILOT_SCHOOL.name} ({PILOT_SCHOOL.subCounty}, {PILOT_SCHOOL.county})
          </p>
        </div>
        <div className="inline-flex items-center gap-1 px-3 py-1.5 bg-blue-50 text-blue-700 rounded-full text-sm">
          <MapPin size={14} />
          {subCounty ? `${subCounty}, ${county}` : county ? county : 'Kenya'}
        </div>
      </div>

      {failedEndpoints.length > 0 && (
        <div className="mt-4 p-3 bg-yellow-50 border border-yellow-200 text-yellow-800 rounded-md text-sm">
          Some widgets could not load: {failedEndpoints.join(', ')}. The rest of the dashboard is still shown.
        </div>
      )}

      {/* Filters */}
      <div className="mt-6 bg-white rounded-lg shadow-sm border p-4 flex flex-wrap items-end gap-4">
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
        <div>
          <label className="block text-sm text-gray-500 mb-1">Grade</label>
          <select value={grade} onChange={(e) => setGrade(e.target.value)} className="border rounded-md px-3 py-2 text-sm">
            <option value="">All</option>
            <option value="Grade 4">Grade 4</option>
            <option value="Grade 5">Grade 5</option>
            <option value="Grade 6">Grade 6</option>
          </select>
        </div>
        <div>
          <label className="block text-sm text-gray-500 mb-1">Stream</label>
          <select value={stream} onChange={(e) => setStream(e.target.value)} className="border rounded-md px-3 py-2 text-sm">
            <option value="">All</option>
            <option value="North">North</option>
            <option value="South">South</option>
          </select>
        </div>
      </div>

      {loading ? (
        <div className="mt-8 flex items-center justify-center py-20 text-gray-500">
          <Activity className="animate-spin mr-2" size={20} /> Loading dashboard...
        </div>
      ) : (
        <>
          {/* Summary tiles */}
          <div className="mt-6 grid grid-cols-1 md:grid-cols-4 gap-4">
            {stats.map((s) => {
              const Icon = s.icon;
              return (
                <div key={s.label} className="bg-white rounded-lg shadow-sm border border-gray-200 p-4">
                  <div className={`w-10 h-10 rounded-lg flex items-center justify-center mb-3 ${s.color}`}>
                    <Icon size={20} />
                  </div>
                  <p className="text-sm text-gray-500">{s.label}</p>
                  <p className="text-2xl font-bold">{s.value}</p>
                </div>
              );
            })}
          </div>

          {/* Charts grid */}
          <div className="mt-6 grid grid-cols-1 lg:grid-cols-2 gap-6">
            <ChartCard title="Fee Collection by Channel" subtitle="Breakdown of payments by channel (KES)">
              <PaymentChannelPie data={channels} />
            </ChartCard>

            <ChartCard title="Monthly Collection Trend" subtitle="Fee collection over time (KES)">
              <MonthlyTrendLine data={trend} />
            </ChartCard>

            <ChartCard title="Competency Distribution" subtitle="Learners by rubric level (1-4)">
              <CompetencyBar data={distribution} />
            </ChartCard>

            <ChartCard title="Learning Area Performance" subtitle="Average rubric level across learning areas">
              <LearningAreaRadar data={radarData} />
            </ChartCard>
          </div>

          {/* Kenya map - full width */}
          <div className="mt-6 bg-white rounded-lg shadow-sm border border-gray-200 p-6">
            <h2 className="text-lg font-semibold text-gray-900">Kenya School Map</h2>
            <p className="text-sm text-gray-500 mt-1">
              Drill down to your county and sub-county to focus the dashboard on your locale.
            </p>
            <div className="mt-4">
              <KenyaMap
                selectedSubCounty={subCounty}
                onSelect={(s) => {
                  setCounty('Nairobi');
                  setSubCounty(s);
                }}
              />
            </div>
          </div>
        </>
      )}
    </div>
  );
}