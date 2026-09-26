'use client';

import { Suspense, useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAuth } from '@/lib/auth';

// The page reads ?next= via useSearchParams, so it must render dynamically
// (or within a Suspense boundary) during prerendering.
export const dynamic = 'force-dynamic';

function LoginForm() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const router = useRouter();
  const searchParams = useSearchParams();
  const { loginStaff, token, ready } = useAuth();

  // Demo credentials are a development aid only — never shown in production.
  const showDemoCredentials = process.env.NODE_ENV !== 'production';
  // Suspended while a session already exists (proxy also enforces this).
  useEffect(() => {
    if (ready && token) {
      router.replace('/dashboard');
    }
  }, [ready, token, router]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    try {
      await loginStaff(email, password);
      const next = searchParams.get('next');
      router.push(next && next.startsWith('/') ? next : '/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-900">Shule360</h1>
          <p className="text-gray-500 mt-2">School Management System</p>
        </div>

        <div className="card p-8">
          <h2 className="text-lg font-semibold mb-6">Sign In</h2>
          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 text-red-700 rounded-md text-sm">
              {error}
            </div>
          )}
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="email" className="label">Email</label>
              <input
                id="email"
                type="email"
                className="input"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@school.ac.ke"
                autoComplete="email"
                required
              />
            </div>
            <div>
              <label htmlFor="password" className="label">Password</label>
              <input
                id="password"
                type="password"
                className="input"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                autoComplete="current-password"
                required
              />
            </div>
            <button type="submit" className="btn-primary w-full" disabled={loading}>
              {loading ? 'Signing in...' : 'Sign In'}
            </button>
          </form>
          {showDemoCredentials && (
            <div className="mt-6 p-4 bg-gray-50 rounded-md text-sm text-gray-600">
              <p className="font-medium mb-2">Demo credentials (development only):</p>
              <p>super_admin: admin@juakali.sch.ke / password123</p>
              <p>principal: principal@juakali.sch.ke / password123</p>
              <p>bursar: bursar@juakali.sch.ke / password123</p>
              <p>teacher: teacher1@juakali.sch.ke / password123</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-sm text-gray-500">Loading…</div>
      </div>
    }>
      <LoginForm />
    </Suspense>
  );
}
