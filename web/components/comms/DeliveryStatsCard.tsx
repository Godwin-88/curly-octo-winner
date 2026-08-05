'use client';

import { DeliveryStats } from '@/lib/api';

interface Props {
  stats: DeliveryStats;
}

export default function DeliveryStatsCard({ stats }: Props) {
  return (
    <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
      <div className="card p-4">
        <h4 className="text-xs text-gray-500">Total</h4>
        <p className="text-2xl font-bold mt-1">{stats.total}</p>
      </div>
      <div className="card p-4">
        <h4 className="text-xs text-gray-500">Sent</h4>
        <p className="text-2xl font-bold mt-1 text-blue-600">{stats.sent}</p>
      </div>
      <div className="card p-4">
        <h4 className="text-xs text-gray-500">Delivered</h4>
        <p className="text-2xl font-bold mt-1 text-green-600">{stats.delivered}</p>
      </div>
      <div className="card p-4">
        <h4 className="text-xs text-gray-500">Failed</h4>
        <p className="text-2xl font-bold mt-1 text-red-600">{stats.failed}</p>
      </div>
      <div className="card p-4">
        <h4 className="text-xs text-gray-500">Rate</h4>
        <p className="text-2xl font-bold mt-1">{stats.delivery_rate.toFixed(1)}%</p>
      </div>
    </div>
  );
}