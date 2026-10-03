'use client';

import { useQuery } from '@tanstack/react-query';
import Link from 'next/link';
import { useState } from 'react';
import { apiRequest } from '@/lib/api';
import { page, resource, type Ctx, type FormValues, type ModuleManifest, type Option } from '@/framework/types';
import { formatPhone } from '@/lib/phone';
import { pathTo } from '@/shell/context';
import { Button, Card, ErrorNote, SimpleTable, Spinner, Status, Tile, when } from '@/ui/kit';

// Fees: what each grade is charged, who has been billed, what has been paid
// and what is still owed. The API keeps the rules (a learner is billed once a
// term, a payment is never deleted, M-Pesa counts only when Safaricom says
// so); this file only describes the screens.

const PAGE = 50;
const FINANCE_ROLES = ['super_admin', 'principal', 'bursar'];

// --- Shapes -----------------------------------------------------------------

interface FeeItem {
  id: string;
  name: string;
  amount_cents: number;
  item_type: string;
  is_optional: boolean;
}

interface FeeStructure {
  id: string;
  name: string;
  grade: string;
  term: number;
  year: number;
  total_cents: number;
  active: boolean;
  notes?: string;
  items?: FeeItem[];
  created_at: string;
}

interface Invoice {
  id: string;
  learner_id: string;
  fee_structure_id?: string;
  invoice_number: string;
  term: number;
  year: number;
  issue_date: string;
  due_date?: string;
  total_cents: number;
  discount_cents: number;
  paid_cents: number;
  balance_cents: number;
  credit_cents: number;
  status: string;
  notes?: string;
  learner_name: string;
  grade: string;
  stream?: string;
  learner_upi?: string;
  voided_at?: string;
  void_reason?: string;
  items?: FeeItem[];
  created_at: string;
}

interface Payment {
  id: string;
  invoice_id: string;
  amount_cents: number;
  channel: string;
  status: string;
  reference?: string;
  paid_by?: string;
  phone?: string;
  paid_at?: string;
  notes?: string;
  mpesa_receipt?: string;
  mpesa_result_desc?: string;
  receipt_number?: string;
  failure_code?: string;
  reversed_at?: string;
  reversal_reason?: string;
  invoice_number: string;
  learner_id: string;
  learner_name: string;
  grade: string;
  created_at: string;
}

interface Discount {
  id: string;
  amount_cents: number;
  discount_type: string;
  reason?: string;
  created_at: string;
}

interface InboxPayment {
  id: string;
  trans_id: string;
  trans_time: string;
  amount_cents: number;
  allocated_cents: number;
  remaining_cents: number;
  bill_ref?: string;
  payer_name?: string;
  payer_phone?: string;
  learner_id?: string;
  learner_name?: string;
  status: string;
  allocations?: Payment[];
}

interface ArrearsRow {
  learner_id: string;
  learner_name: string;
  grade: string;
  stream: string;
  invoices: number;
  balance_cents: number;
  oldest_due?: string;
}

interface Summary {
  invoices: number;
  billed_cents: number;
  discount_cents: number;
  collected_cents: number;
  outstanding_cents: number;
  overdue_cents: number;
  learners_owing: number;
  by_channel: Record<string, number>;
  unmatched_count: number;
  unmatched_cents: number;
  pending_mpesa: number;
}

interface Statement {
  learner_name: string;
  grade: string;
  stream: string;
  billed_cents: number;
  paid_cents: number;
  balance_cents: number;
  invoices: Invoice[];
  payments: Payment[];
}

interface BulkResult {
  learners: number;
  created: number;
  already_invoiced: number;
  each_cents: number;
  total_cents: number;
}

// --- Words and numbers ------------------------------------------------------

