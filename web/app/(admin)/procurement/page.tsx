'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import {
  api,
  Supplier,
  PurchaseRequisition,
  PurchaseOrder,
  GoodsReceipt,
  SupplierPayment,
} from '@/lib/api';

const token = ''; // TODO: Get from auth context

export default function ProcurementPage() {
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [requisitions, setRequisitions] = useState<PurchaseRequisition[]>([]);
  const [orders, setOrders] = useState<PurchaseOrder[]>([]);
  const [receipts, setReceipts] = useState<GoodsReceipt[]>([]);
  const [payments, setPayments] = useState<SupplierPayment[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const [sup, req, po, grn, pay] = await Promise.all([
          api.listSuppliers({}, token),
          api.listRequisitions({}, token),
          api.listPurchaseOrders({}, token),
          api.listGoodsReceipts({}, token),
          api.listSupplierPayments({}, token),
        ]);
        setSuppliers(sup);
        setRequisitions(req);
        setOrders(po);
        setReceipts(grn);
        setPayments(pay);
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    }
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const pendingReqs = requisitions.filter((r) => r.status === 'pending' || r.status === 'hod_approved').length;
  const activeOrders = orders.filter((o) => o.status === 'draft' || o.status === 'sent').length;
  const pendingPayments = payments.filter((p) => p.status === 'pending' || p.status === 'authorised').length;
  const totalSpend = payments.filter((p) => p.status === 'paid').reduce((sum, p) => sum + p.amount_cents, 0);

  const stats = [
    { label: 'Active Suppliers', value: suppliers.filter((s) => s.is_active).length, href: '/procurement/suppliers' },
    { label: 'Pending Requisitions', value: pendingReqs, href: '/procurement/requisitions' },
    { label: 'Open Purchase Orders', value: activeOrders, href: '/procurement/orders' },
    { label: 'Pending Payments', value: pendingPayments, href: '/procurement/payments' },
  ];

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Procurement</h1>
          <p className="text-gray-500">Supplier registry, requisitions, purchase orders, goods receipt & payments</p>
        </div>
        <Link href="/procurement/requisitions" className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700">
          New Requisition
        </Link>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {stats.map((s) => (
          <Link key={s.label} href={s.href} className="bg-white p-4 rounded-lg shadow border border-gray-200 hover:border-blue-400">
            <p className="text-sm text-gray-500">{s.label}</p>
            <p className="text-2xl font-bold mt-1">{s.value}</p>
          </Link>
        ))}
      </div>

      <div className="bg-white p-4 rounded-lg shadow border border-gray-200">
        <h2 className="font-semibold mb-2">Total Paid to Suppliers</h2>
        <p className="text-3xl font-bold text-green-600">KSh {(totalSpend / 100).toLocaleString()}</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white p-4 rounded-lg shadow border border-gray-200">
          <h2 className="font-semibold mb-3">Recent Requisitions</h2>
          {requisitions.length === 0 ? (
            <p className="text-gray-400 text-sm">No requisitions yet</p>
          ) : (
            <ul className="divide-y divide-gray-100">
              {requisitions.slice(0, 5).map((r) => (
                <li key={r.id} className="py-2 flex items-center justify-between">
                  <div>
                    <p className="font-medium text-sm">{r.title}</p>
                    <p className="text-xs text-gray-500">{r.requisition_no} · {r.department || 'General'}</p>
                  </div>
                  <span className={`text-xs px-2 py-1 rounded-full ${statusColor(r.status)}`}>{r.status}</span>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="bg-white p-4 rounded-lg shadow border border-gray-200">
          <h2 className="font-semibold mb-3">Recent Purchase Orders</h2>
          {orders.length === 0 ? (
            <p className="text-gray-400 text-sm">No purchase orders yet</p>
          ) : (
            <ul className="divide-y divide-gray-100">
              {orders.slice(0, 5).map((o) => (
                <li key={o.id} className="py-2 flex items-center justify-between">
                  <div>
                    <p className="font-medium text-sm">{o.po_number}</p>
                    <p className="text-xs text-gray-500">{o.supplier_name} · KSh {(o.total_amount_cents / 100).toLocaleString()}</p>
                  </div>
                  <span className={`text-xs px-2 py-1 rounded-full ${statusColor(o.status)}`}>{o.status}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      {loading && <p className="text-center text-gray-400">Loading...</p>}
    </div>
  );
}

function statusColor(status: string) {
  const map: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-700',
    hod_approved: 'bg-blue-100 text-blue-700',
    approved: 'bg-green-100 text-green-700',
    rejected: 'bg-red-100 text-red-700',
    cancelled: 'bg-gray-100 text-gray-600',
    ordered: 'bg-purple-100 text-purple-700',
    draft: 'bg-gray-100 text-gray-600',
    sent: 'bg-blue-100 text-blue-700',
    partially_received: 'bg-yellow-100 text-yellow-700',
    received: 'bg-green-100 text-green-700',
    paid: 'bg-green-100 text-green-700',
    authorised: 'bg-blue-100 text-blue-700',
    pending_payment: 'bg-yellow-100 text-yellow-700',
  };
  return map[status] || 'bg-gray-100 text-gray-600';
}