'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { API_BASE } from '@/lib/api';
import { useAuth } from '@/lib/auth';

interface ClassSummary {
  grade: string;
  stream: string;
  learner_count: number;
}

export default function TeacherDashboardPage() {
  const [classes, setClasses] = useState<ClassSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const { token, staff } = useAuth();

  useEffect(() => {
    if (!staff) return;
    fetch(`${API_BASE}/teacher/classes`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error('Failed to load classes'))))
      .then(setClasses)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [token]);

  if (!token || loading) {
    return <div className="text-center py-12 text-gray-500">Loading...</div>;
  }

  if (error) {
    return <div className="bg-red-50 text-red-700 p-3 rounded-md text-sm">{error}</div>;
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Teacher Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <div className="bg-white p-6 rounded-lg shadow">
          <h3 className="text-lg font-semibold">My Classes</h3>
          <p className="text-3xl font-bold text-indigo-600">{classes.length}</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow">
          <h3 className="text-lg font-semibold">Total Learners</h3>
          <p className="text-3xl font-bold text-blue-600">
            {classes.reduce((sum, c) => sum + c.learner_count, 0)}
          </p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow">
          <h3 className="text-lg font-semibold">Assessments This Week</h3>
          <p className="text-3xl font-bold text-green-600">12</p>
        </div>
      </div>
      <div className="bg-white rounded-lg shadow">
        <h2 className="text-xl font-semibold p-6 border-b">My Classes</h2>
        {classes.length === 0 ? (
          <p className="p-6 text-gray-500">No classes assigned.</p>
        ) : (
          <div className="divide-y">
            {classes.map((cls, i) => (
              <div key={i} className="p-4 flex justify-between items-center hover:bg-gray-50">
                <div>
                  <p className="font-medium">Grade {cls.grade} | Stream {cls.stream}</p>
                  <p className="text-sm text-gray-500">{cls.learner_count} learners</p>
                </div>
                <div className="space-x-2">
                  <Link href={`/teacher/attendance?grade=${cls.grade}&stream=${cls.stream}`} className="text-blue-600 hover:underline text-sm">
                    Attendance
                  </Link>
                  <Link href={`/teacher/assessments?grade=${cls.grade}&stream=${cls.stream}`} className="text-blue-600 hover:underline text-sm">
                    Assessments
                  </Link>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
