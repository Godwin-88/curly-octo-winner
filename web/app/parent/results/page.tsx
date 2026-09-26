'use client';

import { useEffect, useState } from 'react';
import { API_BASE } from '@/lib/api';
import { useAuth } from '@/lib/auth';

interface ReportCardBrief {
  id: string;
  learner_id: string;
  learner_name: string;
  term: number;
  year: number;
  status: string;
  overall_rating?: number;
  generated_at: string;
}

export default function ParentResultsPage() {
  const [cards, setCards] = useState<ReportCardBrief[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const { token, guardian } = useAuth();

  useEffect(() => {
    if (!guardian) return;
    const learnerId = typeof window !== 'undefined' ? new URLSearchParams(window.location.search).get('learner_id') || '' : '';
    if (!learnerId) {
      setLoading(false);
      return;
    }
    fetch(`${API_BASE}/parent/results?learner_id=${learnerId}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error('Failed to load results'))))
      .then(setCards)
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
      <h1 className="text-2xl font-bold mb-6">Report Cards</h1>
      {cards.length === 0 ? (
        <p className="text-gray-500">No report cards found. Select a learner to view results.</p>
      ) : (
        <div className="space-y-4">
          {cards.map((card) => (
            <div key={card.id} className="bg-white p-6 rounded-lg shadow flex justify-between items-center">
              <div>
                <h3 className="font-semibold">{card.learner_name}</h3>
                <p className="text-sm text-gray-500">Term {card.term} | {card.year}</p>
                <p className="text-sm text-gray-500">Rating: {card.overall_rating || 'N/A'}</p>
              </div>
              <button className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm hover:bg-blue-700">
                Download PDF
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
