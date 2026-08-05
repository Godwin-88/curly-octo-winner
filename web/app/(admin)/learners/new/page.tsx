'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Save } from 'lucide-react';
import { api } from '@/lib/api';

const GRADES = ['PP1', 'PP2', 'Grade 1', 'Grade 2', 'Grade 3', 'Grade 4', 'Grade 5', 'Grade 6', 'Grade 7', 'Grade 8', 'Grade 9'];
const ENTRY_LEVELS = ['PP1', 'PP2', 'Grade 1', 'Grade 2', 'Grade 3', 'Grade 4', 'Grade 5', 'Grade 6', 'Grade 7', 'Grade 8', 'Grade 9', 'Transfer'];
const STREAMS = ['A', 'B', 'C', 'D', 'E'];

export default function NewLearnerPage() {
  const router = useRouter();
  const token = ''; // TODO: Get from auth context
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [form, setForm] = useState({
    upi: '',
    full_name: '',
    date_of_birth: '',
    grade: '',
    stream: '',
    guardian_ids: '',
    birth_cert_no: '',
    entry_level: '',
    special_needs: false,
    admission_date: new Date().toISOString().split('T')[0],
  });

  const set = (field: string, value: any) => setForm((f) => ({ ...f, [field]: value }));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSaving(true);
    try {
      const guardianIDs = form.guardian_ids
        ? form.guardian_ids.split(',').map((s) => s.trim()).filter(Boolean)
        : [];
      await api.createLearner({
        upi: form.upi.trim(),
        full_name: form.full_name.trim(),
        date_of_birth: form.date_of_birth || undefined,
        grade: form.grade,
        stream: form.stream || undefined,
        guardian_ids: guardianIDs,
        birth_cert_no: form.birth_cert_no || undefined,
        entry_level: form.entry_level || undefined,
        special_needs: form.special_needs,
        admission_date: form.admission_date || undefined,
      }, token);
      router.push('/learners');
    } catch (err: any) {
      setError(err.message || 'Failed to register learner');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="max-w-3xl">
      <div className="flex items-center gap-4 mb-6">
        <button onClick={() => router.back()} className="p-2 rounded-md hover:bg-gray-100">
          <ArrowLeft size={20} />
        </button>
        <div>
          <h1 className="text-2xl font-bold">Register Learner</h1>
          <p className="text-sm text-gray-500">Enroll a new learner (NEMIS UPI validated)</p>
        </div>
      </div>

      {error && <div className="bg-red-50 text-red-700 p-3 rounded-md mb-4 text-sm">{error}</div>}

      <form onSubmit={handleSubmit} className="card p-6 space-y-6">
        {/* NEMIS / Enrollment */}
        <div>
          <h3 className="font-semibold mb-4">NEMIS Enrollment</h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                UPI (NEMIS Unique Personal Identifier) *
              </label>
              <input
                type="text"
                value={form.upi}
                onChange={(e) => set('upi', e.target.value)}
                required
                placeholder="TEST12345678"
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
              <p className="text-xs text-gray-400 mt-1">Format: TEST + 8 digits (sandbox) or 16-char NEMIS UPI</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Birth Certificate No.</label>
              <input
                type="text"
                value={form.birth_cert_no}
                onChange={(e) => set('birth_cert_no', e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Admission Date</label>
              <input
                type="date"
                value={form.admission_date}
                onChange={(e) => set('admission_date', e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Entry Level</label>
              <select
                value={form.entry_level}
                onChange={(e) => set('entry_level', e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm"
              >
                <option value="">Select entry level</option>
                {ENTRY_LEVELS.map((l) => (
                  <option key={l} value={l}>{l}</option>
                ))}
              </select>
            </div>
          </div>
        </div>

        {/* Personal Details */}
        <div className="border-t pt-6">
          <h3 className="font-semibold mb-4">Personal Details</h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="sm:col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-1">Full Name *</label>
              <input
                type="text"
                value={form.full_name}
                onChange={(e) => set('full_name', e.target.value)}
                required
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Date of Birth</label>
              <input
                type="date"
                value={form.date_of_birth}
                onChange={(e) => set('date_of_birth', e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Grade *</label>
              <select
                value={form.grade}
                onChange={(e) => set('grade', e.target.value)}
                required
                className="w-full px-3 py-2 border rounded-md text-sm"
              >
                <option value="">Select grade</option>
                {GRADES.map((g) => (
                  <option key={g} value={g}>{g}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Stream</label>
              <select
                value={form.stream}
                onChange={(e) => set('stream', e.target.value)}
                className="w-full px-3 py-2 border rounded-md text-sm"
              >
                <option value="">Select stream</option>
                {STREAMS.map((s) => (
                  <option key={s} value={s}>Stream {s}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Guardian IDs (comma-separated)</label>
              <input
                type="text"
                value={form.guardian_ids}
                onChange={(e) => set('guardian_ids', e.target.value)}
                placeholder="uuid-1, uuid-2"
                className="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
          </div>

          <label className="flex items-center gap-2 mt-4 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={form.special_needs}
              onChange={(e) => set('special_needs', e.target.checked)}
            />
            Learner has special needs
          </label>
        </div>

        <div className="border-t pt-6 flex justify-end">
          <button
            type="submit"
            disabled={saving}
            className="btn-primary flex items-center gap-2"
          >
            <Save size={16} />
            {saving ? 'Registering...' : 'Register Learner'}
          </button>
        </div>
      </form>
    </div>
  );
}