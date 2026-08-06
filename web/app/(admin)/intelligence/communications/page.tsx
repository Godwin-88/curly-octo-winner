'use client';

import { useEffect, useState } from 'react';
import { MessageSquare, CheckCircle2, XCircle, BarChart3 } from 'lucide-react';
import { api, CampaignDeliverySummary, ChannelReach, FailedNumber } from '@/lib/api';

export default function CommunicationAnalyticsPage() {
  const token = ''; // TODO: Get from auth context
  const [campaigns, setCampaigns] = useState<CampaignDeliverySummary[]>([]);
  const [reach, setReach] = useState<ChannelReach[]>([]);
  const [failed, setFailed] = useState<FailedNumber[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [camp, ch, fail] = await Promise.all([
        api.getCampaignDeliverySummary(token),
        api.getChannelReach(token),
        api.getFailedNumbers(token),
      ]);
      setCampaigns(camp);
      setReach(ch);
      setFailed(fail);
    } catch (e: any) {
      setError(e.message || 'Failed to load communication analytics');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const totalRecipients = reach.reduce((s, c) => s + c.total_recipients, 0);
  const totalDelivered = reach.reduce((s, c) => s + c.total_delivered, 0);
  const totalFailed = reach.reduce((s, c) => s + c.total_failed, 0);
  const overallRate = totalRecipients ? (totalDelivered / totalRecipients) * 100 : 0;

  const stats = [
    { label: 'Total Recipients', value: String(totalRecipients), icon: MessageSquare, color: 'text-blue-600 bg-blue-50' },
    { label: 'Delivered', value: String(totalDelivered), icon: CheckCircle2, color: 'text-green-600 bg-green-50' },
    { label: 'Failed', value: String(totalFailed), icon: XCircle, color: 'text-red-600 bg-red-50' },
    { label: 'Overall Delivery', value: `${overallRate.toFixed(1)}%`, icon: BarChart3, color: 'text-indigo-600 bg-indigo-50' },
  ];

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Communication Analytics</h1>
          <p className="text-gray-500">Campaign delivery, channel reach, and failed numbers</p>
        </div>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {loading ? (
        <p className="text-gray-500 mt-4">Loading...</p>
      ) : (
        <>
          <div className="mt-6 grid grid-cols-1 md:grid-cols-4 gap-4">
            {stats.map((s) => {
              const Icon = s.icon;
              return (
                <div key={s.label} className="bg-white rounded-lg shadow border p-4">
                  <div className={`w-10 h-10 rounded-lg flex items-center justify-center mb-3 ${s.color}`}>
                    <Icon size={20} />
                  </div>
                  <p className="text-sm text-gray-500">{s.label}</p>
                  <p className="text-2xl font-bold">{s.value}</p>
                </div>
              );
            })}
          </div>

          {/* Channel reach */}
          <div className="mt-8 bg-white rounded-lg shadow border overflow-hidden">
            <div className="p-6 pb-0">
              <h2 className="text-lg font-semibold">Channel Reach</h2>
              <p className="text-sm text-gray-500">Campaigns and recipients per channel</p>
            </div>
            <div className="p-6">
              {reach.length === 0 ? (
                <p className="text-gray-400 text-sm">No channel reach data available.</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-left text-gray-500">
                    <tr>
                      <th className="px-3 py-2">Channel</th>
                      <th className="px-3 py-2">Campaigns</th>
                      <th className="px-3 py-2">Recipients</th>
                      <th className="px-3 py-2">Delivered</th>
                      <th className="px-3 py-2">Failed</th>
                    </tr>
                  </thead>
                  <tbody>
                    {reach.map((c) => (
                      <tr key={c.channel} className="border-t">
                        <td className="px-3 py-2 uppercase font-medium">{c.channel}</td>
                        <td className="px-3 py-2">{c.campaign_count}</td>
                        <td className="px-3 py-2">{c.total_recipients}</td>
                        <td className="px-3 py-2 text-green-600">{c.total_delivered}</td>
                        <td className="px-3 py-2 text-red-600">{c.total_failed}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>

          {/* Campaign delivery */}
          <div className="mt-8 bg-white rounded-lg shadow border overflow-hidden">
            <div className="p-6 pb-0">
              <h2 className="text-lg font-semibold">Campaign Delivery</h2>
              <p className="text-sm text-gray-500">Delivery rate by campaign</p>
            </div>
            <div className="p-6">
              {campaigns.length === 0 ? (
                <p className="text-gray-400 text-sm">No campaign data available.</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-left text-gray-500">
                    <tr>
                      <th className="px-3 py-2">Channel</th>
                      <th className="px-3 py-2">Audience</th>
                      <th className="px-3 py-2">Status</th>
                      <th className="px-3 py-2">Recipients</th>
                      <th className="px-3 py-2">Delivered</th>
                      <th className="px-3 py-2">Failed</th>
                      <th className="px-3 py-2">Rate</th>
                    </tr>
                  </thead>
                  <tbody>
                    {campaigns.map((c) => (
                      <tr key={c.message_id} className="border-t">
                        <td className="px-3 py-2 uppercase">{c.channel}</td>
                        <td className="px-3 py-2">{c.audience_type}</td>
                        <td className="px-3 py-2">{c.status}</td>
                        <td className="px-3 py-2">{c.recipient_count}</td>
                        <td className="px-3 py-2">{c.delivered_count}</td>
                        <td className="px-3 py-2">{c.failed_count}</td>
                        <td className="px-3 py-2">
                          <span className={`px-2 py-1 rounded-full text-xs ${c.delivery_rate >= 90 ? 'bg-green-50 text-green-700' : c.delivery_rate >= 70 ? 'bg-yellow-50 text-yellow-700' : 'bg-red-50 text-red-700'}`}>
                            {c.delivery_rate.toFixed(1)}%
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>

          {/* Failed numbers */}
          <div className="mt-8 bg-white rounded-lg shadow border overflow-hidden">
            <div className="p-6 pb-0">
              <h2 className="text-lg font-semibold">Failed Number Analysis</h2>
              <p className="text-sm text-gray-500">Failed deliveries with error codes for data quality review</p>
            </div>
            <div className="p-6">
              {failed.length === 0 ? (
                <p className="text-gray-400 text-sm">No failed deliveries recorded.</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-left text-gray-500">
                    <tr>
                      <th className="px-3 py-2">Phone</th>
                      <th className="px-3 py-2">Channel</th>
                      <th className="px-3 py-2">Error Code</th>
                      <th className="px-3 py-2">Error Message</th>
                      <th className="px-3 py-2">Time</th>
                    </tr>
                  </thead>
                  <tbody>
                    {failed.slice(0, 20).map((f, i) => (
                      <tr key={i} className="border-t">
                        <td className="px-3 py-2 font-mono">{f.phone}</td>
                        <td className="px-3 py-2 uppercase">{f.channel}</td>
                        <td className="px-3 py-2 text-red-600">{f.error_code || '—'}</td>
                        <td className="px-3 py-2">{f.error_message || '—'}</td>
                        <td className="px-3 py-2">{new Date(f.created_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}