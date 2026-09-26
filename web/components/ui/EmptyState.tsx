// Standard "no data yet" panel for lists and detail views — keeps empty
// states consistent (and honest) across modules.

import type { ReactNode } from 'react';

export function EmptyState({
  title,
  message,
  action,
}: {
  title: string;
  message?: string;
  action?: ReactNode;
}) {
  return (
    <div className="card p-10 flex flex-col items-center justify-center text-center gap-2">
      <div className="text-3xl opacity-50" aria-hidden="true">
        🗂️
      </div>
      <h3 className="text-base font-semibold text-gray-900">{title}</h3>
      {message && <p className="text-sm text-gray-500 max-w-md">{message}</p>}
      {action && <div className="mt-3">{action}</div>}
    </div>
  );
}
