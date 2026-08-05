'use client';

import { useState } from 'react';
import { ClipboardList, Plus, Download } from 'lucide-react';

export default function AssessmentsPage() {
  const [activeTab, setActiveTab] = useState('record');

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Formative Assessment</h1>
        <div className="flex gap-3">
          <button className="btn-secondary flex items-center gap-2">
            <Download size={16} /> Report Card
          </button>
          <button className="btn-primary flex items-center gap-2">
            <Plus size={16} /> New Observation
          </button>
        </div>
      </div>

      <div className="flex gap-2 mb-6 border-b">
        {['record', 'rubric-builder', 'portfolio', 'report-cards'].map((tab) => (
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

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <div className="card p-6">
            <h3 className="font-semibold mb-4">Learner Observations</h3>
            <div className="border rounded-lg p-8 text-center text-gray-400">
              <ClipboardList className="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p>Assessment recording interface</p>
              <p className="text-xs mt-2">Select learner → strand/sub-strand → rubric level → add note</p>
            </div>
          </div>
        </div>

        <div>
          <div className="card p-6">
            <h3 className="font-semibold mb-4">Competency Distribution</h3>
            <div className="space-y-3">
              {[
                { label: 'Below Expectation', level: 1, color: 'bg-red-100 text-red-800', count: 0 },
                { label: 'Approaching', level: 2, color: 'bg-yellow-100 text-yellow-800', count: 0 },
                { label: 'Meeting', level: 3, color: 'bg-green-100 text-green-800', count: 0 },
                { label: 'Exceeding', level: 4, color: 'bg-blue-100 text-blue-800', count: 0 },
              ].map((item) => (
                <div key={item.level} className="flex items-center justify-between">
                  <span className="text-sm">{item.label}</span>
                  <span className={`px-2 py-1 rounded text-xs font-medium ${item.color}`}>
                    {item.count} learners
                  </span>
                </div>
              ))}
            </div>
          </div>

          <div className="card p-6 mt-6">
            <h3 className="font-semibold mb-4">Strand Coverage</h3>
            <div className="border rounded-lg p-8 text-center text-gray-400">
              <p className="text-sm">Heatmap of strand coverage per class</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}