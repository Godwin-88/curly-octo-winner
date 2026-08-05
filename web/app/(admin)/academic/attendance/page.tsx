'use client';

import { useState } from 'react';
import { CalendarCheck, Plus, AlertTriangle } from 'lucide-react';

export default function AttendancePage() {
  const [activeTab, setActiveTab] = useState('daily');

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Attendance Management</h1>
        <button className="btn-primary flex items-center gap-2">
          <Plus size={16} /> Mark Attendance
        </button>
      </div>

      <div className="flex gap-2 mb-6 border-b">
        {['daily', 'weekly', 'chronic', 'reports'].map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
              activeTab === tab
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            {tab.charAt(0).toUpperCase() + tab.slice(1)}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <div className="card p-6">
            <h3 className="font-semibold mb-4">Daily Roll Call</h3>
            <div className="border rounded-lg p-8 text-center text-gray-400">
              <CalendarCheck className="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p>Attendance marking interface</p>
              <p className="text-xs mt-2">Select date → mark present/absent/late/excused per learner</p>
            </div>
          </div>
        </div>

        <div>
          <div className="card p-6">
            <h3 className="font-semibold mb-4">Attendance Stats</h3>
            <div className="space-y-4">
              <div className="flex justify-between">
                <span className="text-sm text-gray-500">Today Present</span>
                <span className="font-medium">0</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-gray-500">Today Absent</span>
                <span className="font-medium text-red-600">0</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-gray-500">Late</span>
                <span className="font-medium text-yellow-600">0</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-gray-500">Excused</span>
                <span className="font-medium text-blue-600">0</span>
              </div>
              <div className="border-t pt-2">
                <div className="flex justify-between">
                  <span className="text-sm font-medium">Attendance Rate</span>
                  <span className="font-medium">0%</span>
                </div>
              </div>
            </div>
          </div>

          <div className="card p-6 mt-6">
            <h3 className="font-semibold mb-4">
              <AlertTriangle className="w-4 h-4 inline mr-1 text-yellow-500" />
              Chronic Absenteeism
            </h3>
            <div className="border rounded-lg p-6 text-center text-gray-400">
              <p className="text-sm">Learners below 75% attendance threshold</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}