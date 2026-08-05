'use client';

import { useState } from 'react';

interface Template {
  id: string;
  name: string;
  category: string;
  status: 'PENDING' | 'APPROVED' | 'REJECTED';
  body: string;
}

const MOCK_TEMPLATES: Template[] = [
  {
    id: '1',
    name: 'fee_reminder',
    category: 'UTILITY',
    status: 'APPROVED',
    body: 'Dear {{parent_name}}, your child {{learner_name}} has an outstanding fee balance of KES {{fee_balance}}. Please pay at the bursar\'s office.',
  },
  {
    id: '2',
    name: 'result_release',
    category: 'UTILITY',
    status: 'APPROVED',
    body: 'Dear {{parent_name}}, the term report for {{learner_name}} is now available. Please collect from the school office.',
  },
  {
    id: '3',
    name: 'event_announcement',
    category: 'MARKETING',
    status: 'PENDING',
    body: 'Join us for our annual school open day on Saturday. All parents are welcome!',
  },
];

const STATUS_COLORS: Record<string, string> = {
  APPROVED: 'bg-green-100 text-green-800',
  PENDING: 'bg-yellow-100 text-yellow-800',
  REJECTED: 'bg-red-100 text-red-800',
};

export default function TemplatesPage() {
  const [templates, setTemplates] = useState<Template[]>(MOCK_TEMPLATES);
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState('');
  const [category, setCategory] = useState('UTILITY');
  const [body, setBody] = useState('');

  const handleSubmit = () => {
    const newTemplate: Template = {
      id: String(Date.now()),
      name: name.toLowerCase().replace(/[^a-z0-9_]/g, '_'),
      category,
      status: 'PENDING',
      body,
    };
    setTemplates([newTemplate, ...templates]);
    setShowForm(false);
    setName('');
    setCategory('UTILITY');
    setBody('');
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">WhatsApp Templates</h1>
        <button className="btn-primary" onClick={() => setShowForm(!showForm)}>
          {showForm ? 'Cancel' : 'New Template'}
        </button>
      </div>

      {showForm && (
        <div className="card p-6 mb-6 max-w-2xl">
          <h2 className="text-lg font-semibold mb-4">Submit Template for Approval</h2>
          <div className="space-y-4">
            <div>
              <label className="label">Template Name</label>
              <input
                className="input"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. fee_reminder"
              />
            </div>
            <div>
              <label className="label">Category</label>
              <select
                className="input"
                value={category}
                onChange={(e) => setCategory(e.target.value)}
              >
                <option value="UTILITY">UTILITY</option>
                <option value="MARKETING">MARKETING</option>
                <option value="AUTHENTICATION">AUTHENTICATION</option>
              </select>
            </div>
            <div>
              <label className="label">Message Body</label>
              <textarea
                className="input min-h-[120px]"
                value={body}
                onChange={(e) => setBody(e.target.value)}
                placeholder="Use {{variable}} for dynamic content"
              />
            </div>
            <button className="btn-primary" onClick={handleSubmit} disabled={!name || !body}>
              Submit for Approval
            </button>
          </div>
        </div>
      )}

      <div className="space-y-4">
        {templates.map((t) => (
          <div key={t.id} className="card p-6">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-3">
                <h3 className="font-semibold">{t.name}</h3>
                <span className="px-2 py-1 rounded-full text-xs bg-gray-100 text-gray-600">
                  {t.category}
                </span>
              </div>
              <span className={`px-2 py-1 rounded-full text-xs ${STATUS_COLORS[t.status]}`}>
                {t.status}
              </span>
            </div>
            <p className="text-sm text-gray-600">{t.body}</p>
          </div>
        ))}
      </div>
    </div>
  );
}