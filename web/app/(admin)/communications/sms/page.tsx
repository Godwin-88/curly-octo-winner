'use client';

import { useState } from 'react';
import AudienceSegmentBuilder from '@/components/comms/AudienceSegmentBuilder';
import MessageComposer from '@/components/comms/MessageComposer';
import { api, CreateMessageRequest, ReachEstimate } from '@/lib/api';

const STEPS = ['Audience', 'Message', 'Review & Send'];

export default function SMSCampaignPage() {
  const [step, setStep] = useState(0);
  const [audienceType, setAudienceType] = useState('all_parents');
  const [audienceFilter, setAudienceFilter] = useState<Record<string, unknown>>({});
  const [content, setContent] = useState('');
  const [scheduledAt, setScheduledAt] = useState<string | undefined>();
  const [estimate, setEstimate] = useState<ReachEstimate | null>(null);
  const [sending, setSending] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const token = ''; // TODO: Get from auth context

  const handleEstimate = async () => {
    try {
      const req: CreateMessageRequest = {
        channel: 'sms',
        audience_type: audienceType,
        audience_filter: audienceFilter,
        content_type: 'text',
        content,
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
        channel: 'sms',
        audience_type: audienceType,
        audience_filter: audienceFilter,
        content_type: 'text',
        content,
        scheduled_at: scheduledAt,
      };
      await api.createMessage(req, token);
      setShowConfirm(false);
      setStep(0);
      alert('Message sent successfully!');
    } catch (err) {
      console.error('Send failed:', err);
      alert('Failed to send message');
    } finally {
      setSending(false);
    }
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">SMS Campaign Builder</h1>

      {/* Step indicator */}
      <div className="flex items-center gap-2 mb-8">
        {STEPS.map((label, i) => (
          <div key={label} className="flex items-center gap-2">
            <div
              className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold ${
                i <= step ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-500'
              }`}
            >
              {i + 1}
            </div>
            <span className={i <= step ? 'text-blue-600 font-medium' : 'text-gray-500'}>
              {label}
            </span>
            {i < STEPS.length - 1 && <div className="w-8 h-px bg-gray-300" />}
          </div>
        ))}
      </div>

      {/* Step 1: Audience */}
      {step === 0 && (
        <AudienceSegmentBuilder
          audienceType={audienceType}
          setAudienceType={setAudienceType}
          audienceFilter={audienceFilter}
          setAudienceFilter={setAudienceFilter}
          onEstimate={handleEstimate}
          estimate={estimate}
        />
      )}

      {/* Step 2: Message */}
      {step === 1 && (
        <MessageComposer
          content={content}
          setContent={setContent}
          scheduledAt={scheduledAt}
          setScheduledAt={setScheduledAt}
        />
      )}

      {/* Step 3: Review & Send */}
      {step === 2 && (
        <div className="card p-6 max-w-2xl">
          <h2 className="text-lg font-semibold mb-4">Review & Send</h2>
          <div className="space-y-4">
            <div className="flex justify-between">
              <span className="text-gray-500">Recipients</span>
              <span className="font-medium">{estimate?.recipient_count ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">Estimated Cost</span>
              <span className="font-medium">KES {estimate?.estimated_kes?.toFixed(2) ?? '0.00'}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-500">SMS Units</span>
              <span className="font-medium">{estimate?.sms_units ?? 0}</span>
            </div>
            <div className="border-t pt-4">
              <h3 className="text-sm font-medium text-gray-500 mb-2">Message Preview</h3>
              <div className="bg-gray-50 rounded-md p-4 text-sm whitespace-pre-wrap">
                {content || 'No message content yet'}
              </div>
            </div>
            {scheduledAt && (
              <div className="flex justify-between">
                <span className="text-gray-500">Scheduled For</span>
                <span className="font-medium">{new Date(scheduledAt).toLocaleString()}</span>
              </div>
            )}
          </div>

          <div className="flex gap-3 mt-6">
            <button className="btn-secondary" onClick={() => setStep(1)}>
              Back
            </button>
            <button
              className="btn-primary flex-1"
              onClick={() => {
                if ((estimate?.recipient_count ?? 0) > 500) {
                  setShowConfirm(true);
                } else {
                  handleSend();
                }
              }}
              disabled={sending}
            >
              {sending ? 'Sending...' : 'Send Now'}
            </button>
          </div>
        </div>
      )}

      {/* Navigation */}
      {step < 2 && (
        <div className="flex gap-3 mt-6">
          {step > 0 && (
            <button className="btn-secondary" onClick={() => setStep(step - 1)}>
              Back
            </button>
          )}
          <button
            className="btn-primary"
            onClick={() => {
              if (step === 0) handleEstimate();
              setStep(step + 1);
            }}
          >
            Continue
          </button>
        </div>
      )}

      {/* Confirmation modal for >500 recipients */}
      {showConfirm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full">
            <h3 className="text-lg font-semibold mb-2">Confirm Bulk Send</h3>
            <p className="text-sm text-gray-600 mb-4">
              You are about to send to <strong>{estimate?.recipient_count}</strong> recipients.
              This will cost approximately <strong>KES {estimate?.estimated_kes?.toFixed(2)}</strong>.
              Are you sure you want to proceed?
            </p>
            <div className="flex gap-3">
              <button className="btn-secondary flex-1" onClick={() => setShowConfirm(false)}>
                Cancel
              </button>
              <button className="btn-primary flex-1" onClick={handleSend} disabled={sending}>
                {sending ? 'Sending...' : 'Confirm Send'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}