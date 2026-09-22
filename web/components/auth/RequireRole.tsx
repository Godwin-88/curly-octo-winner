'use client';

// Page-level role guard for the admin area. The Go API enforces the same
// role rules server-side (RequireRole middleware in api/internal/middleware),
// this only shapes the UX: unauthorized roles get a clear "no access" panel
// instead of a wall of API errors.

import type { ReactNode } from 'react';
import { ShieldAlert } from 'lucide-react';
import { useAuth } from '@/lib/auth';

interface RequireRoleProps {
  /** Staff roles allowed to see this content (e.g. ['principal', 'bursar']). */
  roles: string[];
  children: ReactNode;
  /** Optional custom UI to render instead of the default "no access" panel. */
  fallback?: ReactNode;
}

export default function RequireRole({ roles, children, fallback }: RequireRoleProps) {
  const { staff, ready } = useAuth();

  if (!ready || !staff) return null;

  if (!roles.includes(staff.role)) {
    if (fallback) return <>{fallback}</>;
    return (
      <div className="card p-8 text-center max-w-md mx-auto mt-12">
        <ShieldAlert className="w-10 h-10 mx-auto mb-3 text-gray-300" />
        <h2 className="text-lg font-semibold text-gray-900">Access restricted</h2>
        <p className="text-sm text-gray-500 mt-1">
          This section is limited to: {roles.join(', ')}.
        </p>
        <p className="text-xs text-gray-400 mt-3">
          You are signed in as <span className="font-medium">{staff.role}</span>. Ask your
          school&apos;s administrator if you believe you should have access.
        </p>
      </div>
    );
  }

  return <>{children}</>;
}