const SHILLINGS = new Intl.NumberFormat('en-KE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });

/** An amount in cents as it is read: "KES 17,500.00". */
export function kes(cents: number | undefined | null): string {
  return `KES ${SHILLINGS.format((cents ?? 0) / 100)}`;
}

const Money = ({ cents }: { cents: number | undefined | null }) => (
  <span className="whitespace-nowrap tabular-nums">{kes(cents)}</span>
);

const DAY = new Intl.DateTimeFormat('en-KE', { timeZone: 'Africa/Nairobi', day: 'numeric', month: 'short', year: 'numeric' });

function day(value: string | undefined | null): string {
  if (!value) return '—';
  const date = new Date(value.length === 10 ? `${value}T12:00:00+03:00` : value);
  return Number.isNaN(date.getTime()) ? '—' : DAY.format(date);
}

const GRADES = ['PP1', 'PP2', 'Grade 1', 'Grade 2', 'Grade 3', 'Grade 4', 'Grade 5', 'Grade 6', 'Grade 7', 'Grade 8', 'Grade 9'];
const GRADE_OPTIONS: Option[] = GRADES.map((grade) => ({ value: grade, label: grade }));
const TERM_OPTIONS: Option[] = [1, 2, 3].map((term) => ({ value: String(term), label: `Term ${term}` }));

const CHANNEL: Record<string, string> = { mpesa: 'M-Pesa', cash: 'Cash', bank: 'Bank', cheque: 'Cheque' };
const ITEM_TYPES: Option[] = [
  { value: 'tuition', label: 'Tuition' },
  { value: 'activity', label: 'Activity' },
  { value: 'transport', label: 'Transport' },
  { value: 'boarding', label: 'Boarding' },
  { value: 'caution', label: 'Caution money' },
  { value: 'other', label: 'Other' },
];

/** Why an M-Pesa request did not end in a payment, in the bursar's words. */
const FAILURE: Record<string, string> = {
  CANCELLED: 'The parent cancelled the request on their phone.',
  NO_RESPONSE: 'The parent did not respond; the phone may be off or out of reach.',
  INSUFFICIENT_FUNDS: 'There was not enough money in the M-Pesa account.',
  WRONG_PIN: 'The wrong M-Pesa PIN was entered.',
  EXPIRED: 'The request timed out before the parent answered.',
  BUSY: 'The phone was busy with another M-Pesa request. Try again in a minute.',
  REJECTED: 'Safaricom refused the request, so the parent was never asked.',
  DECLINED: 'M-Pesa declined the payment.',
  OUTCOME_UNKNOWN: 'Safaricom did not report a result. Check the M-Pesa statement before asking the parent to pay again.',
};

function workspaceLink(ctx: Ctx, section: string, record: string, label: string) {
  return <Link className="font-semibold text-blue-700 underline" href={pathTo(ctx, 'finance', section, record)}>{label}</Link>;
}

// --- Overview ---------------------------------------------------------------

function Overview({ ctx }: { ctx: Ctx }) {
  const thisYear = new Date().getFullYear();
  const [term, setTerm] = useState('');
  const [year, setYear] = useState(String(thisYear));
  const summary = useQuery({
    queryKey: ['finance', ctx.schoolId, 'summary', term, year],
    queryFn: () => apiRequest<Summary>(`/finance/summary?term=${term || 0}&year=${year || 0}`),
  });
  const data = summary.data;
  const rate = data && data.billed_cents - data.discount_cents > 0
    ? Math.round((data.collected_cents / (data.billed_cents - data.discount_cents)) * 100)
    : 0;

  return (
    <div className="space-y-4 p-4">
      <div className="flex flex-wrap items-end gap-3">
        <label className="text-sm font-semibold">
          Year
          <select className="mt-1 block rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-sm font-normal" value={year} onChange={(event) => setYear(event.target.value)}>
            <option value="">All years</option>
            {[thisYear + 1, thisYear, thisYear - 1, thisYear - 2].map((y) => <option key={y} value={y}>{y}</option>)}
          </select>
        </label>
        <label className="text-sm font-semibold">
          Term
          <select className="mt-1 block rounded-lg border border-gray-300 bg-white px-2 py-1.5 text-sm font-normal" value={term} onChange={(event) => setTerm(event.target.value)}>
            <option value="">All terms</option>
            {TERM_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
          </select>
        </label>
      </div>

      {summary.isLoading && <Spinner />}
      {summary.error && <ErrorNote error={summary.error} />}
      {data && (
        <>
          <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
            <Tile label="Billed" value={kes(data.billed_cents - data.discount_cents)} note={`${data.invoices} invoice${data.invoices === 1 ? '' : 's'}${data.discount_cents > 0 ? `, after ${kes(data.discount_cents)} in discounts` : ''}`} />
            <Tile label="Collected" value={kes(data.collected_cents)} note={`${rate}% of what was billed`} />
            <Tile label="Still owed" value={kes(data.outstanding_cents)} note={`${data.learners_owing} learner${data.learners_owing === 1 ? '' : 's'}`} />
            <Tile label="Past its due date" value={kes(data.overdue_cents)} note="Part of what is still owed" />
          </div>

          {(data.unmatched_count > 0 || data.pending_mpesa > 0) && (
            <div className="space-y-2">
              {data.unmatched_count > 0 && (
                <p role="status" className="rounded-lg border border-amber-300 bg-amber-50 p-3 text-sm text-amber-950">
                  <strong>{kes(data.unmatched_cents)}</strong> paid to the paybill in {data.unmatched_count} payment{data.unmatched_count === 1 ? '' : 's'} is not yet against any invoice.{' '}
                  <Link className="font-semibold underline" href={pathTo(ctx, 'finance', 'paybill')}>Allocate it</Link>
                </p>
              )}
              {data.pending_mpesa > 0 && (
                <p role="status" className="rounded-lg border border-sky-300 bg-sky-50 p-3 text-sm text-sky-950">
                  {data.pending_mpesa} M-Pesa request{data.pending_mpesa === 1 ? ' is' : 's are'} waiting for a parent to enter their PIN.
                </p>
              )}
            </div>
          )}

          <Card title="Collected, by how it was paid">
            <SimpleTable
              caption="Collected by payment method"
              rows={Object.entries(data.by_channel).sort((a, b) => b[1] - a[1])}
              empty="Nothing has been collected for this period yet."
              columns={[
                { header: 'Paid by', cell: ([channel]) => CHANNEL[channel] ?? channel },
                { header: 'Amount', align: 'right', cell: ([, cents]) => <Money cents={cents} /> },
              ]}
            />
          </Card>
        </>
      )}
    </div>
  );
}

const overview = page({
  id: 'overview',
  label: 'Overview',
  scopes: ['school'],
  roles: FINANCE_ROLES,
  purpose: 'What has been billed, collected and is still owed.',
  page: Overview,
});

// --- Fee structures ---------------------------------------------------------

function FeeItems({ row }: { row: FeeStructure; ctx: Ctx }) {
  return (
    <Card title="What is charged">
      <SimpleTable
        caption="Fee items"
        rows={row.items ?? []}
        empty="No items yet."
        columns={[
          { header: 'Item', cell: (item) => item.name },
          { header: 'Kind', cell: (item) => ITEM_TYPES.find((type) => type.value === item.item_type)?.label ?? item.item_type },
          { header: 'Billed', cell: (item) => (item.is_optional ? 'Only when chosen' : 'To every learner') },
          { header: 'Amount', align: 'right', cell: (item) => <Money cents={item.amount_cents} /> },
        ]}
      />
    </Card>
  );
}

const bulkBody = (row: FeeStructure, values: FormValues, dryRun: boolean) => ({
  fee_structure_id: row.id,
  due_date: values.due_date,
  stream: values.stream,
  include_optional: Boolean(values.include_optional),
  dry_run: dryRun,
});

const feeStructures = resource<FeeStructure>({
  id: 'fees',
  label: 'Fee structures',
  noun: 'fee structure',
  scopes: ['school'],
  roles: FINANCE_ROLES,
  purpose: 'What each grade is charged for a term. Learners are billed from these.',
  list: async (_ctx, filters) => {
    const query = new URLSearchParams();
    if (filters.grade) query.set('grade', filters.grade);
    if (filters.year) query.set('year', filters.year);
    return { items: (await apiRequest<FeeStructure[] | null>(`/fee-structures?${query}`)) ?? [] };
  },
  get: (_ctx, id) => apiRequest<FeeStructure>(`/fee-structures/${id}`),
  rowId: (row) => row.id,
  title: (row) => row.name,
  status: (row) => (row.active ? 'active' : 'switched off'),
  filters: [
    { name: 'grade', label: 'Grade', type: 'select', options: GRADE_OPTIONS },
    { name: 'year', label: 'Year', type: 'number', placeholder: String(new Date().getFullYear()) },
  ],
  columns: [
    { header: 'Fee structure', cell: (row) => row.name },
    { header: 'For', cell: (row) => <span className="whitespace-nowrap">{row.grade}, Term {row.term} {row.year}</span> },
    { header: 'Total', align: 'right', cell: (row) => <Money cents={row.total_cents} /> },
  ],
  fields: [
    { label: 'Grade', value: (row) => row.grade },
    { label: 'Term', value: (row) => `Term ${row.term}, ${row.year}` },
    { label: 'Total of all items', value: (row) => kes(row.total_cents) },
    { label: 'Notes', value: (row) => row.notes || '—' },
  ],
  extra: FeeItems,
  create: {
    id: 'create',
    label: 'New fee structure',
    initial: () => ({ year: String(new Date().getFullYear()) }),
    fields: [
      { name: 'grade', label: 'Grade', type: 'select', required: true, options: GRADE_OPTIONS },
      { name: 'term', label: 'Term', type: 'select', required: true, options: TERM_OPTIONS },
      { name: 'year', label: 'Year', type: 'number', required: true },
      { name: 'name', label: 'Name', type: 'text', placeholder: 'Grade 4 Term 1 Fees', help: 'Leave empty to name it after the grade and term.' },
      { name: 'tuition', label: 'Tuition (KES)', type: 'money', required: true, help: 'Other items, such as activity or transport, are added once it is created.' },
      { name: 'notes', label: 'Notes', type: 'textarea' },
    ],
    run: (_ctx, _row, values) =>
      apiRequest<FeeStructure>('/fee-structures', {
        method: 'POST',
        body: {
          name: values.name || `${values.grade} Term ${values.term} Fees`,
          grade: values.grade,
          term: Number(values.term),
          year: values.year,
          notes: values.notes,
          items: [{ name: 'Tuition', amount_cents: values.tuition, item_type: 'tuition' }],
        },
      }),
    createdId: (created: FeeStructure) => created.id,
  },
  actions: [
    {
      id: 'bill',
      label: 'Bill this grade',
      tone: 'primary',
      when: (row) => row.active && (row.items?.length ?? 1) > 0,
      fields: [
        { name: 'due_date', label: 'Pay by', type: 'date', help: 'After this date an unpaid invoice shows as overdue.' },
        { name: 'stream', label: 'Only this stream', type: 'text', placeholder: 'North', help: 'Leave empty to bill the whole grade.' },
        { name: 'include_optional', label: 'Also bill the items marked "only when chosen"', type: 'checkbox' },
      ],
      preview: async (_ctx, row, values) => {
        const dry = await apiRequest<BulkResult>('/invoices/bulk', { method: 'POST', body: bulkBody(row, values, true) });
        if (dry.learners === 0) throw new Error(`There are no active learners in ${row.grade}${values.stream ? ` ${values.stream}` : ''}.`);
        if (dry.created === 0) throw new Error(`All ${dry.learners} learners already have an invoice for Term ${row.term} ${row.year}. Nothing to do.`);
        const skipped = dry.already_invoiced > 0 ? ` ${dry.already_invoiced} already have an invoice for this term and are left alone.` : '';
        return `This creates ${dry.created} invoice${dry.created === 1 ? '' : 's'} of ${kes(dry.each_cents)} each: ${kes(dry.total_cents)} in all.${skipped}`;
      },
      submitLabel: 'Create the invoices',
      run: (_ctx, row, values) => apiRequest<BulkResult>('/invoices/bulk', { method: 'POST', body: bulkBody(row, values, false) }),
    },
    {
      id: 'add-item',
      label: 'Add an item',
      fields: [
        { name: 'name', label: 'Item', type: 'text', required: true, placeholder: 'Activity fee' },
        { name: 'amount_cents', label: 'Amount (KES)', type: 'money', required: true },
        { name: 'item_type', label: 'Kind', type: 'select', required: true, options: ITEM_TYPES },
        { name: 'is_optional', label: 'Billed only when chosen (for example transport)', type: 'checkbox' },
      ],
      initial: () => ({ item_type: 'other' }),
      confirm: 'Invoices already created keep the items they were created with.',
      run: (_ctx, row, values) =>
        apiRequest<FeeStructure>(`/fee-structures/${row.id}/items`, {
          method: 'POST',
          body: { name: values.name, amount_cents: values.amount_cents, item_type: values.item_type, is_optional: Boolean(values.is_optional) },
        }),
    },
    {
      id: 'remove-item',
      label: 'Remove an item',
      when: (row) => (row.items?.length ?? 0) > 1,
      fields: (row) => [
        {
          name: 'item_id', label: 'Item to remove', type: 'select', required: true,
          options: (row.items ?? []).map((item) => ({ value: item.id, label: `${item.name} (${kes(item.amount_cents)})` })),
        },
      ],
      confirm: 'Invoices already created keep the items they were created with.',
      submitLabel: 'Remove',
      run: (_ctx, _row, values) => apiRequest<void>(`/fee-structures/items/${values.item_id}`, { method: 'DELETE' }),
    },
    {
      id: 'edit',
      label: 'Edit',
      fields: [
        { name: 'name', label: 'Name', type: 'text', required: true },
        { name: 'active', label: 'Active (learners can be billed from it)', type: 'checkbox' },
        { name: 'notes', label: 'Notes', type: 'textarea' },
      ],
      initial: (row) => ({ name: row.name, active: row.active, notes: row.notes ?? '' }),
      submitLabel: 'Save',
      run: (_ctx, row, values) =>
        apiRequest<FeeStructure>(`/fee-structures/${row.id}`, {
          method: 'PATCH',
          body: { name: values.name, active: Boolean(values.active), notes: values.notes ?? '' },
        }),
    },
    {
      id: 'delete',
      label: 'Delete',
      tone: 'danger',
      confirm: 'The fee structure is removed. Invoices already created from it are kept, with their items and payments.',
      submitLabel: 'Delete the fee structure',
      removes: true,
      run: (_ctx, row) => apiRequest<void>(`/fee-structures/${row.id}`, { method: 'DELETE' }),
    },
  ],
});

// --- Invoices ---------------------------------------------------------------

function InvoiceDetail({ row, ctx }: { row: Invoice; ctx: Ctx }) {
  const payments = useQuery({
    queryKey: ['finance', ctx.schoolId, 'invoice-payments', row.id, row.paid_cents, row.status],
    queryFn: () => apiRequest<Payment[] | null>(`/invoices/${row.id}/payments`),
    // An M-Pesa request appears, then settles, a few seconds after it is sent;
    // a refused or cancelled one changes nothing on the invoice itself, so the
    // list is re-read for as long as money is still owed.
    refetchInterval: row.status !== 'void' && row.balance_cents > 0 ? 3000 : false,
  });
  const discounts = useQuery({
    queryKey: ['finance', ctx.schoolId, 'invoice-discounts', row.id, row.discount_cents],
    queryFn: () => apiRequest<Discount[] | null>(`/invoices/${row.id}/discounts`),
  });

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <Tile label="Billed" value={kes(row.total_cents)} />
        <Tile label="Discount" value={kes(row.discount_cents)} />
        <Tile label="Paid" value={kes(row.paid_cents)} />
        <Tile label={row.credit_cents > 0 ? 'Overpaid' : 'Still owed'} value={kes(row.credit_cents > 0 ? row.credit_cents : row.balance_cents)} />
      </div>

      <Card title="Charged">
        <SimpleTable
          caption="Invoice items"
          rows={row.items ?? []}
          empty="This invoice has no items."
          columns={[
            { header: 'Item', cell: (item) => item.name },
            { header: 'Amount', align: 'right', cell: (item) => <Money cents={item.amount_cents} /> },
          ]}
        />
      </Card>

      {(discounts.data?.length ?? 0) > 0 && (
        <Card title="Discounts">
          <SimpleTable
            caption="Discounts on this invoice"
            rows={discounts.data ?? []}
            empty=""
            columns={[
              { header: 'Reason', cell: (discount) => discount.reason || '—' },
              { header: 'Kind', cell: (discount) => discount.discount_type },
              { header: 'Given', cell: (discount) => day(discount.created_at) },
              { header: 'Amount', align: 'right', cell: (discount) => <Money cents={discount.amount_cents} /> },
            ]}
          />
        </Card>
      )}

      <Card title="Payments">
        {payments.isLoading && <Spinner />}
        {payments.error && <div className="p-4"><ErrorNote error={payments.error} /></div>}
        {payments.data !== undefined && (
          <SimpleTable
            caption="Payments against this invoice"
            rows={payments.data ?? []}
            empty="Nothing has been paid against this invoice yet."
            columns={[
              { header: 'Receipt', cell: (payment) => workspaceLink(ctx, 'payments', payment.id, payment.receipt_number ?? 'Open') },
              { header: 'Paid by', cell: (payment) => CHANNEL[payment.channel] ?? payment.channel },
              { header: 'Status', cell: (payment) => <Status value={payment.status} /> },
              { header: 'When', cell: (payment) => <span className="whitespace-nowrap">{when(payment.paid_at ?? payment.created_at)}</span> },
              { header: 'Amount', align: 'right', cell: (payment) => <Money cents={payment.amount_cents} /> },
            ]}
          />
        )}
      </Card>

      <p className="text-sm">{workspaceLink(ctx, 'arrears', row.learner_id, `Everything billed to and paid for ${row.learner_name}`)}</p>
    </div>
  );
}

interface LearnerHit {
  id: string;
  full_name: string;
  grade: string;
  stream?: string;
  upi?: string;
}

const owed = (row: Invoice) => row.status !== 'void' && row.balance_cents > 0;

const invoices = resource<Invoice>({
  id: 'invoices',
  label: 'Invoices',
  noun: 'invoice',
  scopes: ['school'],
  roles: FINANCE_ROLES,
  purpose: 'What each learner has been billed for a term, and what is still owed on it.',
  list: async (_ctx, filters, cursor) => {
    const offset = Number(cursor ?? 0);
    const query = new URLSearchParams({ limit: String(PAGE), offset: String(offset) });
    for (const name of ['status', 'grade', 'search', 'term', 'year']) if (filters[name]) query.set(name, filters[name]);
    const items = (await apiRequest<Invoice[] | null>(`/invoices?${query}`)) ?? [];
    return { items, next: items.length === PAGE ? String(offset + PAGE) : undefined };
  },
  get: (_ctx, id) => apiRequest<Invoice>(`/invoices/${id}`),
  // While an M-Pesa request is out, the totals are about to change.
  live: (row) => row.status !== 'void' && row.status !== 'paid',
  rowId: (row) => row.id,
  title: (row) => `${row.learner_name} — ${row.invoice_number}`,
  status: (row) => row.status,
  filters: [
    { name: 'search', label: 'Search', type: 'text', placeholder: 'Learner, invoice number or UPI' },
    {
      name: 'status', label: 'Show', type: 'select',
      options: [
        { value: 'open', label: 'Still owed' },
        { value: 'overdue', label: 'Overdue' },
        { value: 'paid', label: 'Paid' },
        { value: 'void', label: 'Void' },
      ],
    },
    { name: 'grade', label: 'Grade', type: 'select', options: GRADE_OPTIONS },
    { name: 'term', label: 'Term', type: 'select', options: TERM_OPTIONS },
  ],
  columns: [
    { header: 'Learner', cell: (row) => row.learner_name },
    { header: 'Status', cell: (row) => <Status value={row.status} /> },
    { header: 'Owed', align: 'right', cell: (row) => <Money cents={row.balance_cents} /> },
  ],
  fields: [
    { label: 'Learner', value: (row) => `${row.learner_name}, ${[row.grade, row.stream].filter(Boolean).join(' ')}` },
    { label: 'Invoice number', value: (row) => row.invoice_number },
    { label: 'For', value: (row) => `Term ${row.term}, ${row.year}` },
    { label: 'Issued', value: (row) => day(row.issue_date) },
    { label: 'Pay by', value: (row) => day(row.due_date) },
    { label: 'Paybill account number', value: (row) => row.learner_upi || row.invoice_number },
    { label: 'Notes', value: (row) => row.notes || '—' },
    { label: 'Voided', value: (row) => (row.voided_at ? `${when(row.voided_at)}: ${row.void_reason ?? ''}` : '—') },
  ],
  extra: InvoiceDetail,
  create: {
    id: 'create',
    label: 'Bill one learner',
    fields: [
      {
        name: 'learner', label: 'Learner', type: 'picker', required: true,
        placeholder: 'Search by name or UPI',
        help: 'To bill a whole grade at once, open its fee structure and choose "Bill this grade".',
        search: async (_ctx, query) => {
          const found = await apiRequest<LearnerHit[] | null>(`/learners?search=${encodeURIComponent(query)}&limit=10`);
          return (found ?? []).map((learner) => ({
            value: learner.id,
            label: learner.full_name,
            hint: [learner.grade, learner.stream].filter(Boolean).join(' '),
          }));
        },
      },
      {
        name: 'fee_structure_id', label: 'Fee structure', type: 'select', required: true,
        options: async () =>
          ((await apiRequest<FeeStructure[] | null>('/fee-structures')) ?? [])
            .filter((structure) => structure.active)
            .map((structure) => ({ value: structure.id, label: `${structure.name} (${kes(structure.total_cents)})` })),
        help: 'It must be for the learner’s grade.',
      },
      { name: 'due_date', label: 'Pay by', type: 'date' },
      { name: 'include_optional', label: 'Also bill the items marked "only when chosen"', type: 'checkbox' },
      { name: 'notes', label: 'Notes', type: 'textarea' },
    ],
    run: async (_ctx, _row, values) => {
      const chosen: string[] = values.learner ?? [];
      if (chosen.length !== 1) throw new Error('Choose one learner. To bill many at once, use "Bill this grade" on the fee structure.');
      return apiRequest<Invoice>('/invoices', {
        method: 'POST',
        body: {
          learner_id: chosen[0],
          fee_structure_id: values.fee_structure_id,
          due_date: values.due_date,
          include_optional: Boolean(values.include_optional),
          notes: values.notes,
        },
      });
    },
    createdId: (invoice: Invoice) => invoice.id,
  },
  actions: [
    {
      id: 'mpesa',
      label: 'Request M-Pesa payment',
      tone: 'primary',
      when: owed,
      fields: [
        { name: 'phone', label: 'Parent’s Safaricom number', type: 'text', required: true, placeholder: '0712 345 678' },
        { name: 'amount_cents', label: 'Amount (KES)', type: 'money', required: true, help: 'Whole shillings. Up to what is still owed.' },
        { name: 'paid_by', label: 'Parent’s name', type: 'text' },
      ],
      initial: (row) => ({ amount_cents: String(Math.floor(row.balance_cents / 100)) }),
      confirm: 'The parent’s phone asks for their M-Pesa PIN. The invoice changes only when Safaricom confirms the payment.',
      submitLabel: 'Send the request',
      run: (_ctx, row, values, key) =>
        apiRequest<Payment>('/payments/mpesa/stk', {
          method: 'POST',
          body: { invoice_id: row.id, phone: values.phone, amount_cents: values.amount_cents, paid_by: values.paid_by, idempotency_key: key },
        }),
    },
    {
      id: 'pay',
      label: 'Record a payment',
      when: owed,
      fields: [
        {
          name: 'channel', label: 'Paid by', type: 'select', required: true,
          options: [
            { value: 'cash', label: 'Cash' },
            { value: 'bank', label: 'Bank deposit or transfer' },
            { value: 'cheque', label: 'Cheque' },
            { value: 'mpesa', label: 'M-Pesa (from the confirmation message)' },
          ],
        },
        { name: 'amount_cents', label: 'Amount received (KES)', type: 'money', required: true },
        {
          name: 'reference', label: 'Reference', type: 'text', required: true,
          showIf: (values) => values.channel !== 'cash',
          help: 'The bank slip number, cheque number or M-Pesa confirmation code. Each can be recorded once.',
        },
        { name: 'paid_by', label: 'Paid by (name)', type: 'text' },
        { name: 'paid_at', label: 'Received on', type: 'datetime', help: 'Leave empty for now.' },
        { name: 'notes', label: 'Notes', type: 'textarea' },
      ],
      initial: () => ({ channel: 'cash' }),
      preview: async (_ctx, row, values) => {
        const amount = Number(values.amount_cents);
        const left = row.balance_cents - amount;
        return `${kes(amount)} is recorded against ${row.learner_name}'s invoice and a receipt number is issued. ${left > 0 ? `${kes(left)} will still be owed.` : 'The invoice will be paid in full.'}`;
      },
      submitLabel: 'Record the payment',
      run: (_ctx, row, values, key) =>
        apiRequest<Payment>('/payments', {
          method: 'POST',
          body: {
            invoice_id: row.id, channel: values.channel, amount_cents: values.amount_cents, reference: values.reference,
            paid_by: values.paid_by, paid_at: values.paid_at, notes: values.notes, idempotency_key: key,
          },
        }),
    },
    {
      id: 'discount',
      label: 'Give a discount',
      roles: ['super_admin', 'principal'],
      when: owed,
      fields: [
        { name: 'amount_cents', label: 'Amount to take off (KES)', type: 'money', required: true },
        {
          name: 'discount_type', label: 'Kind', type: 'select', required: true,
          options: [
            { value: 'sibling', label: 'Sibling discount' },
            { value: 'scholarship', label: 'Scholarship or bursary' },
            { value: 'waiver', label: 'Waiver' },
            { value: 'other', label: 'Other' },
          ],
        },
        { name: 'reason', label: 'Reason', type: 'textarea', required: true, help: 'Recorded with your name as the approver.' },
      ],
      submitLabel: 'Give the discount',
      run: (_ctx, row, values) =>
        apiRequest<Discount>(`/invoices/${row.id}/discounts`, {
          method: 'POST',
          body: { amount_cents: values.amount_cents, discount_type: values.discount_type, reason: values.reason },
        }),
    },
    {
      id: 'edit',
      label: 'Change the due date',
      when: (row) => row.status !== 'void',
      fields: [
        { name: 'due_date', label: 'Pay by', type: 'date', required: true },
        { name: 'notes', label: 'Notes', type: 'textarea' },
      ],
      initial: (row) => ({ due_date: row.due_date ?? '', notes: row.notes ?? '' }),
      submitLabel: 'Save',
      run: (_ctx, row, values) =>
        apiRequest<Invoice>(`/invoices/${row.id}`, { method: 'PATCH', body: { due_date: values.due_date, notes: values.notes ?? '' } }),
    },
    {
      id: 'void',
      label: 'Void this invoice',
      tone: 'danger',
      when: (row) => row.status !== 'void' && row.paid_cents === 0,
      fields: [{ name: 'reason', label: 'Why is it being voided?', type: 'textarea', required: true }],
      confirm: 'The invoice stays on record as void and nothing is owed on it. The learner can then be billed again for the term. This cannot be undone.',
      submitLabel: 'Void the invoice',
      run: (_ctx, row, values) => apiRequest<Invoice>(`/invoices/${row.id}/void`, { method: 'POST', body: { reason: values.reason } }),
    },
  ],
});

// --- Payments ---------------------------------------------------------------

function PaymentDetail({ row, ctx }: { row: Payment; ctx: Ctx }) {
  return (
    <div className="space-y-3">
      {row.status === 'pending' && (
        <p role="status" className="rounded-lg border border-sky-300 bg-sky-50 p-3 text-sm text-sky-950">
          Waiting for the parent to enter their M-Pesa PIN. This page updates when Safaricom reports the result; nothing is counted as paid until then.
        </p>
      )}
      {row.status === 'failed' && (
        <p role="alert" className="rounded-lg border border-red-300 bg-red-50 p-3 text-sm text-red-900">
          {FAILURE[row.failure_code ?? ''] ?? row.mpesa_result_desc ?? 'The payment did not go through.'}
        </p>
      )}
      {row.status === 'reversed' && (
        <p role="status" className="rounded-lg border border-gray-300 bg-gray-50 p-3 text-sm">
          Reversed {when(row.reversed_at)}: {row.reversal_reason}
        </p>
      )}
      {row.status === 'completed' && (
        <div className="flex flex-wrap gap-2 print:hidden">
          <Button onClick={() => window.print()}>Print this receipt</Button>
        </div>
      )}
      <p className="text-sm">{workspaceLink(ctx, 'invoices', row.invoice_id, `Open invoice ${row.invoice_number}`)}</p>
    </div>
  );
}

const payments = resource<Payment>({
  id: 'payments',
  label: 'Payments',
  noun: 'payment',
  scopes: ['school'],
  roles: FINANCE_ROLES,
  purpose: 'Every payment received, with its receipt number. A payment is recorded from its invoice.',
  list: async (_ctx, filters, cursor) => {
    const offset = Number(cursor ?? 0);
    const query = new URLSearchParams({ limit: String(PAGE), offset: String(offset) });
    for (const name of ['status', 'channel', 'search']) if (filters[name]) query.set(name, filters[name]);
    const items = (await apiRequest<Payment[] | null>(`/payments?${query}`)) ?? [];
    return { items, next: items.length === PAGE ? String(offset + PAGE) : undefined };
  },
  get: (_ctx, id) => apiRequest<Payment>(`/payments/${id}`),
  live: (row) => row.status === 'pending',
  rowId: (row) => row.id,
  title: (row) => (row.receipt_number ? `Receipt ${row.receipt_number}` : `${CHANNEL[row.channel] ?? row.channel} request`),
  status: (row) => row.status,
  filters: [
    { name: 'search', label: 'Search', type: 'text', placeholder: 'Learner, receipt, reference or M-Pesa code' },
    {
      name: 'status', label: 'Status', type: 'select',
      options: [
        { value: 'completed', label: 'Received' },
        { value: 'pending', label: 'Waiting for the parent' },
        { value: 'failed', label: 'Did not go through' },
        { value: 'reversed', label: 'Reversed' },
      ],
    },
    { name: 'channel', label: 'Paid by', type: 'select', options: Object.entries(CHANNEL).map(([value, label]) => ({ value, label })) },
  ],
  columns: [
    { header: 'Learner', cell: (row) => row.learner_name },
    { header: 'Status', cell: (row) => <Status value={row.status} /> },
    { header: 'Amount', align: 'right', cell: (row) => <Money cents={row.amount_cents} /> },
  ],
  fields: [
    { label: 'Amount', value: (row) => kes(row.amount_cents) },
    { label: 'For', value: (row) => `${row.learner_name}, ${row.grade}` },
    { label: 'Paid by', value: (row) => [CHANNEL[row.channel] ?? row.channel, row.paid_by].filter(Boolean).join(' — ') },
    { label: 'Received', value: (row) => when(row.paid_at) },
    { label: 'Receipt number', value: (row) => row.receipt_number || '—' },
    { label: 'Reference', value: (row) => row.mpesa_receipt || row.reference || '—' },
    { label: 'Phone', value: (row) => (row.phone ? formatPhone(row.phone) : '—') },
    { label: 'Notes', value: (row) => row.notes || '—' },
  ],
  extra: PaymentDetail,
  actions: [
    {
      id: 'reverse',
      label: 'Reverse this payment',
      tone: 'danger',
      roles: ['super_admin', 'principal', 'bursar'],
      when: (row) => row.status === 'completed',
      fields: [{ name: 'reason', label: 'Why is it being reversed?', type: 'textarea', required: true, placeholder: 'Cheque bounced; recorded against the wrong learner…' }],
      confirm: 'The payment stays on record as reversed, with your name and reason, and the invoice is owed again. This does not send any money back. It cannot be undone.',
      submitLabel: 'Reverse the payment',
      run: (_ctx, row, values) => apiRequest<Payment>(`/payments/${row.id}/reverse`, { method: 'POST', body: { reason: values.reason } }),
    },
  ],
});

// --- Paybill payments -------------------------------------------------------

function Allocations({ row, ctx }: { row: InboxPayment; ctx: Ctx }) {
  return (
    <div className="space-y-3">
      {row.remaining_cents > 0 && (
        <p role="status" className="rounded-lg border border-amber-300 bg-amber-50 p-3 text-sm text-amber-950">
          <strong>{kes(row.remaining_cents)}</strong> of this payment is not against any invoice yet.
          {row.learner_name ? ` ${row.learner_name} has nothing more owing; put it against their next invoice when it is created.` : ' The account number typed did not match a learner.'}
        </p>
      )}
      <Card title="Where it went">
        <SimpleTable
          caption="Invoices this payment was put against"
          rows={row.allocations ?? []}
          empty="Not against any invoice yet."
          columns={[
            { header: 'Receipt', cell: (payment) => workspaceLink(ctx, 'payments', payment.id, payment.receipt_number ?? 'Open') },
            { header: 'Learner', cell: (payment) => payment.learner_name },
            { header: 'Invoice', cell: (payment) => workspaceLink(ctx, 'invoices', payment.invoice_id, payment.invoice_number) },
            { header: 'Status', cell: (payment) => <Status value={payment.status} /> },
            { header: 'Amount', align: 'right', cell: (payment) => <Money cents={payment.amount_cents} /> },
          ]}
        />
      </Card>
    </div>
  );
}

const paybill = resource<InboxPayment>({
  id: 'paybill',
  label: 'Paybill payments',
  noun: 'paybill payment',
  scopes: ['school'],
  roles: FINANCE_ROLES,
  purpose: 'Money parents paid straight to the school’s paybill. It is matched to a learner by the account number they typed; what could not be matched waits here.',
  defaultFilters: () => ({ status: 'open' }),
  list: async (_ctx, filters, cursor) => {
    const offset = Number(cursor ?? 0);
    const query = new URLSearchParams({ limit: String(PAGE), offset: String(offset) });
    for (const name of ['status', 'search']) if (filters[name]) query.set(name, filters[name]);
    const items = (await apiRequest<InboxPayment[] | null>(`/finance/paybill?${query}`)) ?? [];
    return { items, next: items.length === PAGE ? String(offset + PAGE) : undefined };
  },
  get: async (_ctx, id) => {
    const result = await apiRequest<{ payment: InboxPayment; allocations: Payment[] | null }>(`/finance/paybill/${id}`);
    return { ...result.payment, allocations: result.allocations ?? [] };
  },
  rowId: (row) => row.id,
  title: (row) => `${kes(row.amount_cents)} — ${row.trans_id}`,
  status: (row) => row.status,
  filters: [
    {
      name: 'status', label: 'Show', type: 'select',
      options: [
        { value: 'open', label: 'With money still to allocate' },
        { value: 'allocated', label: 'Fully allocated' },
      ],
      help: '"Any" shows every paybill payment.',
    },
    { name: 'search', label: 'Search', type: 'text', placeholder: 'M-Pesa code, account number or name' },
  ],
  columns: [
    { header: 'Account typed', cell: (row) => row.bill_ref || '—' },
    { header: 'Status', cell: (row) => <Status value={row.status} /> },
    { header: 'To allocate', align: 'right', cell: (row) => <Money cents={row.remaining_cents} /> },
  ],
  fields: [
    { label: 'M-Pesa code', value: (row) => row.trans_id },
    { label: 'Received', value: (row) => when(row.trans_time) },
    { label: 'Amount', value: (row) => kes(row.amount_cents) },
    { label: 'Account number typed', value: (row) => row.bill_ref || '—' },
    { label: 'Paid by', value: (row) => row.payer_name || '—' },
    { label: 'Matched to', value: (row) => row.learner_name || 'No learner yet' },
  ],
  extra: Allocations,
  actions: [
    {
      id: 'allocate',
      label: 'Put against an invoice',
      tone: 'primary',
      when: (row) => row.remaining_cents > 0,
      fields: [
        {
          name: 'invoice', label: 'Invoice', type: 'picker', required: true,
          placeholder: 'Search by learner or invoice number',
          help: 'Only invoices with money still owed are listed.',
          search: async (_ctx, query) => {
            const found = await apiRequest<Invoice[] | null>(`/invoices?status=open&limit=10&search=${encodeURIComponent(query)}`);
            return (found ?? []).map((invoice) => ({
              value: invoice.id,
              label: `${invoice.learner_name} — ${invoice.invoice_number}`,
              hint: `${kes(invoice.balance_cents)} owed, Term ${invoice.term} ${invoice.year}`,
            }));
          },
        },
        { name: 'amount_cents', label: 'Amount (KES)', type: 'money', help: 'Leave empty to put as much as fits against the invoice.' },
      ],
      submitLabel: 'Allocate',
      run: async (_ctx, row, values) => {
        const chosen: string[] = values.invoice ?? [];
        if (chosen.length !== 1) throw new Error('Choose one invoice at a time.');
        return apiRequest<InboxPayment>(`/finance/paybill/${row.id}/allocate`, {
          method: 'POST',
          body: { invoice_id: chosen[0], amount_cents: values.amount_cents ?? 0 },
        });
      },
    },
  ],
});

// --- Arrears and statements -------------------------------------------------

function StatementPanel({ row, ctx }: { row: ArrearsRow; ctx: Ctx }) {
  const statement = useQuery({
    queryKey: ['finance', ctx.schoolId, 'statement', row.learner_id],
    queryFn: () => apiRequest<Statement>(`/finance/learners/${row.learner_id}/statement`),
  });
  if (statement.isLoading) return <Spinner />;
  if (statement.error) return <ErrorNote error={statement.error} />;
  const data = statement.data;
  if (!data) return null;

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-3 gap-2">
        <Tile label="Billed" value={kes(data.billed_cents)} note="After discounts" />
        <Tile label="Paid" value={kes(data.paid_cents)} />
        <Tile label={data.balance_cents < 0 ? 'Overpaid' : 'Balance'} value={kes(Math.abs(data.balance_cents))} />
      </div>
      <div className="print:hidden"><Button onClick={() => window.print()}>Print this statement</Button></div>
      <Card title="Invoices">
        <SimpleTable
          caption="Invoices for this learner"
          rows={data.invoices}
          empty="This learner has not been billed."
          columns={[
            { header: 'Invoice', cell: (invoice) => workspaceLink(ctx, 'invoices', invoice.id, invoice.invoice_number) },
            { header: 'For', cell: (invoice) => <span className="whitespace-nowrap">Term {invoice.term} {invoice.year}</span> },
            { header: 'Status', cell: (invoice) => <Status value={invoice.status} /> },
            { header: 'Billed', align: 'right', cell: (invoice) => <Money cents={invoice.total_cents - invoice.discount_cents} /> },
            { header: 'Owed', align: 'right', cell: (invoice) => <Money cents={invoice.balance_cents} /> },
          ]}
        />
      </Card>
      <Card title="Payments">
        <SimpleTable
          caption="Payments for this learner"
          rows={data.payments}
          empty="No payments yet."
          columns={[
            { header: 'Receipt', cell: (payment) => workspaceLink(ctx, 'payments', payment.id, payment.receipt_number ?? 'Open') },
            { header: 'When', cell: (payment) => <span className="whitespace-nowrap">{when(payment.paid_at)}</span> },
            { header: 'Paid by', cell: (payment) => CHANNEL[payment.channel] ?? payment.channel },
            { header: 'Status', cell: (payment) => <Status value={payment.status} /> },
            { header: 'Amount', align: 'right', cell: (payment) => <Money cents={payment.amount_cents} /> },
          ]}
        />
      </Card>
    </div>
  );
}

