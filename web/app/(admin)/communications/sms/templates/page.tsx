'use client';

import { useEffect, useState } from 'react';
import { api, SMSTemplate } from '@/lib/api';
import { useAuth } from '@/lib/auth';

export default function SMSTemplatesPage() {
  const [templates, setTemplates] = useState<SMSTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState('');
  const [content, setContent] = useState('');
  const [category, setCategory] = useState('general');
  const { token } = useAuth();

  useEffect(() => {
    if (!token) return;
    api.listSMSTemplates(token)
      .then(setTemplates)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [token]);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    await api.createSMSTemplate({ name, content, category }, token);
    setName('');
    setContent('');
    setShowForm(false);
    api.listSMSTemplates(token).then(setTemplates);
  };

  if (loading) {
    return <div className="text-center py-12 text-gray-500">Loading...</div>;
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">SMS Templates</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
        >
          New Template
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="bg-white p-6 rounded-lg shadow mb-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="mt-1 block w-full border rounded-md p-2"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">Content</label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              className="mt-1 block w-full border rounded-md p-2"
              rows={3}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">Category</label>
            <select
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              className="mt-1 block w-full border rounded-md p-2"
            >
              <option value="general">General</option>
              <option value="fee_reminder">Fee Reminder</option>
              <option value="attendance">Attendance</option>
              <option value="transport">Transport</option>
              <option value="results">Results</option>
            </select>
          </div>
          <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700">
            Save Template
          </button>
        </form>
      )}

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Name</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Category</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Content</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-500">Variables</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {templates.map((tpl) => (
              <tr key={tpl.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm font-medium">{tpl.name}</td>
                <td className="px-4 py-3 text-sm">{tpl.category}</td>
                <td className="px-4 py-3 text-sm max-w-xs truncate">{tpl.content}</td>
                <td className="px-4 py-3 text-sm">{tpl.variables?.join(', ') || '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
