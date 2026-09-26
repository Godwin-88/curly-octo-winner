// Teacher portal route-level loading UI.

import { Skeleton, SkeletonCards } from '@/components/ui/Skeleton';

export default function TeacherLoading() {
  return (
    <div className="p-6 space-y-6" aria-busy="true" aria-label="Loading">
      <div className="space-y-2">
        <Skeleton className="h-6 w-40" />
        <Skeleton className="h-4 w-64" />
      </div>
      <SkeletonCards count={3} />
    </div>
  );
}
