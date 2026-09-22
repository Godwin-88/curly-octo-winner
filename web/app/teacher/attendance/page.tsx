'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { CalendarCheck } from 'lucide-react';
import { API_BASE } from '@/lib/api';
import { useAuth } from '@/lib/auth';

interface AttendanceItem {
  id: string;
  learner_id: string;
  learner_name: string;
  status: string;
  reason: string;
  sms_notified: boolean;
}

export default function TeacherAttendancePage() {
  const [items, setItems] = useState<AttendanceItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [grade, setGrade] = useState('');
  const [stream, setStream] = useState('');
  const [date, setDate] = useState(new Date().toISOString().split('T')[0]);
  const router = useRouter();
  const { token } = useAuth();

  useEffect(() => {
    if (!token) {
      router.push('/auth/login');
      return;
    }
    if (!grade || !stream || !date) return;
    fetch(`${API_BASE}/teacher/classes/attendance?grade=${grade}&stream=${stream}&date=${date}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error('Failed to load attendance'))))
      .then(setItems)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [token, grade, stream, date, router]);

  const handleStatusChange = (learnerId: string, status: string) => {
    setItems((prev) =>
      prev.map((item) =>
        item.learner_id === learnerId ? { ...item, status } : item
      )
    );
  };

  const handleSave = async () => {
    if (!token || !grade || !stream) return;
    setError('');
    try {
      const res = await fetch(`${API_BASE}/teacher/classes/attendance`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          grade,
          stream,
          date,
          records: items.map((item) => ({
            learner_id: item.learner_id,
            status: item.status,
            reason: item.reason,
          })),
        }),
      });
      if (!res.ok) throw new Error('Failed to save attendance');
      alert('Attendance saved');
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to save attendance');
    }
  };

  if (!token || loading) {
    return <div className="text-center py-12 text-gray-500">Loading...</div>;
  }

  if (error) {
    return <div className="bg-red-50 text-red-700 p-3 rounded-md text-sm">{error}</div>;
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Mark Attendance</h1>
        <div className="flex gap-2">
          <input
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
            className="border rounded-md p-2"
          />
          <button onClick={handleSave} className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">
            Save Attendance
          </button>
        </div>
      </div>
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Learner</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Status</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">SMS Notified</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {items.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm">{item.learner_name}</td>
                <td className="px-4 py-3">
                  <select
                    value={item.status}
                    onChange={(e) => handleStatusChange(item.learner_id, e.target.value)}
                    className="border rounded-md p-1 text-sm"
                  >
                    <option value="present">Present</option>
                    <option value="absent">Absent</option>
                    <option value="late">Late</option>
                    <option value="excused">Excused</option>
                  </select>
                </td>
                <td className="px-4 py-3 text-sm">
                  {item.sms_notified ? 'Yes' : 'No'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
