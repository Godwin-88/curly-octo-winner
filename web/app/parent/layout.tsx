'use client';

// Parent portal layout: guardian-session guard + nav + logout.

import { useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth';

export default function ParentLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { guardianToken, ready, logoutGuardian, guardian } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (ready && !guardianToken) {
      router.replace('/parent/login');
    }
  }, [ready, guardianToken, router]);

  const signOut = () => logoutGuardian();

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-blue-600 text-white p-4">
        <div className="max-w-4xl mx-auto flex justify-between items-center gap-4">
          <h1 className="text-xl font-bold">Shule360 Parent Portal</h1>
          {guardianToken ? (
            <div className="flex items-center gap-4">
              <div className="hidden sm:flex space-x-4">
                <Link href="/parent" className="hover:underline">Dashboard</Link>
                <Link href="/parent/results" className="hover:underline">Results</Link>
                <Link href="/parent/fees" className="hover:underline">Fees</Link>
                <Link href="/parent/transport" className="hover:underline">Transport</Link>
              </div>
              <div className="flex items-center gap-3">
                {guardian && (
                  <span className="hidden sm:inline text-sm text-blue-100">
                    {guardian.full_name}
                  </span>
                )}
                <button
                  onClick={signOut}
                  className="text-sm underline hover:text-blue-100"
                >
                  Logout
                </button>
              </div>
            </div>
          ) : (
            <Link href="/parent/login" className="hover:underline">Login</Link>
          )}
        </div>
      </nav>
      <main className="max-w-4xl mx-auto p-6">
        {children}
      </main>
    </div>
  );
}
