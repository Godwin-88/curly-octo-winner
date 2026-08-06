'use client';

import { useEffect, useState } from 'react';
import { Users, Wallet, CalendarClock, ClipboardCheck } from 'lucide-react';
import { api, StaffProfile, PayrollRun, LeaveRequest, StaffAppraisal } from '@/lib/api';

export default function HRPage() {
  const token = ''; // TODO: Get from auth context
  const [staff, setStaff] = useState<StaffProfile[]>([]);
  const [payroll, setPayroll] = useState<PayrollRun[]>([]);
  const [leave, setLeave] = useState<LeaveRequest[]>([]);
  const [appraisals, setAppraisals] = useState<StaffAppraisal[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [staffData, payrollData, leaveData, appraisalData] = await Promise.all([
        api.listStaff({}, token),
        api.listPayrollRuns({ year: 2026 }, token),
        api.listLeaveRequests({}, token),
        api.listAppraisals({ year: 2026 }, token),
      ]);
      setStaff(staffData);
      setPayroll(payrollData);
      setLeave(leaveData);
      setAppraisals(appraisalData);
    } catch (e: any) {
      setError(e.message || 'Failed to load HR data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const totalPayroll = payroll.reduce((sum, p) => sum + p.net_cents, 0);
  const pendingLeave = leave.filter((l) => l.status === 'pending').length;
  const activeStaff = staff.filter((s) => s.is_active).length;

  const stats = [
    { label: 'Active Staff', value: activeStaff, icon: Users, color: 'bg-blue-50 text-blue-700' },
    { label: 'Net Payroll (KES)', value: `KES ${(totalPayroll / 100).toLocaleString()}`, icon: Wallet, color: 'bg-green-50 text-green-700' },
    { label: 'Pending Leave', value: pendingLeave, icon: CalendarClock, color: 'bg-yellow-50 text-yellow-700' },
    { label: 'Appraisals', value: appraisals.length, icon: ClipboardCheck, color: 'bg-purple-50 text-purple-700' },
  ];

  return (
    <div className="p-6">
      <div>
        <h1 className="text-2xl font-bold">Human Resources</h1>
        <p className="text-gray-500">Staff records, payroll, leave & performance</p>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {loading ? (
        <p className="text-gray-500 mt-4">Loading...</p>
      ) : (
        <>
          <div className="mt-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {stats.map((s) => {
              const Icon = s.icon;
              return (
                <div key={s.label} className="bg-white rounded-lg shadow border p-4">
                  <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${s.color}`}>
                    <Icon size={20} />
                  </div>
                  <p className="mt-3 text-2xl font-bold">{s.value}</p>
                  <p className="text-sm text-gray-500">{s.label}</p>
                </div>
              );
            })}
          </div>

          <div className="mt-6 grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="bg-white rounded-lg shadow border overflow-hidden">
              <div className="px-4 py-3 border-b font-semibold">Recent Leave Requests</div>
              <table className="w-full text-sm">
                <thead className="bg-gray-50 text-left text-gray-500">
                  <tr>
                    <th className="px-4 py-2">Staff</th>
                    <th className="px-4 py-2">Type</th>
                    <th className="px-4 py-2">Days</th>
                    <th className="px-4 py-2">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {leave.slice(0, 5).map((l) => (
                    <tr key={l.id} className="border-t">
                      <td className="px-4 py-2 font-medium">{l.staff_name}</td>
                      <td className="px-4 py-2 capitalize">{l.leave_type}</td>
                      <td className="px-4 py-2">{l.days}</td>
                      <td className="px-4 py-2">
                        <span className={`px-2 py-1 rounded-full text-xs ${
                          l.status === 'approved' ? 'bg-green-50 text-green-700' :
                          l.status === 'pending' ? 'bg-yellow-50 text-yellow-700' :
                          l.status === 'denied' ? 'bg-red-50 text-red-700' : 'bg-gray-100 text-gray-600'
                        }`}>{l.status}</span>
                      </td>
                    </tr>
                  ))}
                  {leave.length === 0 && (
                    <tr><td colSpan={4} className="px-4 py-6 text-center text-gray-400">No leave requests yet.</td></tr>
                  )}
                </tbody>
              </table>
            </div>

            <div className="bg-white rounded-lg shadow border overflow-hidden">
              <div className="px-4 py-3 border-b font-semibold">Recent Payroll</div>
              <table className="w-full text-sm">
                <thead className="bg-gray-50 text-left text-gray-500">
                  <tr>
                    <th className="px-4 py-2">Staff</th>
                    <th className="px-4 py-2">Period</th>
                    <th className="px-4 py-2">Net (KES)</th>
                    <th className="px-4 py-2">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {payroll.slice(0, 5).map((p) => (
                    <tr key={p.id} className="border-t">
                      <td className="px-4 py-2 font-medium">{p.staff_name}</td>
                      <td className="px-4 py-2">{new Date(p.year, p.month - 1).toLocaleString('default', { month: 'short' })} {p.year}</td>
                      <td className="px-4 py-2">{(p.net_cents / 100).toLocaleString()}</td>
                      <td className="px-4 py-2">
                        <span className={`px-2 py-1 rounded-full text-xs ${
                          p.status === 'paid' ? 'bg-green-50 text-green-700' :
                          p.status === 'approved' ? 'bg-blue-50 text-blue-700' : 'bg-yellow-50 text-yellow-700'
                        }`}>{p.status}</span>
                      </td>
                    </tr>
                  ))}
                  {payroll.length === 0 && (
                    <tr><td colSpan={4} className="px-4 py-6 text-center text-gray-400">No payroll runs yet.</td></tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}
    </div>
  );
}