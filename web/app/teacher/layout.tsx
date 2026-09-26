'use client';

// Teacher workspace layout: staff-session guard + nav + logout.

import { useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/lib/auth';

export default function TeacherLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { token, ready, logoutStaff, staff } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (ready && !token) {
      router.replace('/auth/login');
    }
  }, [ready, token, router]);

  const signOut = () => logoutStaff('/auth/login');

  return (
    <div className="min-h-screen bg-gray-50">
      <a href="#main-content" className="skip-link">
        Skip to content
      </a>
      <nav className="bg-indigo-600 text-white p-4" aria-label="Teacher navigation">
        <div className="max-w-4xl mx-auto flex justify-between items-center gap-4">
          <h1 className="text-xl font-bold">Shule360 Teacher</h1>
          {token ? (
            <div className="flex items-center gap-4">
              {/* Horizontally scrollable nav strip on mobile — every link
                  stays reachable instead of being hidden below sm. */}
              <div className="flex gap-4 overflow-x-auto text-sm -mx-1 px-1">
                <Link href="/teacher" className="hover:underline whitespace-nowrap">Dashboard</Link>
                <Link href="/teacher/attendance" className="hover:underline whitespace-nowrap">Attendance</Link>
                <Link href="/teacher/assessments" className="hover:underline whitespace-nowrap">Assessments</Link>
              </div>
              <div className="flex items-center gap-3">
                {staff && (
                  <span className="hidden sm:inline text-sm text-indigo-100">
                    {staff.full_name}
                  </span>
                )}
                <button
                  onClick={signOut}
                  className="text-sm underline hover:text-indigo-100"
                >
                  Logout
                </button>
              </div>
            </div>
          ) : (
            <Link href="/auth/login" className="hover:underline">Login</Link>
          )}
        </div>
      </nav>
      <main id="main-content" tabIndex={-1} className="max-w-4xl mx-auto p-6 focus:outline-none">
        {children}
      </main>
    </div>
  );
}
