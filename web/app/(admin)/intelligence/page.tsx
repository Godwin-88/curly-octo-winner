'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { TrendingUp, MessageSquare, Brain, Wallet, BarChart3 } from 'lucide-react';
import { api, FeeCollectionSummary, ChannelReach, CampaignDeliverySummary } from '@/lib/api';

export default function IntelligenceDashboardPage() {
  const token = ''; // TODO: Get from auth context
  const [feeSummary, setFeeSummary] = useState<FeeCollectionSummary[]>([]);
  const [channelReach, setChannelReach] = useState<ChannelReach[]>([]);
  const [campaigns, setCampaigns] = useState<CampaignDeliverySummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [fee, reach, camp] = await Promise.all([
        api.getFeeCollectionSummary({ term: 1, year: 2026 }, token),
        api.getChannelReach(token),
        api.getCampaignDeliverySummary(token),
      ]);
      setFeeSummary(fee);
      setChannelReach(reach);
      setCampaigns(camp);
    } catch (e: any) {
      setError(e.message || 'Failed to load intelligence data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const totalCollected = feeSummary.reduce((s, f) => s + f.total_collected_cents, 0);
  const totalOutstanding = feeSummary.reduce((s, f) => s + f.outstanding_cents, 0);
  const totalRecipients = channelReach.reduce((s, c) => s + c.total_recipients, 0);
  const totalDelivered = channelReach.reduce((s, c) => s + c.total_delivered, 0);
  const avgDeliveryRate = campaigns.length ? campaigns.reduce((s, c) => s + c.delivery_rate, 0) / campaigns.length : 0;

  const stats = [
    { label: 'Fees Collected', value: `KES ${(totalCollected / 100).toLocaleString()}`, icon: Wallet, color: 'text-green-600 bg-green-50' },
    { label: 'Outstanding Fees', value: `KES ${(totalOutstanding / 100).toLocaleString()}`, icon: TrendingUp, color: 'text-red-600 bg-red-50' },
    { label: 'Message Recipients', value: String(totalRecipients), icon: MessageSquare, color: 'text-blue-600 bg-blue-50' },
    { label: 'Avg Delivery Rate', value: `${avgDeliveryRate.toFixed(1)}%`, icon: BarChart3, color: 'text-indigo-600 bg-indigo-50' },
  ];

  const cards = [
    { href: '/intelligence/financial', label: 'Financial Analytics', desc: 'Fee collection, payment channels, defaulters', icon: Wallet, color: 'text-green-600 bg-green-50' },
    { href: '/intelligence/communications', label: 'Communication Analytics', desc: 'Campaign delivery, channel reach, failed numbers', icon: MessageSquare, color: 'text-blue-600 bg-blue-50' },
    { href: '/intelligence/ai', label: 'AI Assistant', desc: 'FAQ knowledge base, template suggestions, auto-response', icon: Brain, color: 'text-purple-600 bg-purple-50' },
  ];

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Digital Intelligence</h1>
          <p className="text-gray-500">Analytics and AI-powered insights for school leadership</p>
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

          <div className="mt-8 grid grid-cols-1 md:grid-cols-3 gap-4">
            {cards.map((c) => {
              const Icon = c.icon;
              return (
                <Link key={c.href} href={c.href} className="bg-white rounded-lg shadow border p-6 hover:shadow-md transition-shadow">
                  <div className={`w-10 h-10 rounded-lg flex items-center justify-center mb-3 ${c.color}`}>
                    <Icon size={20} />
                  </div>
                  <h2 className="text-lg font-semibold">{c.label}</h2>
                  <p className="text-sm text-gray-500 mt-1">{c.desc}</p>
                </Link>
              );
            })}
          </div>

          <div className="mt-8 bg-white rounded-lg shadow border overflow-hidden">
            <div className="p-6 pb-0">
              <h2 className="text-lg font-semibold">Recent Campaign Delivery</h2>
              <p className="text-sm text-gray-500">Latest communication campaigns and their delivery rates</p>
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
                      <th className="px-3 py-2">Recipients</th>
                      <th className="px-3 py-2">Delivered</th>
                      <th className="px-3 py-2">Failed</th>
                      <th className="px-3 py-2">Rate</th>
                    </tr>
                  </thead>
                  <tbody>
                    {campaigns.slice(0, 10).map((c) => (
                      <tr key={c.message_id} className="border-t">
                        <td className="px-3 py-2 uppercase">{c.channel}</td>
                        <td className="px-3 py-2">{c.audience_type}</td>
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
        </>
      )}
    </div>
  );
}