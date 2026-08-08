'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api, Conversation } from '@/lib/api';
import { subscribeToMessages } from '@/lib/supabase';

const STATUS_COLORS: Record<string, string> = {
  open: 'bg-yellow-100 text-yellow-800',
  in_progress: 'bg-blue-100 text-blue-800',
  waiting: 'bg-purple-100 text-purple-800',
  resolved: 'bg-green-100 text-green-800',
};

export default function InboxPage() {
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState('');
  const [token, setToken] = useState('');
  const [tenantId, setTenantId] = useState('');

  useEffect(() => {
    // Read auth from localStorage (set on login), like the dashboard does
    const t = typeof window !== 'undefined' ? localStorage.getItem('token') || '' : '';
    setToken(t);
    const staff = typeof window !== 'undefined' ? localStorage.getItem('staff') : null;
    if (staff) {
      try {
        setTenantId(JSON.parse(staff).tenant_id || '');
      } catch {
        setTenantId('');
      }
    }
  }, []);

  useEffect(() => {
    if (token) loadConversations();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [statusFilter, token]);

  useEffect(() => {
    if (!tenantId) return;
    const sub = subscribeToMessages(tenantId, () => {
      loadConversations();
    });
    return () => {
      sub.unsubscribe();
    };
  }, [tenantId]);

  const loadConversations = async () => {
    try {
      const data = await api.listConversations(
        { status: statusFilter || undefined, limit: 50 },
        token
      );
      setConversations(data);
    } catch (err) {
      console.error('Failed to load conversations:', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">WhatsApp Inbox</h1>

      {/* Status filter */}
      <div className="flex gap-2 mb-6">
        <button
          className={`px-3 py-1 rounded-md text-sm ${
            statusFilter === '' ? 'bg-blue-600 text-white' : 'bg-gray-200'
          }`}
          onClick={() => setStatusFilter('')}
        >
          All
        </button>
        {['open', 'in_progress', 'waiting', 'resolved'].map((s) => (
          <button
            key={s}
            className={`px-3 py-1 rounded-md text-sm ${
              statusFilter === s ? 'bg-blue-600 text-white' : 'bg-gray-200'
            }`}
            onClick={() => setStatusFilter(s)}
          >
            {s.replace('_', ' ')}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="text-center py-12 text-gray-500">Loading conversations...</div>
      ) : conversations.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          No conversations yet. Incoming WhatsApp messages will appear here.
        </div>
      ) : (
        <div className="card overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Contact</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Status</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Last Message</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Time</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Unread</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {conversations.map((conv) => (
                <tr key={conv.id} className="hover:bg-gray-50 cursor-pointer">
                  <td className="px-4 py-3">
                    <Link href={`/communications/inbox/${conv.id}`} className="block">
                      <div className="font-medium">{conv.wa_contact_name || 'Unknown'}</div>
                      <div className="text-sm text-gray-500">{conv.wa_contact_phone}</div>
                    </Link>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-1 rounded-full text-xs ${STATUS_COLORS[conv.status] || 'bg-gray-100'}`}>
                      {conv.status.replace('_', ' ')}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600 max-w-[200px] truncate">
                    {conv.last_message_preview}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    {new Date(conv.last_message_at).toLocaleString()}
                  </td>
                  <td className="px-4 py-3">
                    {conv.unread_count > 0 && (
                      <span className="inline-flex items-center justify-center w-6 h-6 bg-red-500 text-white rounded-full text-xs font-bold">
                        {conv.unread_count}
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}