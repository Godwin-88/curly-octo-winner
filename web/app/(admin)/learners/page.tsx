'use client';

// Learners directory — reference implementation for the Phase 3 data layer:
// React Query for fetching/caching, server-side pagination (X-Total-Count),
// debounced search, and the shared DataTable + StatCard components.

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { ChevronLeft, ChevronRight, Plus, Search, Users, GraduationCap, HeartHandshake } from 'lucide-react';
import { api, Learner } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { useApiQuery } from '@/lib/query';
import { DataTable, type Column } from '@/components/ui/DataTable';
import { StatCard } from '@/components/ui/StatCard';

const GRADES = ['PP1', 'PP2', 'Grade 1', 'Grade 2', 'Grade 3', 'Grade 4', 'Grade 5', 'Grade 6', 'Grade 7', 'Grade 8', 'Grade 9'];
const PAGE_SIZE = 25;

export default function LearnersPage() {
  const { token } = useAuth();
  const [searchInput, setSearchInput] = useState('');
  const [search, setSearch] = useState('');
  const [grade, setGrade] = useState('');
  const [stream, setStream] = useState('');
  const [includeInactive, setIncludeInactive] = useState(false);
  const [page, setPage] = useState(0);

  // Debounced search: fires 300ms after typing stops.
  useEffect(() => {
    const t = setTimeout(() => setSearch(searchInput.trim()), 300);
    return () => clearTimeout(t);
  }, [searchInput]);

  // Filters changed -> jump back to the first page.
  useEffect(() => {
    setPage(0);
  }, [search, grade, stream, includeInactive]);

  const learnersQuery = useApiQuery(
    ['learners', { grade, stream, search, includeInactive, page }],
    () =>
      api.listLearnersPage(
        {
          grade: grade || undefined,
          stream: stream || undefined,
          search: search || undefined,
          include_inactive: includeInactive,
          limit: PAGE_SIZE,
          offset: page * PAGE_SIZE,
        },
        token
      ),
    { enabled: !!token }
  );

  const learners = learnersQuery.data?.items ?? [];
  const total = learnersQuery.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const specialNeeds = learners.filter((l) => l.special_needs).length;

  const columns: Column<Learner>[] = [
    {
      key: 'full_name',
      header: 'Name',
      render: (l) => (
        <Link href={`/learners/${l.id}`} className="font-medium text-blue-700 hover:underline">
          {l.full_name}
          {l.special_needs && (
            <span className="ml-2 text-xs bg-yellow-100 text-yellow-800 px-2 py-0.5 rounded-full">SN</span>
          )}
        </Link>
      ),
    },
    { key: 'upi', header: 'UPI', className: 'text-gray-600' },
    { key: 'grade', header: 'Grade' },
    { key: 'stream', header: 'Stream', render: (l) => l.stream || '—', hideBelow: 'sm' },
    {
      key: 'is_active',
      header: 'Status',
      render: (l) => (
        <span
          className={`px-2 py-0.5 rounded-full text-xs ${
            l.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-700'
          }`}
        >
          {l.is_active ? 'Active' : 'Inactive'}
        </span>
      ),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (l) => (
        <Link href={`/learners/${l.id}`} className="text-blue-700 hover:underline text-xs">
          View<span className="sr-only"> {l.full_name}</span>
        </Link>
      ),
    },
  ];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Learners</h1>
          <p className="text-sm text-gray-600">Manage learner records, enrollment &amp; progression</p>
        </div>
        <Link href="/learners/new" className="btn-primary flex items-center gap-2">
          <Plus size={16} aria-hidden="true" /> Register Learner
        </Link>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
        <StatCard label="Total learners" value={total.toLocaleString()} icon={Users} tone="blue" />
        <StatCard label="On this page" value={learners.length} icon={GraduationCap} tone="green" />
        <StatCard label="Special needs (page)" value={specialNeeds} icon={HeartHandshake} tone="yellow" />
      </div>

      <div className="card p-4 mb-6">
        <div className="flex flex-wrap gap-3">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500" aria-hidden="true" />
            <input
              type="search"
              aria-label="Search learners by name or UPI"
              placeholder="Search by name or UPI..."
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              className="w-full pl-9 pr-3 py-2 border rounded-md text-sm"
            />
          </div>
          <select aria-label="Filter by grade" value={grade} onChange={(e) => setGrade(e.target.value)} className="px-3 py-2 border rounded-md text-sm">
            <option value="">All Grades</option>
            {GRADES.map((g) => (
              <option key={g} value={g}>{g}</option>
            ))}
          </select>
          <select aria-label="Filter by stream" value={stream} onChange={(e) => setStream(e.target.value)} className="px-3 py-2 border rounded-md text-sm">
            <option value="">All Streams</option>
            {['A', 'B', 'C', 'D', 'E'].map((s) => (
              <option key={s} value={s}>Stream {s}</option>
            ))}
          </select>
          <label className="flex items-center gap-2 text-sm text-gray-700">
            <input type="checkbox" checked={includeInactive} onChange={(e) => setIncludeInactive(e.target.checked)} />
            Include inactive
          </label>
          <button
            type="button"
            onClick={() => {
              setSearchInput('');
              setGrade('');
              setStream('');
              setIncludeInactive(false);
            }}
            className="btn-secondary text-sm"
          >
            Clear
          </button>
        </div>
      </div>

      <DataTable
        columns={columns}
        rows={learners}
        keyOf={(l) => l.id}
        caption="Learners directory"
        loading={learnersQuery.isLoading}
        error={learnersQuery.isError ? (learnersQuery.error as Error).message : null}
        onRetry={() => void learnersQuery.refetch()}
        emptyTitle="No learners found"
        emptyMessage="Adjust the filters or register a new learner."
        pageSize={PAGE_SIZE}
      />

      {total > PAGE_SIZE && (
        <nav aria-label="Learner pages" className="flex items-center justify-between mt-4">
          <button
            type="button"
            className="btn-secondary text-sm disabled:opacity-40"
            onClick={() => setPage((p) => Math.max(0, p - 1))}
            disabled={page === 0 || learnersQuery.isFetching}
          >
            <ChevronLeft size={14} aria-hidden="true" /> Previous
          </button>
          <p className="text-sm text-gray-600" aria-live="polite">
            Page {page + 1} of {pageCount} · {total.toLocaleString()} learners
          </p>
          <button
            type="button"
            className="btn-secondary text-sm disabled:opacity-40"
            onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))}
            disabled={page >= pageCount - 1 || learnersQuery.isFetching}
          >
            Next <ChevronRight size={14} aria-hidden="true" />
          </button>
        </nav>
      )}
    </div>
  );
}