const arrears = resource<ArrearsRow>({
  id: 'arrears',
  label: 'Balances',
  noun: 'learner',
  scopes: ['school'],
  roles: FINANCE_ROLES,
  purpose: 'Learners who still owe fees, largest balance first, each with their full statement. To remind their parents, send an SMS to "Parents with a fee balance".',
  list: async (_ctx, filters, cursor) => {
    const offset = Number(cursor ?? 0);
    const query = new URLSearchParams({ limit: String(PAGE), offset: String(offset) });
    for (const name of ['grade', 'search']) if (filters[name]) query.set(name, filters[name]);
    const items = (await apiRequest<ArrearsRow[] | null>(`/finance/arrears?${query}`)) ?? [];
    return { items, next: items.length === PAGE ? String(offset + PAGE) : undefined };
  },
  // A statement can be opened for any learner, including one who owes nothing
  // (from an invoice or a payment), so the record is built from the statement.
  get: async (_ctx, id) => {
    const statement = await apiRequest<Statement>(`/finance/learners/${id}/statement`);
    return {
      learner_id: id,
      learner_name: statement.learner_name,
      grade: statement.grade,
      stream: statement.stream,
      invoices: statement.invoices.filter((invoice) => invoice.balance_cents > 0).length,
      balance_cents: Math.max(statement.balance_cents, 0),
    };
  },
  rowId: (row) => row.learner_id,
  title: (row) => row.learner_name,
  filters: [
    { name: 'search', label: 'Search', type: 'text', placeholder: 'Learner name' },
    { name: 'grade', label: 'Grade', type: 'select', options: GRADE_OPTIONS },
  ],
  columns: [
    { header: 'Learner', cell: (row) => row.learner_name },
    { header: 'Class', cell: (row) => <span className="whitespace-nowrap">{[row.grade, row.stream].filter(Boolean).join(' ')}</span> },
    { header: 'Owes', align: 'right', cell: (row) => <Money cents={row.balance_cents} /> },
  ],
  fields: [
    { label: 'Class', value: (row) => [row.grade, row.stream].filter(Boolean).join(' ') },
    { label: 'Invoices with money owed', value: (row) => row.invoices },
    { label: 'Oldest due date', value: (row) => day(row.oldest_due) },
  ],
  extra: StatementPanel,
});

export const finance: ModuleManifest = {
  id: 'finance',
  label: 'Finance',
  sections: [overview, invoices, payments, paybill, arrears, feeStructures],
};

