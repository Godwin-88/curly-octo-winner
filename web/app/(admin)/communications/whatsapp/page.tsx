'use client';

import { useState } from 'react';
import AudienceSegmentBuilder from '@/components/comms/AudienceSegmentBuilder';
import { api, CreateMessageRequest, ReachEstimate } from '@/lib/api';

export default function WhatsAppBroadcastPage() {
  const [audienceType, setAudienceType] = useState('all_parents');
  const [audienceFilter, setAudienceFilter] = useState<Record<string, unknown>>({});
  const [content, setContent] = useState('');
  const [templateId, setTemplateId] = useState('');
  const [mediaUrl, setMediaUrl] = useState('');
  const [estimate, setEstimate] = useState<ReachEstimate | null>(null);
  const [sending, setSending] = useState(false);

  const token = ''; // TODO: Get from auth context

  const handleEstimate = async () => {
    try {
      const req: CreateMessageRequest = {
        channel: 'whatsapp',
        audience_type: audienceType,
        audience_filter: audienceFilter,
        content_type: templateId ? 'template' : 'text',
        content,
        template_id: templateId || undefined,
        media_url: mediaUrl || undefined,
      };
      const result = await api.estimateReach(req, token);
      setEstimate(result);
    } catch (err) {
      console.error('Estimate failed:', err);
    }
  };

  const handleSend = async () => {
    setSending(true);
    try {
      const req: CreateMessageRequest = {
        channel: 'whatsapp',
        audience_type: audienceType,
        audience_filter: audienceFilter,
        content_type: templateId ? 'template' : 'text',
        content,
        template_id: templateId || undefined,
        media_url: mediaUrl || undefined,
      };
      await api.createMessage(req, token);
      alert('WhatsApp broadcast sent successfully!');
    } catch (err) {
      console.error('Send failed:', err);
      alert('Failed to send WhatsApp broadcast');
    } finally {
      setSending(false);
    }
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">WhatsApp Broadcast</h1>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left: Builder */}
        <div className="space-y-6">
          <AudienceSegmentBuilder
            audienceType={audienceType}
            setAudienceType={setAudienceType}
            audienceFilter={audienceFilter}
            setAudienceFilter={setAudienceFilter}
            onEstimate={handleEstimate}
            estimate={estimate}
          />

          <div className="card p-6">
            <h2 className="text-lg font-semibold mb-4">Message</h2>
            <div className="space-y-4">
              <div>
                <label className="label">Template (optional)</label>
                <select
                  className="input"
                  value={templateId}
                  onChange={(e) => setTemplateId(e.target.value)}
                >
                  <option value="">Free-form text (24h window)</option>
                  <option value="fee_reminder">Fee Reminder</option>
                  <option value="result_release">Result Release</option>
                  <option value="event_announcement">Event Announcement</option>
                </select>
              </div>

              <div>
                <label className="label">Message Content</label>
                <textarea
                  className="input min-h-[120px]"
                  value={content}
                  onChange={(e) => setContent(e.target.value)}
                  placeholder="Type your WhatsApp message..."
                />
              </div>

              <div>
                <label className="label">Media URL (optional)</label>
                <input
                  className="input"
                  value={mediaUrl}
                  onChange={(e) => setMediaUrl(e.target.value)}
                  placeholder="https://... (uploaded to Backblaze B2)"
                />
              </div>
            </div>
          </div>

          <button
            className="btn-primary w-full"
            onClick={handleSend}
            disabled={sending || !content.trim()}
          >
            {sending ? 'Sending...' : 'Send Broadcast'}
          </button>
        </div>

        {/* Right: Preview */}
        <div className="card p-6">
          <h2 className="text-lg font-semibold mb-4">Preview</h2>
          <div className="bg-[#e5ddd5] rounded-lg p-4 min-h-[300px]">
            <div className="max-w-[80%] ml-auto bg-[#dcf8c6] rounded-lg px-4 py-2">
              <p className="text-sm whitespace-pre-wrap">{content || 'Your message will appear here'}</p>
              <p className="text-xs text-gray-500 mt-1 text-right">
                {new Date().toLocaleTimeString()}
              </p>
            </div>
          </div>
          {templateId && (
            <p className="text-sm text-gray-500 mt-3">
              Using template: <strong>{templateId}</strong>
            </p>
          )}
        </div>
      </div>
    </div>
  );
}