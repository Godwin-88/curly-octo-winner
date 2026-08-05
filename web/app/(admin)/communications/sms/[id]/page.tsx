'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { api, DeliveryStats, Message, MessageLog } from '@/lib/api';

const STATUS_COLORS: Record<string, string> = {
  sent: 'bg-blue-100 text-blue-800',
  delivered: 'bg-green-100 text-green-800',
  failed: 'bg-red-100 text-red-800',
  pending: 'bg-yellow-100 text-yellow-800',
  read: 'bg-purple-100 text-purple-800',
};

export default function DeliveryReportPage() {
  const params = useParams();
  const messageId = params.id as string;
  const [message, setMessage] = useState<Message | null>(null);
  const [stats, setStats] = useState<DeliveryStats | null>(null);
  const [logs, setLogs] = useState<MessageLog[]>([]);
  const [loading, setLoading] = useState(true);

  const token = ''; // TODO: Get from auth context

  useEffect(() => {
    loadData();
  }, [messageId]);

  const loadData = async () => {
    try {
      const [msgData, logData] = await Promise.all([
        api.getMessage(messageId, token),
        api.getMessageLogs(messageId, { limit: 100 }, token),
      ]);
      setMessage(msgData.message);
      setStats(msgData.stats);
      setLogs(logData);
    } catch (err) {
      console.error('Failed to load delivery report:', err);
    } finally {
      setLoading(false);
    }
  };

  const exportCSV = () => {
    const headers = ['Phone', 'Status', 'Channel', 'Delivered At', 'Error Code', 'Error Message'];
    const rows = logs.map((l) => [
      l.phone,
      l.status,
      l.channel,
      l.delivered_at || '',
      l.error_code || '',
      l.error_message || '',
    ]);
    const csv = [headers, ...rows].map((r) => r.join(',')).join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `delivery-report-${messageId}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  if (loading) {
    return <div className="text-center py-12 text-gray-500">Loading delivery report...</div>;
  }

  if (!message || !stats) {
    return <div className="text-center py-12 text-gray-500">Message not found</div>;
  }

  const failedLogs = logs.filter((l) => l.status === 'failed');

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Delivery Report</h1>
        <button className="btn-secondary" onClick={exportCSV}>
          Export CSV
        </button>
      </div>

      {/* Stats cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <div className="card p-6">
          <h3 className="text-sm text-gray-500">Sent</h3>
          <p className="text-3xl font-bold mt-2 text-blue-600">{stats.sent}</p>
        </div>
        <div className="card p-6">
          <h3 className="text-sm text-gray-500">Delivered</h3>
          <p className="text-3xl font-bold mt-2 text-green-600">{stats.delivered}</p>
        </div>
        <div className="card p-6">
          <h3 className="text-sm text-gray-500">Failed</h3>
          <p className="text-3xl font-bold mt-2 text-red-600">{stats.failed}</p>
        </div>
        <div className="card p-6">
          <h3 className="text-sm text-gray-500">Delivery Rate</h3>
          <p className="text-3xl font-bold mt-2">{stats.delivery_rate.toFixed(1)}%</p>
        </div>
      </div>

      {/* Message info */}
      <div className="card p-6 mb-8">
        <h2 className="text-lg font-semibold mb-2">Message Details</h2>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-gray-500">Channel:</span> {message.channel}
          </div>
          <div>
            <span className="text-gray-500">Audience:</span> {message.audience_type}
          </div>
          <div>
            <span className="text-gray-500">Status:</span> {message.status}
          </div>
          <div>
            <span className="text-gray-500">Recipients:</span> {message.recipient_count}
          </div>
        </div>
        <div className="mt-4 bg-gray-50 rounded-md p-4 text-sm whitespace-pre-wrap">
          {message.content}
        </div>
      </div>

      {/* Recipient table */}
      <div className="card overflow-hidden">
        <div className="px-4 py-3 border-b">
          <h2 className="font-semibold">Recipients</h2>
        </div>
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Phone</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Status</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Channel</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Delivered At</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {logs.map((log) => (
              <tr key={log.id}>
                <td className="px-4 py-3 text-sm">
                  {log.phone.replace(/^(\+254)(\d{3})(\d{3})(\d{3})$/, '$1 $2***$4')}
                </td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-1 rounded-full text-xs ${STATUS_COLORS[log.status] || 'bg-gray-100'}`}>
                    {log.status}
                  </span>
                </td>
                <td className="px-4 py-3 text-sm">{log.channel}</td>
                <td className="px-4 py-3 text-sm text-gray-500">
                  {log.delivered_at ? new Date(log.delivered_at).toLocaleString() : '-'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Failed numbers */}
      {failedLogs.length > 0 && (
        <div className="card overflow-hidden mt-6">
          <div className="px-4 py-3 border-b">
            <h2 className="font-semibold text-red-600">Failed Numbers ({failedLogs.length})</h2>
          </div>
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Phone</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Error Code</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Error Message</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {failedLogs.map((log) => (
                <tr key={log.id}>
                  <td className="px-4 py-3 text-sm">{log.phone}</td>
                  <td className="px-4 py-3 text-sm text-red-600">{log.error_code || '-'}</td>
                  <td className="px-4 py-3 text-sm text-gray-600">{log.error_message || '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}