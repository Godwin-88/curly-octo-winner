'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import { FileText, BarChart3, AlertTriangle, ArrowRight, Users } from 'lucide-react';
import { api, ReportCard, AlertLearner, SchoolOverview } from '@/lib/api';

export default function ReportsOverviewPage() {
  const token = ''; // TODO: Get from auth context
  const [cards, setCards] = useState<ReportCard[]>([]);
  const [atRisk, setAtRisk] = useState<AlertLearner[]>([]);
  const [overview, setOverview] = useState<SchoolOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [cardData, riskData, ovData] = await Promise.all([
        api.listReportCards({}, token),
        api.getAtRiskLearners({}, token),
        api.getSchoolOverview(token),
      ]);
      setCards(cardData);
      setAtRisk(riskData);
      setOverview(ovData);
    } catch (e: any) {
      setError(e.message || 'Failed to load reports data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const finalCards = cards.filter((c) => c.status === 'final');
  const draftCards = cards.filter((c) => c.status === 'draft');

  const stats = [
    { label: 'Learners', value: overview ? String(overview.learner_count) : '—', icon: Users, color: 'text-blue-600 bg-blue-50' },
    { label: 'Report Cards', value: String(cards.length), icon: FileText, color: 'text-green-600 bg-green-50' },
    { label: 'Finalized', value: String(finalCards.length), icon: FileText, color: 'text-indigo-600 bg-indigo-50' },
    { label: 'At-Risk Learners', value: String(atRisk.length), icon: AlertTriangle, color: 'text-red-600 bg-red-50' },
  ];

  return (
    <div className="p-6">
      <div>
        <h1 className="text-2xl font-bold">Reports & Analytics</h1>
        <p className="text-gray-500">CBC report cards and learning analytics dashboards</p>
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

      <div className="mt-8 grid grid-cols-1 md:grid-cols-2 gap-4">
        <Link
          href="/reports/cards"
          className="bg-white rounded-lg shadow border p-5 hover:border-blue-400 transition-colors"
        >
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold">Report Cards</h3>
            <ArrowRight size={18} className="text-gray-400" />
          </div>
          <p className="text-sm text-gray-500">
            Generate CBC-compliant report cards with per-strand ratings, core competency remarks, and attendance.
          </p>
        </Link>

        <Link
          href="/analytics"
          className="bg-white rounded-lg shadow border p-5 hover:border-blue-400 transition-colors"
        >
          <div className="flex items-center justify-between mb-3">
            <h3 className="font-semibold">Analytics Dashboard</h3>
            <ArrowRight size={18} className="text-gray-400" />
          </div>
          <p className="text-sm text-gray-500">
            Strand coverage heatmap, competency distribution, teacher velocity, and at-risk learner radar.
          </p>
        </Link>
      </div>

      {atRisk.length > 0 && (
        <div className="mt-8">
          <h2 className="text-lg font-semibold mb-3 flex items-center gap-2">
            <AlertTriangle size={18} className="text-red-500" />
            At-Risk Learners
          </h2>
          <div className="bg-white rounded-lg shadow border overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-left text-gray-500">
                <tr>
                  <th className="px-4 py-3">Learner</th>
                  <th className="px-4 py-3">Grade</th>
                  <th className="px-4 py-3">Stream</th>
                  <th className="px-4 py-3">Avg Rubric</th>
                  <th className="px-4 py-3">Attendance</th>
                </tr>
              </thead>
              <tbody>
                {atRisk.slice(0, 5).map((l) => (
                  <tr key={l.learner_id} className="border-t">
                    <td className="px-4 py-3 font-medium">{l.learner_name}</td>
                    <td className="px-4 py-3">{l.grade}</td>
                    <td className="px-4 py-3">{l.stream}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded-full text-xs ${l.overall_avg_rubric < 2.5 ? 'bg-red-50 text-red-700' : 'bg-yellow-50 text-yellow-700'}`}>
                        {l.overall_avg_rubric.toFixed(2)}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-1 rounded-full text-xs ${l.attendance_rate < 75 ? 'bg-red-50 text-red-700' : 'bg-yellow-50 text-yellow-700'}`}>
                        {l.attendance_rate.toFixed(1)}%
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}