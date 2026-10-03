'use client';

import { useState, type ButtonHTMLAttributes, type ReactNode } from 'react';
import { APIError } from '@/lib/api';

type Tone = 'good' | 'progress' | 'warning' | 'critical' | 'neutral';

const STATUS_TONE: Record<string, Tone> = {
  delivered: 'good', read: 'good', active: 'good', paid: 'good', approved: 'good', completed: 'good', allocated: 'good',
  sent: 'progress', sending: 'progress', pending: 'progress', scheduled: 'progress',
  draft: 'neutral', cancelled: 'neutral', archived: 'neutral', deactivated: 'neutral',
  'opted out': 'warning', partial: 'warning', unpaid: 'warning', partially_paid: 'warning', unmatched: 'warning', part_allocated: 'warning',
  void: 'neutral', reversed: 'neutral', 'switched off': 'neutral',
  failed: 'critical', rejected: 'critical', overdue: 'critical',
};

const TONE_CLASS: Record<Tone, string> = {
  good: 'bg-emerald-50 text-emerald-800 ring-emerald-600/25',
  progress: 'bg-sky-50 text-sky-800 ring-sky-600/25',
  warning: 'bg-amber-50 text-amber-900 ring-amber-600/30',
  critical: 'bg-red-50 text-red-800 ring-red-600/25',
  neutral: 'bg-slate-100 text-slate-700 ring-slate-500/25',
};

const TONE_DOT: Record<Tone, string> = {
  good: 'bg-emerald-600', progress: 'bg-sky-600', warning: 'bg-amber-600', critical: 'bg-red-600', neutral: 'bg-slate-500',
};

export function humanise(value: string): string {
  const text = value.replace(/_/g, ' ').toLowerCase();
  return text.charAt(0).toUpperCase() + text.slice(1);
}

/** A state, always spelled out in words, so its meaning never depends on the colour beside it. */
export function Status({ value }: { value: string | undefined | null }) {
  if (!value) return <span className="text-gray-500">—</span>;
  const tone = STATUS_TONE[value.toLowerCase()] ?? 'neutral';
  return (
    <span className={`inline-flex w-fit items-center gap-1.5 whitespace-nowrap rounded-full px-2 py-0.5 text-xs font-semibold ring-1 ring-inset ${TONE_CLASS[tone]}`}>
      <span aria-hidden="true" className={`h-1.5 w-1.5 rounded-full ${TONE_DOT[tone]}`} />
      {humanise(value)}
    </span>
  );
}

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  tone?: 'primary' | 'danger' | 'plain';
  small?: boolean;
}

export function Button({ tone = 'plain', small, className = '', type = 'button', ...rest }: ButtonProps) {
  const tones = {
    primary: 'bg-blue-700 text-white hover:bg-blue-800 border-transparent',
    danger: 'bg-white text-red-800 border-red-300 hover:bg-red-50',
    plain: 'bg-white text-gray-900 border-gray-300 hover:bg-gray-50',
  };
  const size = small ? 'px-2.5 py-1 text-xs' : 'px-3.5 py-2 text-sm';
  return (
    <button
      type={type}
      className={`inline-flex items-center justify-center gap-1.5 rounded-lg border font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${size} ${tones[tone]} ${className}`}
      {...rest}
    />
  );
}

export function Spinner({ label = 'Loading' }: { label?: string }) {
  return (
    <p role="status" className="flex items-center gap-2 p-6 text-sm text-gray-600">
      <span className="h-4 w-4 animate-spin rounded-full border-2 border-gray-200 border-t-blue-700" aria-hidden="true" />
      {label}…
    </p>
  );
}

/** A refusal in the API's own words, with its stable code. */
export function ErrorNote({ error }: { error: unknown }) {
  const api = error instanceof APIError ? error : undefined;
  const text = api?.status === 404
    ? 'This record does not exist, or it is outside the school chosen above.'
    : error instanceof Error ? error.message : 'Something went wrong.';
  return (
    <div role="alert" className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-900">
      <p className="font-semibold">{text}</p>
      {api && api.code !== 'UNKNOWN' && <p className="mt-1 font-mono text-xs text-red-800">{api.code}</p>}
    </div>
  );
}

