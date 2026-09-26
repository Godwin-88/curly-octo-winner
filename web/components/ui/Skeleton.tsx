// Skeleton loading placeholders — the standard way to render "content is on
// its way" instead of spinner text. Pair with route-level loading.tsx files
// or per-widget conditional rendering.

export function Skeleton({ className = '' }: { className?: string }) {
  return <div className={`animate-pulse rounded-md bg-gray-200 ${className}`} />;
}

/** A skeleton mimicking the dashboard/section KPI card row. */
export function SkeletonCards({ count = 4 }: { count?: number }) {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4" aria-hidden="true">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="card p-5 space-y-3">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-7 w-16" />
          <Skeleton className="h-3 w-32" />
        </div>
      ))}
    </div>
  );
}

/** A skeleton mimicking the standard data table. */
export function SkeletonTable({ rows = 6 }: { rows?: number }) {
  return (
    <div className="card p-4 space-y-3" aria-hidden="true">
      <Skeleton className="h-4 w-40" />
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-4">
          <Skeleton className="h-3 flex-1" />
          <Skeleton className="h-3 w-20 hidden sm:block" />
          <Skeleton className="h-3 w-16" />
        </div>
      ))}
    </div>
  );
}
