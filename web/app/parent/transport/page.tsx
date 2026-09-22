'use client';

import { useEffect, useState } from 'react';
import { API_BASE } from '@/lib/api';
import { useAuth } from '@/lib/auth';

interface TripBrief {
  id: string;
  route_id: string;
  route_name: string;
  direction: string;
  status: string;
  scheduled_departure: string;
  actual_departure?: string;
  boarded_count: number;
}

export default function ParentTransportPage() {
  const [trips, setTrips] = useState<TripBrief[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const { guardianToken: token } = useAuth();

  useEffect(() => {
    if (!token) return;
    fetch(`${API_BASE}/parent/transport`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error('Failed to load transport'))))
      .then(setTrips)
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
      <h1 className="text-2xl font-bold mb-6">Transport</h1>
      {trips.length === 0 ? (
        <p className="text-gray-500">No active trips found.</p>
      ) : (
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Route</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Direction</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Status</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Scheduled</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Boarded</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {trips.map((trip) => (
                <tr key={trip.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm">{trip.route_name}</td>
                  <td className="px-4 py-3 text-sm capitalize">{trip.direction}</td>
                  <td className="px-4 py-3 text-sm">
                    <span className={`px-2 py-1 rounded-full text-xs ${
                      trip.status === 'in_progress' ? 'bg-green-100 text-green-800' :
                      trip.status === 'scheduled' ? 'bg-yellow-100 text-yellow-800' :
                      'bg-gray-100 text-gray-800'
                    }`}>
                      {trip.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm">{new Date(trip.scheduled_departure).toLocaleString()}</td>
                  <td className="px-4 py-3 text-sm">{trip.boarded_count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
