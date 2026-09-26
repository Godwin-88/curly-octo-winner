'use client';

// Settings → Integrations: one card per provider, with a live connectivity
// test so a principal knows whether a credential actually works.

import { useQuery } from '@tanstack/react-query';
import { settings as settingsApi, type TenantIntegration } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { Skeleton } from '@/components/ui/Skeleton';
import { ProviderCard } from './ProviderCard';
import { PROVIDERS } from './providers';

const EMPTY: TenantIntegration = {
  provider: '',
  is_enabled: false,
  use_platform_default: true,
  config: {},
  secret_fields: [],
  last_test_status: 'never',
  updated_at: '',
};

export function IntegrationsPanel({ canEdit }: { canEdit: boolean }) {
  const { token, staff, ready } = useAuth();

  const { data, isLoading, error } = useQuery({
    queryKey: ['settings', 'integrations'],
    queryFn: () => settingsApi.listIntegrations(token),
    // The session (cookie) is the source of truth, not the in-memory token.
    enabled: ready && !!staff,
  });

  if (isLoading) {
    return (
      <div className="space-y-4" aria-hidden="true">
        {[0, 1, 2].map((i) => (
          <div key={i} className="card p-5 space-y-3">
            <Skeleton className="h-4 w-48" />
            <Skeleton className="h-3 w-72" />
            <Skeleton className="h-9 w-full" />
          </div>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="card p-5 border-red-200 bg-red-50" role="alert">
        <p className="text-sm text-red-800">Could not load integrations: {error.message}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-gray-600">
        Credentials are encrypted before they are stored and are never shown again. Use{' '}
        <span className="font-medium">Test connection</span> to confirm a provider works before
        relying on it.
      </p>
      {PROVIDERS.map((spec) => (
        <ProviderCard
          key={spec.id}
          spec={spec}
          integration={data?.find((i) => i.provider === spec.id) ?? { ...EMPTY, provider: spec.id }}
          canEdit={canEdit}
        />
      ))}
    </div>
  );
}