export function Empty({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-2 px-6 py-12 text-center">
      <p className="font-semibold">{title}</p>
      {children && <div className="max-w-md text-sm text-gray-600">{children}</div>}
    </div>
  );
}

export function Card({ title, actions, children, className = '' }: { title?: string; actions?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <section className={`rounded-xl border border-gray-200 bg-white ${className}`}>
      {(title || actions) && (
        <header className="flex flex-wrap items-center justify-between gap-2 border-b border-gray-200 px-4 py-3">
          <h2 className="font-bold">{title}</h2>
          {actions}
        </header>
      )}
      {children}
    </section>
  );
}

/** A headline figure. */
export function Tile({ label, value, note }: { label: string; value: ReactNode; note?: ReactNode }) {
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-3">
      <p className="text-xs font-semibold uppercase tracking-wide text-gray-600">{label}</p>
      <p className="mt-1 text-2xl font-bold tabular-nums">{value}</p>
      {note && <p className="mt-1 text-xs text-gray-600">{note}</p>}
    </div>
  );
}

export function Definitions({ items }: { items: { label: string; value: ReactNode }[] }) {
  return (
    <dl className="grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2">
      {items.map((item) => (
        <div key={item.label} className="min-w-0">
          <dt className="text-xs font-semibold uppercase tracking-wide text-gray-600">{item.label}</dt>
          <dd className="mt-0.5 whitespace-pre-wrap break-words text-sm">{item.value ?? '—'}</dd>
        </div>
      ))}
    </dl>
  );
}

export function SimpleTable<T>({ rows, columns, empty, caption }: {
  rows: T[];
  columns: { header: string; cell: (row: T) => ReactNode; align?: 'right' }[];
  empty: string;
  caption: string;
}) {
  if (rows.length === 0) return <p className="px-4 py-6 text-sm text-gray-600">{empty}</p>;
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left text-sm">
        <caption className="sr-only">{caption}</caption>
        <thead>
          <tr className="border-b border-gray-200 text-xs uppercase tracking-wide text-gray-600">
            {columns.map((column) => (
              <th key={column.header} scope="col" className={`px-4 py-2 font-semibold ${column.align === 'right' ? 'text-right' : ''}`}>
                {column.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, index) => (
            <tr key={index} className="border-b border-gray-200 last:border-0">
              {columns.map((column) => (
                <td key={column.header} className={`px-4 py-2 align-top ${column.align === 'right' ? 'text-right' : ''}`}>
                  {column.cell(row)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

const NAIROBI = new Intl.DateTimeFormat('en-KE', {
  timeZone: 'Africa/Nairobi', day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit', hour12: false,
});

/** A moment, in Nairobi time, whatever the device is set to. */
export function when(iso: string | undefined | null): string {
  if (!iso) return '—';
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? '—' : NAIROBI.format(date);
}

/** A secret shown once. It is never stored by the interface and cannot be shown again. */
export function RevealBox({ title, note, items, onDone }: {
  title: string;
  note: string;
  items: { label: string; value: string }[];
  onDone: () => void;
}) {
  const [copied, setCopied] = useState('');
  return (
    <div role="status" className="space-y-3 rounded-xl border border-amber-300 bg-amber-50 p-4">
      <p className="font-bold">{title}</p>
      <p className="text-sm text-amber-950">{note}</p>
      {items.map((item) => (
        <div key={item.label}>
          <p className="text-xs font-semibold uppercase tracking-wide text-amber-900">{item.label}</p>
          <div className="mt-1 flex items-start gap-2">
            <code className="min-w-0 flex-1 break-all rounded-lg bg-white p-2 font-mono text-sm">{item.value}</code>
            <Button
              small
              onClick={() => {
                void navigator.clipboard?.writeText(item.value);
                setCopied(item.label);
              }}
            >
              {copied === item.label ? 'Copied' : 'Copy'}
            </Button>
          </div>
        </div>
      ))}
      <Button tone="primary" onClick={onDone}>I have saved it</Button>
    </div>
  );
}
