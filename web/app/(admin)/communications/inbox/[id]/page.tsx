'use client';

import { useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { api, Conversation, ConversationMessage } from '@/lib/api';
import { subscribeToMessages } from '@/lib/supabase';

export default function ConversationThreadPage() {
  const params = useParams();
  const conversationId = params.id as string;
  const [conversation, setConversation] = useState<Conversation | null>(null);
  const [messages, setMessages] = useState<ConversationMessage[]>([]);
  const [reply, setReply] = useState('');
  const [sending, setSending] = useState(false);

  const token = ''; // TODO: Get from auth context
  const tenantId = ''; // TODO: Get from auth context

  useEffect(() => {
    loadConversation();
  }, [conversationId]);

  useEffect(() => {
    if (!tenantId) return;
    const sub = subscribeToMessages(tenantId, () => {
      loadConversation();
    });
    return () => {
      sub.unsubscribe();
    };
  }, [tenantId, conversationId]);

  const loadConversation = async () => {
    try {
      const data = await api.getConversation(conversationId, token);
      setConversation(data.conversation);
      setMessages(data.messages);
    } catch (err) {
      console.error('Failed to load conversation:', err);
    }
  };

  const handleSendReply = async () => {
    if (!reply.trim()) return;
    setSending(true);
    try {
      await api.sendReply(conversationId, { content: reply }, token);
      setReply('');
      await loadConversation();
    } catch (err) {
      console.error('Failed to send reply:', err);
    } finally {
      setSending(false);
    }
  };

  const handleStatusChange = async (status: string) => {
    try {
      await api.updateConversationStatus(conversationId, status, token);
      await loadConversation();
    } catch (err) {
      console.error('Failed to update status:', err);
    }
  };

  if (!conversation) {
    return <div className="text-center py-12 text-gray-500">Loading conversation...</div>;
  }

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)]">
      {/* Header */}
      <div className="card p-4 flex items-center justify-between">
        <div>
          <h1 className="text-lg font-semibold">
            {conversation.wa_contact_name || 'Unknown Contact'}
          </h1>
          <p className="text-sm text-gray-500">{conversation.wa_contact_phone}</p>
        </div>
        <div className="flex items-center gap-2">
          <select
            className="input w-auto"
            value={conversation.status}
            onChange={(e) => handleStatusChange(e.target.value)}
          >
            <option value="open">Open</option>
            <option value="in_progress">In Progress</option>
            <option value="waiting">Waiting</option>
            <option value="resolved">Resolved</option>
          </select>
        </div>
      </div>

      {/* Messages thread */}
      <div className="flex-1 overflow-y-auto py-4 space-y-3">
        {messages.map((msg) => {
          const isInbound = msg.direction === 'inbound';
          const text = (msg.content as any)?.text || '';
          return (
            <div
              key={msg.id}
              className={`flex ${isInbound ? 'justify-start' : 'justify-end'}`}
            >
              <div
                className={`max-w-[70%] rounded-lg px-4 py-2 ${
                  isInbound
                    ? 'bg-white border border-gray-200'
                    : 'bg-blue-600 text-white'
                }`}
              >
                <p className="text-sm">{text}</p>
                <p className={`text-xs mt-1 ${isInbound ? 'text-gray-400' : 'text-blue-100'}`}>
                  {new Date(msg.timestamp).toLocaleTimeString()}
                </p>
              </div>
            </div>
          );
        })}
      </div>

      {/* Reply box */}
      <div className="card p-4">
        <div className="flex gap-3">
          <textarea
            className="input flex-1"
            rows={2}
            value={reply}
            onChange={(e) => setReply(e.target.value)}
            placeholder="Type your reply..."
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                handleSendReply();
              }
            }}
          />
          <button
            className="btn-primary self-end"
            onClick={handleSendReply}
            disabled={sending || !reply.trim()}
          >
            {sending ? 'Sending...' : 'Send'}
          </button>
        </div>
      </div>
    </div>
  );
}