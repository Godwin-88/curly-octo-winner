'use client';

// Auth guard for the whole admin area: renders a loading state until the
// session has been hydrated, then redirects anonymous visitors to sign-in.
// (web/proxy.ts performs the same check on the edge via the session cookie.)
//
// Layout: fixed sidebar on desktop; off-canvas drawer + top bar below md.

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Menu, X } from 'lucide-react';
import { useAuth } from '@/lib/auth';
import Sidebar from '@/components/layout/Sidebar';

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { token, ready } = useAuth();
  const router = useRouter();
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    if (ready && !token) {
      router.replace('/auth/login');
    }
  }, [ready, token, router]);

  if (!ready || !token) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-sm text-gray-600">
          {ready ? 'Redirecting to sign in…' : 'Loading…'}
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen">
      <a
        href="#main-content"
        className="skip-link"
      >
        Skip to content
      </a>

      <Sidebar mobileOpen={mobileOpen} onClose={() => setMobileOpen(false)} />

      {/* Backdrop for the mobile drawer */}
      {mobileOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-30 md:hidden"
          aria-hidden="true"
          onClick={() => setMobileOpen(false)}
        />
      )}

      <div className="flex-1 min-w-0 flex flex-col">
        {/* Mobile top bar (sidebar becomes a drawer below md) */}
        <header className="md:hidden sticky top-0 z-20 bg-gray-900 text-white flex items-center gap-3 px-4 py-3">
          <button
            type="button"
            className="p-2 -ml-2 rounded-md hover:bg-gray-800 transition-colors"
            onClick={() => setMobileOpen((o) => !o)}
            aria-expanded={mobileOpen}
            aria-controls="sidebar"
            aria-label={mobileOpen ? 'Close menu' : 'Open menu'}
          >
            {mobileOpen ? <X size={20} aria-hidden="true" /> : <Menu size={20} aria-hidden="true" />}
          </button>
          <span className="font-bold">Shule360</span>
        </header>

        <main id="main-content" tabIndex={-1} className="flex-1 p-4 sm:p-6 lg:p-8 md:ml-64 focus:outline-none">
          {children}
        </main>
      </div>
    </div>
  );
}
