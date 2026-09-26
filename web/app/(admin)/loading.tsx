// Admin dashboard route-level loading UI (shown during server navigation).

import { Skeleton, SkeletonCards, SkeletonTable } from '@/components/ui/Skeleton';

export default function AdminLoading() {
  return (
    <div className="p-6 space-y-6" aria-busy="true" aria-label="Loading">
      <div className="space-y-2">
        <Skeleton className="h-6 w-48" />
        <Skeleton className="h-4 w-72" />
      </div>
      <SkeletonCards />
      <SkeletonTable />
    </div>
  );
}
