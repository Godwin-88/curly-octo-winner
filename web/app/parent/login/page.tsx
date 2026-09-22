'use client';

// Parent portal sign-in: phone + PIN + school (tenant) selection.
// The school list comes from the public /auth/guardian/schools endpoint so
// guardians never need to know their tenant UUID.

import { useEffect, useState } from 'react';
import { useAuth } from '@/lib/auth';

interface School {
  id: string;
  name: string;
}

export default function ParentLoginPage() {
  const [schools, setSchools] = useState<School[]>([]);
  const [tenantId, setTenantId] = useState('');
  const [phone, setPhone] = useState('');
  const [pin, setPin] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [schoolsLoading, setSchoolsLoading] = useState(true);
  const { loginGuardian, guardianToken, ready } = useAuth();

  // Already signed in -> dashboard (proxy also enforces this).
  useEffect(() => {
    if (ready && guardianToken) {
      window.location.assign('/parent');
    }
  }, [ready, guardianToken]);

  // Load the school list on mount.
  useEffect(() => {
    const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';
    fetch(`${API_BASE}/auth/guardian/schools`)
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error('Failed to load schools'))))
      .then(setSchools)
      .catch(() => {
        // Non-fatal: guardian can still fall back to a stored tenant_id.
      })
      .finally(() => setSchoolsLoading(false));
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      const effectiveTenant =
        tenantId ||
        (typeof window !== 'undefined' ? window.localStorage.getItem('tenant_id') || '' : '');
      if (!effectiveTenant) {
        setError('Please select your school');
        return;
      }
      await loginGuardian(phone, pin, effectiveTenant);
      window.location.assign('/parent');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8 p-8 bg-white rounded-lg shadow">
        <div>
          <h2 className="text-3xl font-bold text-center text-gray-900">Parent Portal</h2>
          <p className="mt-2 text-center text-sm text-gray-600">
            Sign in with your phone number and PIN
          </p>
        </div>
        <form onSubmit={handleSubmit} className="mt-8 space-y-6">
          {error && (
            <div className="bg-red-50 text-red-700 p-3 rounded-md text-sm">{error}</div>
          )}
          <div>
            <label htmlFor="school" className="block text-sm font-medium text-gray-700">School</label>
            {schoolsLoading ? (
              <p className="mt-1 text-sm text-gray-400">Loading schools…</p>
            ) : (
              <select
                id="school"
                value={tenantId}
                onChange={(e) => setTenantId(e.target.value)}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 border p-2"
                required
              >
                <option value="">Select your school…</option>
                {schools.map((s) => (
                  <option key={s.id} value={s.id}>{s.name}</option>
                ))}
              </select>
            )}
          </div>
          <div>
            <label htmlFor="phone" className="block text-sm font-medium text-gray-700">Phone Number</label>
            <input
              id="phone"
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="+254712345678"
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 border p-2"
              required
            />
          </div>
          <div>
            <label htmlFor="pin" className="block text-sm font-medium text-gray-700">PIN</label>
            <input
              id="pin"
              type="password"
              value={pin}
              onChange={(e) => setPin(e.target.value)}
              placeholder="Enter your PIN"
              inputMode="numeric"
              autoComplete="current-password"
              className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 border p-2"
              required
            />
          </div>
          <button
            type="submit"
            disabled={loading}
            className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
          >
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
          <p className="text-center text-xs text-gray-500">
            Demo: use phone + PIN from seed data
          </p>
        </form>
      </div>
    </div>
  );
}
