'use client';

import { useEffect, useState } from 'react';
import { AlertTriangle, Users, BookOpen, ClipboardList } from 'lucide-react';
import { api, StrandCoverage, CompetencyDistribution, TeacherVelocity, LearnerPortfolio, AlertLearner, SchoolOverview } from '@/lib/api';

export default function AnalyticsDashboardPage() {
  const token = ''; // TODO: Get from auth context
  const [overview, setOverview] = useState<SchoolOverview | null>(null);
  const [coverage, setCoverage] = useState<StrandCoverage[]>([]);
  const [distribution, setDistribution] = useState<CompetencyDistribution[]>([]);
  const [velocity, setVelocity] = useState<TeacherVelocity[]>([]);
  const [portfolio, setPortfolio] = useState<LearnerPortfolio[]>([]);
  const [atRisk, setAtRisk] = useState<AlertLearner[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [term, setTerm] = useState(1);
  const [year, setYear] = useState(2026);
  const [grade, setGrade] = useState('');
  const [stream, setStream] = useState('');

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [ov, cov, dist, vel, port, risk] = await Promise.all([
        api.getSchoolOverview(token),
        api.getStrandCoverage({ grade, stream, term, year }, token),
        api.getCompetencyDistribution({ grade, stream, term, year }, token),
        api.getTeacherVelocity({ term, year }, token),
        api.getLearnerPortfolio({ grade, stream, term, year }, token),
        api.getAtRiskLearners({ term, year }, token),
      ]);
      setOverview(ov);
      setCoverage(cov);
      setDistribution(dist);
      setVelocity(vel);
      setPortfolio(port);
      setAtRisk(risk);
    } catch (e: any) {
      setError(e.message || 'Failed to load analytics');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, term, year, grade, stream]);

  const rubricColor = (level: number) => {
    if (level === 1) return 'bg-red-50 text-red-700';
    if (level === 2) return 'bg-yellow-50 text-yellow-700';
    if (level === 3) return 'bg-green-50 text-green-700';
    return 'bg-blue-50 text-blue-700';
  };

  // Aggregate distribution by rubric level across all strands
  const distByLevel = [1, 2, 3, 4].map((level) => ({
    level,
    count: distribution.filter((d) => d.rubric_level === level).reduce((s, d) => s + d.learner_count, 0),
  }));
  const totalDist = distByLevel.reduce((s, d) => s + d.count, 0);

  const stats = [
    { label: 'Learners', value: overview ? String(overview.learner_count) : '—', icon: Users, color: 'text-blue-600 bg-blue-50' },
    { label: 'Strands Assessed', value: String(coverage.length), icon: BookOpen, color: 'text-green-600 bg-green-50' },
    { label: 'Assessments', value: String(velocity.reduce((s, v) => s + v.assessment_count, 0)), icon: ClipboardList, color: 'text-indigo-600 bg-indigo-50' },
    { label: 'At-Risk', value: String(atRisk.length), icon: AlertTriangle, color: 'text-red-600 bg-red-50' },
  ];

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Analytics Dashboard</h1>
          <p className="text-gray-500">CBC learning analytics and progress monitoring</p>
        </div>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

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

          {/* Competency distribution */}
          <div className="mt-8 bg-white rounded-lg shadow border p-6">
            <h2 className="text-lg font-semibold mb-4">Competency Distribution</h2>
            {totalDist === 0 ? (
              <p className="text-gray-400 text-sm">No assessment data for this filter.</p>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                {distByLevel.map((d) => (
                  <div key={d.level} className={`rounded-lg p-4 ${rubricColor(d.level)}`}>
                    <p className="text-sm font-medium">Level {d.level}</p>
                    <p className="text-2xl font-bold">{d.count}</p>
                    <p className="text-xs opacity-75">{totalDist ? ((d.count / totalDist) * 100).toFixed(1) : 0}% of learners</p>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Strand coverage heatmap */}
          <div className="mt-8 bg-white rounded-lg shadow border overflow-hidden">
            <div className="p-6 pb-0">
              <h2 className="text-lg font-semibold">Strand Coverage Heatmap</h2>
              <p className="text-sm text-gray-500">Which strands have been assessed per class</p>
            </div>
            <div className="p-6">
              {coverage.length === 0 ? (
                <p className="text-gray-400 text-sm">No strand coverage data for this filter.</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-left text-gray-500">
                    <tr>
                      <th className="px-3 py-2">Class</th>
                      <th className="px-3 py-2">Learning Area</th>
                      <th className="px-3 py-2">Strand</th>
                      <th className="px-3 py-2">Sub-Strands</th>
                      <th className="px-3 py-2">Learners</th>
                    </tr>
                  </thead>
                  <tbody>
                    {coverage.map((c, i) => (
                      <tr key={i} className="border-t">
                        <td className="px-3 py-2">{c.grade} {c.stream}</td>
                        <td className="px-3 py-2">{c.learning_area}</td>
                        <td className="px-3 py-2">{c.strand_name}</td>
                        <td className="px-3 py-2">
                          <span className={`px-2 py-1 rounded-full text-xs ${c.sub_strands_assessed > 0 ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}>
                            {c.sub_strands_assessed}
                          </span>
                        </td>
                        <td className="px-3 py-2">{c.learners_assessed}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>

          {/* Teacher velocity */}
          <div className="mt-8 bg-white rounded-lg shadow border overflow-hidden">
            <div className="p-6 pb-0">
              <h2 className="text-lg font-semibold">Teacher Assessment Velocity</h2>
              <p className="text-sm text-gray-500">Assessments recorded per teacher per week</p>
            </div>
            <div className="p-6">
              {velocity.length === 0 ? (
                <p className="text-gray-400 text-sm">No teacher velocity data for this filter.</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-left text-gray-500">
                    <tr>
                      <th className="px-3 py-2">Teacher</th>
                      <th className="px-3 py-2">Week</th>
                      <th className="px-3 py-2">Assessments</th>
                    </tr>
                  </thead>
                  <tbody>
                    {velocity.slice(0, 20).map((v, i) => (
                      <tr key={i} className="border-t">
                        <td className="px-3 py-2 font-medium">{v.teacher_name}</td>
                        <td className="px-3 py-2">{new Date(v.week_start).toLocaleDateString()}</td>
                        <td className="px-3 py-2">{v.assessment_count}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>

          {/* Learner portfolio */}
          <div className="mt-8 bg-white rounded-lg shadow border overflow-hidden">
            <div className="p-6 pb-0">
              <h2 className="text-lg font-semibold">Learner Portfolio</h2>
              <p className="text-sm text-gray-500">Per-learner rubric averages and attendance</p>
            </div>
            <div className="p-6">
              {portfolio.length === 0 ? (
                <p className="text-gray-400 text-sm">No learner portfolio data for this filter.</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-left text-gray-500">
                    <tr>
                      <th className="px-3 py-2">Learner</th>
                      <th className="px-3 py-2">Grade</th>
                      <th className="px-3 py-2">Stream</th>
                      <th className="px-3 py-2">Areas</th>
                      <th className="px-3 py-2">Avg Rubric</th>
                      <th className="px-3 py-2">Attendance</th>
                    </tr>
                  </thead>
                  <tbody>
                    {portfolio.slice(0, 20).map((p) => (
                      <tr key={p.learner_id} className="border-t">
                        <td className="px-3 py-2 font-medium">{p.learner_name}</td>
                        <td className="px-3 py-2">{p.grade}</td>
                        <td className="px-3 py-2">{p.stream}</td>
                        <td className="px-3 py-2">{p.learning_areas_assessed}</td>
                        <td className="px-3 py-2">
                          <span className={`px-2 py-1 rounded-full text-xs ${p.overall_avg_rubric < 2.5 ? 'bg-red-50 text-red-700' : p.overall_avg_rubric < 3 ? 'bg-yellow-50 text-yellow-700' : 'bg-green-50 text-green-700'}`}>
                            {p.overall_avg_rubric.toFixed(2)}
                          </span>
                        </td>
                        <td className="px-3 py-2">
                          <span className={`px-2 py-1 rounded-full text-xs ${p.attendance_rate < 75 ? 'bg-red-50 text-red-700' : 'bg-green-50 text-green-700'}`}>
                            {p.attendance_rate.toFixed(1)}%
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