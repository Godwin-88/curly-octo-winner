import { ShieldCheck, Users, KeyRound, FileBarChart, ClipboardList, UserCheck } from 'lucide-react';
import Link from 'next/link';

export default function SecurityDashboardPage() {
  const cards = [
    { href: '/security/roles', label: 'Role-Based Access', desc: 'Manage roles, permissions, and the RBAC matrix', icon: ShieldCheck, color: 'text-blue-600 bg-blue-50' },
    { href: '/security/audit', label: 'Audit Log', desc: 'Track every create, update, and delete with actor & IP', icon: ClipboardList, color: 'text-indigo-600 bg-indigo-50' },
    { href: '/security/consent', label: 'Parent Consent', desc: 'WhatsApp opt-in & data processing consent management', icon: UserCheck, color: 'text-green-600 bg-green-50' },
    { href: '/security/data-protection', label: 'Data Protection', desc: 'KDPA processing register & data subject rights', icon: FileBarChart, color: 'text-purple-600 bg-purple-50' },
  ];

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Digital Security & Compliance</h1>
          <p className="text-gray-500">RBAC, audit logs, and Kenya Data Protection Act compliance</p>
        </div>
      </div>

      <div className="mt-8 grid grid-cols-1 md:grid-cols-2 gap-4">
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

      <div className="mt-8 bg-white rounded-lg shadow border p-6">
        <div className="flex items-center gap-3 mb-3">
          <div className="w-10 h-10 rounded-lg flex items-center justify-center bg-gray-50">
            <KeyRound size={20} className="text-gray-600" />
          </div>
          <div>
            <h2 className="text-lg font-semibold">Security Overview</h2>
            <p className="text-sm text-gray-500">Key compliance highlights</p>
          </div>
        </div>
        <ul className="space-y-3 text-sm text-gray-600">
          <li className="flex items-center gap-2">
            <Users size={16} className="text-blue-600" />
            Role-based access control with deny-by-default permissions for Super Admin, Principal, Teacher, Bursar, Transport Manager, and HR.
          </li>
          <li className="flex items-center gap-2">
            <ClipboardList size={16} className="text-indigo-600" />
            Every create/update/delete is logged with actor, timestamp, IP address, and user agent.
          </li>
          <li className="flex items-center gap-2">
            <UserCheck size={16} className="text-green-600" />
            Explicit parent consent captured for WhatsApp opt-in and data processing, per Kenya Data Protection Act 2019.
          </li>
          <li className="flex items-center gap-2">
            <FileBarChart size={16} className="text-purple-600" />
            Data processing register documents legal basis, retention, and third-party transfers.
          </li>
        </ul>
      </div>
    </div>
  );
}