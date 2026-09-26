'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { API_BASE } from '@/lib/api';
import { useAuth } from '@/lib/auth';

interface LearnerBrief {
  id: string;
  full_name: string;
  grade: string;
  stream: string;
}

export default function ParentDashboardPage() {
  const [learners, setLearners] = useState<LearnerBrief[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const { token, guardian } = useAuth();

  useEffect(() => {
    if (!guardian) return;
    fetch(`${API_BASE}/parent/learners`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error('Failed to load learners'))))
      .then((data) => setLearners(data))
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
      <h1 className="text-2xl font-bold mb-6">Parent Dashboard</h1>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <div className="bg-white p-6 rounded-lg shadow">
          <h3 className="text-lg font-semibold">My Learners</h3>
          <p className="text-3xl font-bold text-blue-600">{learners.length}</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow">
          <h3 className="text-lg font-semibold">Upcoming Events</h3>
          <p className="text-3xl font-bold text-green-600">3</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow">
          <h3 className="text-lg font-semibold">Fee Balance</h3>
          <p className="text-3xl font-bold text-orange-600">KES 0</p>
        </div>
      </div>
      <div className="bg-white rounded-lg shadow">
        <h2 className="text-xl font-semibold p-6 border-b">My Learners</h2>
        {learners.length === 0 ? (
          <p className="p-6 text-gray-500">No learners linked to this account.</p>
        ) : (
          <div className="divide-y">
            {learners.map((learner) => (
              <div key={learner.id} className="p-4 flex justify-between items-center hover:bg-gray-50">
                <div>
                  <p className="font-medium">{learner.full_name}</p>
                  <p className="text-sm text-gray-500">Grade {learner.grade} | Stream {learner.stream}</p>
                </div>
                <div className="space-x-2">
                  <Link href={`/parent/results?learner_id=${learner.id}`} className="text-blue-600 hover:underline text-sm">
                    Results
                  </Link>
                  <Link href={`/parent/fees?learner_id=${learner.id}`} className="text-blue-600 hover:underline text-sm">
                    Fees
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
