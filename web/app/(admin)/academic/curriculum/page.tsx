'use client';

import { useState } from 'react';
import { BookOpen, Plus, Trash2 } from 'lucide-react';

export default function CurriculumPage() {
  const [activeTab, setActiveTab] = useState('learning-areas');

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">CBC Curriculum Structure</h1>
        <button className="btn-primary flex items-center gap-2">
          <Plus size={16} /> Add Item
        </button>
      </div>

      <div className="flex gap-2 mb-6 border-b">
        {['learning-areas', 'strands', 'sub-strands', 'competencies', 'values'].map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === tab
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            {tab.replace(/-/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())}
          </button>
        ))}
      </div>

      <div className="card p-6">
        <div className="flex items-center gap-3 mb-4">
          <BookOpen className="w-5 h-5 text-gray-400" />
          <p className="text-gray-500">
            {activeTab === 'learning-areas' && 'Manage KICD learning areas (Mathematics, English, Kiswahili, etc.)'}
            {activeTab === 'strands' && 'Manage strands within each learning area'}
            {activeTab === 'sub-strands' && 'Manage sub-strands within each strand'}
            {activeTab === 'competencies' && 'Manage core competencies (Communication, Critical Thinking, etc.)'}
            {activeTab === 'values' && 'Manage KICD values (Respect, Responsibility, etc.)'}
          </p>
        </div>

        <div className="border rounded-lg p-8 text-center text-gray-400">
          <BookOpen className="w-12 h-12 mx-auto mb-3 opacity-50" />
          <p>Curriculum management interface — {activeTab} tab</p>
          <p className="text-xs mt-2">Full CRUD with KICD code mapping will be implemented here</p>
        </div>
      </div>
    </div>
  );
}