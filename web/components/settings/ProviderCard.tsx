'use client';

// One provider card for Settings → Integrations: save, test, or clear a
// provider without ever seeing a stored secret.

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { CheckCircle2, RefreshCw, Trash2, XCircle } from 'lucide-react';
import { settings as settingsApi, type TenantIntegration } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import type { ProviderSpec } from './providers';

function StatusBadge({ integration }: { integration: TenantIntegration }) {
  if (!integration.is_enabled) {
    return (
      <span className="text-xs font-medium text-gray-500 bg-gray-100 rounded-full px-2 py-0.5">Off</span>
    );
  }
  if (integration.last_test_status === 'ok') {
    return (
      <span className="inline-flex items-center gap-1 text-xs font-medium text-green-700 bg-green-50 rounded-full px-2 py-0.5">
        <CheckCircle2 size={12} aria-hidden="true" /> Working
      </span>
    );
  }
  if (integration.last_test_status === 'failed') {
    return (
      <span className="inline-flex items-center gap-1 text-xs font-medium text-red-700 bg-red-50 rounded-full px-2 py-0.5">
        <XCircle size={12} aria-hidden="true" /> Needs attention
      </span>
    );
  }
  return (
    <span className="text-xs font-medium text-gray-600 bg-gray-100 rounded-full px-2 py-0.5">
      On · not tested
    </span>
  );
}

export function ProviderCard({
  spec,
  integration,
  canEdit,
}: {
  spec: ProviderSpec;
  integration: TenantIntegration;
  canEdit: boolean;
}) {
  const { token } = useAuth();
  const queryClient = useQueryClient();
  const [config, setConfig] = useState<Record<string, string>>(integration.config ?? {});
  const [secrets, setSecrets] = useState<Record<string, string>>({});
  const [enabled, setEnabled] = useState(integration.is_enabled);
  const [useDefault, setUseDefault] = useState(integration.use_platform_default);
  const [notice, setNotice] = useState<{ ok: boolean; text: string } | null>(null);

  const refresh = () => queryClient.invalidateQueries({ queryKey: ['settings', 'integrations'] });
  const fail = (e: Error) => setNotice({ ok: false, text: e.message });

  const save = useMutation({
    mutationFn: () =>
      settingsApi.saveIntegration(
        spec.id,
        {
          is_enabled: enabled,
          use_platform_default: useDefault,
          config,
          // Only credentials actually typed are sent; blanks keep the stored value.
          secrets,
        },
        token
      ),
    onSuccess: () => {
      setSecrets({});
      setNotice({ ok: true, text: 'Saved. Stored credentials are never displayed again.' });
      refresh();
    },
    onError: fail,
  });

  const test = useMutation({
    mutationFn: () => settingsApi.testIntegration(spec.id, token),
    onSuccess: (result) => {
      setNotice({ ok: result.ok, text: result.message });
      refresh();
    },
    onError: fail,
  });

  const remove = useMutation({
    mutationFn: () => settingsApi.deleteIntegration(spec.id, token),
    onSuccess: () => {
      setSecrets({});
      setNotice({ ok: true, text: 'Configuration removed.' });
      refresh();
    },
    onError: fail,
  });

  return (
    <article className="card p-5" aria-labelledby={`provider-${spec.id}-heading`}>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 id={`provider-${spec.id}-heading`} className="text-base font-semibold text-gray-900">
            {spec.name}
          </h3>
          <p className="text-sm text-gray-600 mt-0.5">{spec.purpose}</p>
        </div>
        <StatusBadge integration={integration} />
      </div>

      {canEdit && (
        <div className="mt-4 flex flex-wrap items-center gap-5">
          <label className="inline-flex items-center gap-2 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-primary-600 focus-visible:ring-2 focus-visible:ring-primary-500"
            />
            Enabled for this school
          </label>
          <label className="inline-flex items-center gap-2 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={useDefault}
              onChange={(e) => setUseDefault(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-primary-600 focus-visible:ring-2 focus-visible:ring-primary-500"
            />
            Use the platform credentials
          </label>
        </div>
      )}
      <div className="mt-4 grid grid-cols-1 sm:grid-cols-2 gap-4">
        {spec.fields.map((field) => {
          const stored = integration.secret_fields.includes(field.key);
          const inputId = `${spec.id}-${field.key}`;
          return (
            <div key={field.key}>
              <label htmlFor={inputId} className="block text-sm font-medium text-gray-700">
                {field.label}
              </label>
              {field.secret ? (
                <>
                  <input
                    id={inputId}
                    type="password"
                    autoComplete="new-password"
                    disabled={!canEdit}
                    value={secrets[field.key] ?? ''}
                    onChange={(e) => setSecrets((s) => ({ ...s, [field.key]: e.target.value }))}
                    placeholder={stored ? 'Stored — leave blank to keep' : 'Not set'}
                    aria-describedby={field.help ? `${inputId}-help` : undefined}
                    className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:outline-none disabled:bg-gray-50"
                  />
                  {stored && (
                    <p className="mt-1 text-xs text-green-700">A credential is stored for this field.</p>
                  )}
                </>
              ) : (
                <input
                  id={inputId}
                  type="text"
                  disabled={!canEdit}
                  value={config[field.key] ?? ''}
                  onChange={(e) => setConfig((c) => ({ ...c, [field.key]: e.target.value }))}
                  placeholder={field.placeholder}
                  aria-describedby={field.help ? `${inputId}-help` : undefined}
                  className="mt-1 w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:outline-none disabled:bg-gray-50"
                />
              )}
              {field.help && (
                <p id={`${inputId}-help`} className="mt-1 text-xs text-gray-500">
                  {field.help}
                </p>
              )}
            </div>
          );
        })}
      </div>

      <p className="mt-3 text-xs text-gray-500">{spec.docsHint}</p>

      {integration.last_test_message && (
        <p className="mt-3 text-sm text-gray-700">
          <span className="font-medium">Last test:</span> {integration.last_test_message}
        </p>
      )}

      {notice && (
        <p role="status" className={`mt-3 text-sm ${notice.ok ? 'text-green-700' : 'text-red-700'}`}>
          {notice.text}
        </p>
      )}

      {canEdit && (
        <div className="mt-4 flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={() => save.mutate()}
            disabled={save.isPending}
            className="btn-primary focus-visible:ring-2 focus-visible:ring-primary-500"
          >
            {save.isPending ? 'Saving…' : 'Save'}
          </button>
          <button
            type="button"
            onClick={() => test.mutate()}
            disabled={test.isPending}
            className="btn-secondary inline-flex items-center gap-2 focus-visible:ring-2 focus-visible:ring-primary-500"
          >
            <RefreshCw size={15} aria-hidden="true" />
            {test.isPending ? 'Testing…' : 'Test connection'}
          </button>
          <button
            type="button"
            onClick={() => remove.mutate()}
            disabled={remove.isPending}
            className="btn-danger inline-flex items-center gap-2 focus-visible:ring-2 focus-visible:ring-primary-500"
          >
            <Trash2 size={15} aria-hidden="true" />
            {remove.isPending ? 'Removing…' : 'Remove'}
          </button>
        </div>
      )}
    </article>
  );
}
