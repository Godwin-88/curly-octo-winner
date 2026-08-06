'use client';

import { useEffect, useState } from 'react';
import { ClipboardList } from 'lucide-react';
import { api, AuditLog } from '@/lib/api';

const ACTION_COLORS: Record<string, string> = {
  CREATE: 'bg-green-50 text-green-700',
  UPDATE: 'bg-blue-50 text-blue-700',
  DELETE: 'bg-red-50 text-red-700',
  VIEW: 'bg-gray-50 text-gray-700',
};

export default function AuditLogPage() {
  const token = ''; // TODO: Get from auth context
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [entityType, setEntityType] = useState('');
  const [action, setAction] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const data = await api.listAuditLogs({ entity_type: entityType || undefined, action: action || undefined, limit: 100 }, token);
      setLogs(data);
    } catch (e: any) {
      setError(e.message || 'Failed to load audit logs');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Audit Log</h1>
          <p className="text-gray-500">Every create, update, and delete — with actor, timestamp, and IP</p>
        </div>
      </div>

      {/* Filters */}
      <div className="mt-4 flex gap-3">
        <input
          value={entityType}
          onChange={(e) => setEntityType(e.target.value)}
          placeholder="Entity type (e.g. learner, payment)"
          className="px-3 py-2 border rounded-md text-sm w-64"
        />
        <select value={action} onChange={(e) => setAction(e.target.value)} className="px-3 py-2 border rounded-md text-sm">
          <option value="">All actions</option>
          <option value="CREATE">CREATE</option>
          <option value="UPDATE">UPDATE</option>
          <option value="DELETE">DELETE</option>
          <option value="VIEW">VIEW</option>
        </select>
        <button onClick={load} className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm hover:bg-blue-700">
          Filter
        </button>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {loading ? (
        <p className="text-gray-500 mt-4">Loading...</p>
      ) : (
        <div className="mt-6 bg-white rounded-lg shadow border overflow-hidden">
          {logs.length === 0 ? (
            <div className="p-10 text-center">
              <ClipboardList size={32} className="mx-auto text-gray-300 mb-2" />
              <p className="text-gray-400 text-sm">No audit events found.</p>
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-left text-gray-500">
                <tr>
                  <th className="px-3 py-2">Timestamp</th>
                  <th className="px-3 py-2">Actor</th>
                  <th className="px-3 py-2">Action</th>
                  <th className="px-3 py-2">Entity</th>
                  <th className="px-3 py-2">IP Address</th>
                </tr>
              </thead>
              <tbody>
                {logs.map((l) => (
                  <tr key={l.id} className="border-t">
                    <td className="px-3 py-2 whitespace-nowrap">{new Date(l.created_at).toLocaleString()}</td>
                    <td className="px-3 py-2">{l.actor_staff_id ? l.actor_staff_id.slice(0, 8) : 'System'}</td>
                    <td className="px-3 py-2">
                      <span className={`px-2 py-1 rounded-full text-xs ${ACTION_COLORS[l.action] || 'bg-gray-50 text-gray-700'}`}>
                        {l.action}
                      </span>
                    </td>
                    <td className="px-3 py-2">
                      <span className="font-mono text-xs">{l.entity_type}</span>
                      {l.entity_id && <span className="text-gray-400 text-xs ml-1">({l.entity_id.slice(0, 8)})</span>}
                    </td>
                    <td className="px-3 py-2 font-mono text-xs">{l.ip_address || '-'}</td>
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