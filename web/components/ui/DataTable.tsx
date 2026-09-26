'use client';

// Generic data table with built-in pagination, loading, empty, and error
// states. Adopted by list pages incrementally (see learners page for the
// reference implementation) to kill the per-page table copy-paste.
//
// Accessibility: real <table> semantics with scope="col" headers, a caption,
// and aria-live page status — screen readers announce pagination changes.

import type { ReactNode } from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { useState } from 'react';
import { SkeletonTable } from './Skeleton';
import { ErrorState } from './ErrorState';
import { EmptyState } from './EmptyState';

export interface Column<T> {
  key: string;
  header: string;
  render?: (row: T) => ReactNode;
  className?: string;
  /** Hide on small screens (progressive disclosure, not content loss). */
  hideBelow?: 'sm' | 'md' | 'lg';
}

export function DataTable<T>({
  columns,
  rows,
  keyOf,
  caption,
  loading,
  error,
  onRetry,
  emptyTitle = 'Nothing here yet',
  emptyMessage,
  pageSize = 25,
}: {
  columns: Column<T>[];
  rows: T[];
  keyOf: (row: T) => string;
  caption: string;
  loading?: boolean;
  error?: string | null;
  onRetry?: () => void;
  emptyTitle?: string;
  emptyMessage?: string;
  pageSize?: number;
}) {
  const [page, setPage] = useState(0);

  if (loading) {
    return <SkeletonTable rows={Math.min(pageSize, 6)} />;
  }

  if (error) {
    return <ErrorState title="Could not load data" message={error} onRetry={onRetry} />;
  }

  if (rows.length === 0) {
    return <EmptyState title={emptyTitle} message={emptyMessage} action={onRetry ? undefined : undefined} />;
  }

  const pageCount = Math.max(1, Math.ceil(rows.length / pageSize));
  const safePage = Math.min(page, pageCount - 1);
  const visible = rows.slice(safePage * pageSize, safePage * pageSize + pageSize);
  const hideClass = { sm: 'hidden sm:table-cell', md: 'hidden md:table-cell', lg: 'hidden lg:table-cell' };

  return (
    <div className="card overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <caption className="sr-only">{caption}</caption>
          <thead>
            <tr className="bg-gray-50 text-left text-gray-700 border-b border-gray-200">
              {columns.map((col) => (
                <th
                  key={col.key}
                  scope="col"
                  className={`px-4 py-3 font-medium ${col.hideBelow ? hideClass[col.hideBelow] : ''} ${col.className ?? ''}`}
                >
                  {col.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {visible.map((row) => (
              <tr key={keyOf(row)} className="border-b border-gray-100 last:border-b-0 hover:bg-gray-50">
                {columns.map((col) => (
                  <td
                    key={col.key}
                    className={`px-4 py-3 align-top ${col.hideBelow ? hideClass[col.hideBelow] : ''} ${col.className ?? ''}`}
                  >
                    {col.render ? col.render(row) : String((row as Record<string, unknown>)[col.key] ?? '')}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {pageCount > 1 && (
        <nav
          aria-label="Table pagination"
          className="flex items-center justify-between px-4 py-3 border-t border-gray-200 bg-gray-50"
        >
          <button
            type="button"
            className="btn-secondary text-sm disabled:opacity-40"
            onClick={() => setPage((p) => Math.max(0, p - 1))}
            disabled={safePage === 0}
          >
            <ChevronLeft size={14} aria-hidden="true" /> Previous
          </button>
          <p className="text-sm text-gray-600" aria-live="polite">
            Page {safePage + 1} of {pageCount} · {rows.length} rows
          </p>
          <button
            type="button"
            className="btn-secondary text-sm disabled:opacity-40"
            onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))}
            disabled={safePage >= pageCount - 1}
          >
            Next <ChevronRight size={14} aria-hidden="true" />
          </button>
        </nav>
      )}
    </div>
  );
}
