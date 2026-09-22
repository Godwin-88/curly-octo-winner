'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ClipboardList } from 'lucide-react';
import { API_BASE } from '@/lib/api';
import { useAuth } from '@/lib/auth';

interface AssessmentItem {
  id: string;
  learner_id: string;
  learner_name: string;
  sub_strand_name: string;
  strand_name: string;
  learning_area: string;
  rubric_level: number;
  note: string;
}

export default function TeacherAssessmentsPage() {
  const [items, setItems] = useState<AssessmentItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [grade, setGrade] = useState('');
  const [stream, setStream] = useState('');
  const [term, setTerm] = useState(1);
  const [year, setYear] = useState(new Date().getFullYear());
  const router = useRouter();
  const { token } = useAuth();

  useEffect(() => {
    if (!token) {
      router.push('/auth/login');
      return;
    }
    if (!grade || !stream) return;
    fetch(`${API_BASE}/teacher/classes/assessments?grade=${grade}&stream=${stream}&term=${term}&year=${year}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error('Failed to load assessments'))))
      .then(setItems)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [token, grade, stream, term, year, router]);

  const rubricLabels = ['', 'Below Expectation', 'Approaching Expectation', 'Meeting Expectation', 'Exceeding Expectation'];

  if (!token || loading) {
    return <div className="text-center py-12 text-gray-500">Loading...</div>;
  }

  if (error) {
    return <div className="bg-red-50 text-red-700 p-3 rounded-md text-sm">{error}</div>;
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Assessments</h1>
        <div className="flex gap-2">
          <input
            type="text"
            placeholder="Grade"
            value={grade}
            onChange={(e) => setGrade(e.target.value)}
            className="border rounded-md p-2"
          />
          <input
            type="text"
            placeholder="Stream"
            value={stream}
            onChange={(e) => setStream(e.target.value)}
            className="border rounded-md p-2"
          />
          <select value={term} onChange={(e) => setTerm(Number(e.target.value))} className="border rounded-md p-2">
            <option value={1}>Term 1</option>
            <option value={2}>Term 2</option>
            <option value={3}>Term 3</option>
          </select>
        </div>
      </div>
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Learner</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Learning Area</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Strand</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Level</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {items.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm">{item.learner_name}</td>
                <td className="px-4 py-3 text-sm">{item.learning_area}</td>
                <td className="px-4 py-3 text-sm">{item.strand_name}</td>
                <td className="px-4 py-3 text-sm">{rubricLabels[item.rubric_level] || item.rubric_level}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
