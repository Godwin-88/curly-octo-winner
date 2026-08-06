'use client';

import { useEffect, useState } from 'react';
import { UserCheck, Check, X } from 'lucide-react';
import { api, ConsentAgreement } from '@/lib/api';

const CONSENT_LABELS: Record<string, string> = {
  whatsapp_opt_in: 'WhatsApp Opt-In',
  sms: 'SMS',
  data_processing: 'Data Processing',
  marketing: 'Marketing',
  transport_opt_in: 'Transport Opt-In',
};

export default function ConsentPage() {
  const token = ''; // TODO: Get from auth context
  const [consents, setConsents] = useState<ConsentAgreement[]>([]);
  const [guardianId, setGuardianId] = useState('');
  const [consentType, setConsentType] = useState('whatsapp_opt_in');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const data = await api.listConsentAgreements({ guardian_id: guardianId || undefined }, token);
      setConsents(data);
    } catch (e: any) {
      setError(e.message || 'Failed to load consent records');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const grant = async () => {
    if (!guardianId) {
      setError('Guardian ID is required');
      return;
    }
    setError('');
    try {
      await api.grantConsent({ guardian_id: guardianId, consent_type: consentType, source: 'admin', consent_version: 'v1.0' }, token);
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to grant consent');
    }
  };

  const revoke = async (id: string, type: string) => {
    setError('');
    try {
      await api.revokeConsent({ guardian_id: id, consent_type: type }, token);
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to revoke consent');
    }
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Parent Consent Management</h1>
          <p className="text-gray-500">Explicit consent for WhatsApp opt-in and data processing (KDPA 2019)</p>
        </div>
      </div>

      {/* Grant consent form */}
      <div className="mt-6 bg-white rounded-lg shadow border p-6">
        <h2 className="text-lg font-semibold mb-4">Grant Consent</h2>
        <div className="flex gap-3">
          <input
            value={guardianId}
            onChange={(e) => setGuardianId(e.target.value)}
            placeholder="Guardian ID (UUID)"
            className="px-3 py-2 border rounded-md text-sm flex-1"
          />
          <select value={consentType} onChange={(e) => setConsentType(e.target.value)} className="px-3 py-2 border rounded-md text-sm">
            {Object.entries(CONSENT_LABELS).map(([k, v]) => (
              <option key={k} value={k}>{v}</option>
            ))}
          </select>
          <button onClick={grant} className="px-4 py-2 bg-green-600 text-white rounded-md text-sm hover:bg-green-700 flex items-center gap-2">
            <Check size={16} /> Grant
          </button>
        </div>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {loading ? (
        <p className="text-gray-500 mt-4">Loading...</p>
      ) : (
        <div className="mt-6 bg-white rounded-lg shadow border overflow-hidden">
          {consents.length === 0 ? (
            <div className="p-10 text-center">
              <UserCheck size={32} className="mx-auto text-gray-300 mb-2" />
              <p className="text-gray-400 text-sm">No consent records found.</p>
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-left text-gray-500">
                <tr>
                  <th className="px-3 py-2">Guardian</th>
                  <th className="px-3 py-2">Consent Type</th>
                  <th className="px-3 py-2">Status</th>
                  <th className="px-3 py-2">Granted At</th>
                  <th className="px-3 py-2">Source</th>
                  <th className="px-3 py-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {consents.map((c) => (
                  <tr key={c.id} className="border-t">
                    <td className="px-3 py-2 font-mono text-xs">{c.guardian_id ? c.guardian_id.slice(0, 8) : '-'}</td>
                    <td className="px-3 py-2">{CONSENT_LABELS[c.consent_type] || c.consent_type}</td>
                    <td className="px-3 py-2">
                      <span className={`px-2 py-1 rounded-full text-xs ${c.granted ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}>
                        {c.granted ? 'Granted' : 'Revoked'}
                      </span>
                    </td>
                    <td className="px-3 py-2 whitespace-nowrap">{c.granted_at ? new Date(c.granted_at).toLocaleString() : '-'}</td>
                    <td className="px-3 py-2">{c.source || '-'}</td>
                    <td className="px-3 py-2">
                      {c.granted && (
                        <button
                          onClick={() => c.guardian_id && revoke(c.guardian_id, c.consent_type)}
                          className="px-2 py-1 bg-red-50 text-red-700 rounded-md text-xs hover:bg-red-100 flex items-center gap-1"
                        >
                          <X size={12} /> Revoke
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  );
}