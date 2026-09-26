'use client';

// Standard data hooks over the typed fetch wrapper. Pages migrate to these
// incrementally (React Query handles caching, retries, and loading/error
// states); plain api.* calls keep working side-by-side.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

export { useQueryClient };

/** GET helper: `useApiQuery(['learners', filters], () => api.listLearners(...))`. */
export function useApiQuery<T>(
  key: readonly unknown[],
  fetcher: () => Promise<T>,
  options?: { enabled?: boolean; refetchInterval?: number; staleTime?: number }
) {
  return useQuery({ queryKey: key, queryFn: fetcher, ...options });
}

/** POST/PATCH/DELETE helper with optional cache invalidation. */
export function useApiMutation<TData, TVars = void>(
  mutationFn: (vars: TVars) => Promise<TData>,
  invalidateKeys?: readonly (readonly unknown[])[]
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => {
      if (invalidateKeys) {
        for (const key of invalidateKeys) {
          void queryClient.invalidateQueries({ queryKey: key });
        }
      }
    },
  });
}
