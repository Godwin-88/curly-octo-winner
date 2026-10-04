'use client';

import { useQuery } from '@tanstack/react-query';
import { useEffect, useId, useRef, useState, type FormEvent } from 'react';
import type { Ctx, FormField, FormValues, Option } from '@/framework/types';
import { unsaved } from '@/shell/context';
import { Button, ErrorNote } from './kit';

const INPUT = 'w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-600';

export function newIdempotencyKey(): string {
  return typeof crypto !== 'undefined' && 'randomUUID' in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random()}`;
}

function useOptions(field: FormField, ctx: Ctx): { options: Option[]; loading: boolean } {
  const dynamic = typeof field.options === 'function';
  const query = useQuery({
    queryKey: ['options', field.name, ctx.schoolId, ctx.groupId],
    queryFn: () => (field.options as (ctx: Ctx) => Promise<Option[]>)(ctx),
    enabled: dynamic,
  });
  if (!dynamic) return { options: (field.options as Option[] | undefined) ?? [], loading: false };
  return { options: query.data ?? [], loading: query.isLoading };
}

function Control({ field, ctx, value, onChange, describedBy }: {
  field: FormField;
  ctx: Ctx;
  value: unknown;
  onChange: (value: unknown) => void;
  describedBy?: string;
}) {
  const id = useId();
  const { options, loading } = useOptions(field, ctx);
  const common = { id, 'aria-describedby': describedBy, required: field.required, placeholder: field.placeholder };
  const text = (value as string | undefined) ?? '';
  let control;
  switch (field.type) {
    case 'select':
      control = (
        <select {...common} className={INPUT} value={text} onChange={(event) => onChange(event.target.value)}>
          <option value="">{loading ? 'Loading…' : field.required ? 'Choose…' : 'Any'}</option>
          {options.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </select>
      );
      break;
    case 'textarea':
      control = (
        <>
          <textarea {...common} rows={5} className={INPUT} value={text} onChange={(event) => onChange(event.target.value)} />
          {field.inserts && (
            <div className="mt-1.5 flex flex-wrap items-center gap-1.5" role="group" aria-label={`Insert into ${field.label}`}>
              <span className="text-xs text-gray-600">Insert:</span>
              {field.inserts.map((insert) => (
                <button
                  key={insert.value}
                  type="button"
                  className="rounded-full border border-gray-300 bg-white px-2 py-0.5 text-xs font-medium hover:bg-gray-50"
                  onClick={() => onChange(`${text}${text && !text.endsWith(' ') ? ' ' : ''}${insert.value}`)}
                >
                  {insert.label}
                </button>
              ))}
            </div>
          )}
        </>
      );
      break;
    case 'picker':
      control = <Picker id={id} field={field} ctx={ctx} chosen={(value as Option[] | undefined) ?? []} onChange={onChange} describedBy={describedBy} />;
      break;
    case 'checkbox':
      control = (
        <input id={id} type="checkbox" aria-describedby={describedBy} className="h-4 w-4" checked={Boolean(value)} onChange={(event) => onChange(event.target.checked)} />
      );
      break;
    default:
      control = (
        <input
          {...common}
          className={`${INPUT} ${field.type === 'number' || field.type === 'money' ? 'tabular-nums' : ''}`}
          type={field.type === 'datetime' ? 'datetime-local' : field.type === 'date' ? 'date' : 'text'}
          inputMode={field.type === 'number' ? 'numeric' : field.type === 'money' ? 'decimal' : undefined}
          value={text}
          onChange={(event) => onChange(event.target.value)}
        />
      );
  }
  return (
    <div className={field.type === 'checkbox' ? 'flex items-center gap-2' : 'space-y-1'}>
      {field.type === 'checkbox' && control}
      <label htmlFor={id} className="block text-sm font-semibold">
        {field.label}
        {field.required && <span className="text-red-700" aria-hidden="true"> *</span>}
      </label>
      {field.type !== 'checkbox' && control}
    </div>
  );
}

/**
 * Chooses several records by searching for them. What is chosen stays visible
 * as removable chips; the search list only ever shows a page of matches, so it
 * works the same for 20 parents or 2,000.
 */
function Picker({ id, field, ctx, chosen, onChange, describedBy }: {
  id: string;
  field: FormField;
  ctx: Ctx;
  chosen: Option[];
  onChange: (value: Option[]) => void;
  describedBy?: string;
}) {
  const [query, setQuery] = useState('');
  const [debounced, setDebounced] = useState('');
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), 250);
    return () => clearTimeout(timer);
  }, [query]);

  const results = useQuery({
    queryKey: ['picker', field.name, ctx.schoolId, debounced],
    queryFn: () => field.search!(ctx, debounced),
    enabled: Boolean(field.search),
  });
  const chosenIds = new Set(chosen.map((option) => option.value));
  const matches = (results.data ?? []).filter((option) => !chosenIds.has(option.value)).slice(0, 8);

  return (
    <div className="space-y-2">
      {chosen.length > 0 && (
        <ul className="flex flex-wrap gap-1.5" aria-label={`Chosen: ${field.label}`}>
          {chosen.map((option) => (
            <li key={option.value} className="inline-flex items-center gap-1 rounded-full bg-blue-50 py-0.5 pl-2.5 pr-1 text-sm text-blue-950 ring-1 ring-inset ring-blue-200">
              {option.label}
              <button
                type="button"
                aria-label={`Remove ${option.label}`}
                className="rounded-full px-1.5 text-blue-900 hover:bg-blue-100"
                onClick={() => onChange(chosen.filter((item) => item.value !== option.value))}
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}
      <input
        id={id}
        type="search"
        aria-describedby={describedBy}
        placeholder={field.placeholder ?? 'Search…'}
        className={INPUT}
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        onKeyDown={(event) => {
          // Enter adds the first match instead of submitting the form.
          if (event.key === 'Enter') {
            event.preventDefault();
            if (matches[0]) {
              onChange([...chosen, matches[0]]);
              setQuery('');
            }
          }
        }}
      />
      <ul className="divide-y divide-gray-200 rounded-lg border border-gray-200" aria-label={`Matches for ${field.label}`}>
        {results.isLoading && <li className="px-3 py-2 text-sm text-gray-600">Searching…</li>}
        {results.error && <li className="px-3 py-2 text-sm text-red-800">Could not search. Try again.</li>}
        {results.isSuccess && matches.length === 0 && (
          <li className="px-3 py-2 text-sm text-gray-600">{debounced ? 'Nobody matches.' : 'Everyone listed is already chosen.'}</li>
        )}
        {matches.map((option) => (
          <li key={option.value}>
            <button
              type="button"
              className="flex w-full items-baseline justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-blue-50"
              onClick={() => {
                onChange([...chosen, option]);
                setQuery('');
              }}
            >
              <span className="font-semibold">{option.label}</span>
              {option.hint && <span className="shrink-0 text-xs text-gray-600">{option.hint}</span>}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

/** Turns what was typed into what the API takes. Empty values are left out. */
export function toPayload(fields: FormField[], values: FormValues): { payload: FormValues; problems: Record<string, string> } {
  const payload: FormValues = {};
  const problems: Record<string, string> = {};
  for (const field of fields) {
    if (field.showIf && !field.showIf(values)) continue;
    const raw = values[field.name];
    const blank = raw === undefined || raw === null || raw === '' || (Array.isArray(raw) && raw.length === 0);
    if (blank) {
      if (field.required) problems[field.name] = 'This is needed.';
      continue;
    }
    switch (field.type) {
      case 'number':
        if (/^\d+$/.test(String(raw).trim())) payload[field.name] = Number(raw);
        else problems[field.name] = 'Enter a whole number.';
        break;
      case 'money': {
        // Shillings as typed, with or without thousands commas; cents out, so
        // no amount ever passes through a fraction.
        const typed = String(raw).trim().replace(/^kes\s*/i, '').replace(/,/g, '');
        const match = /^(\d{1,10})(?:\.(\d{1,2}))?$/.exec(typed);
        const cents = match ? Number(match[1]) * 100 + Number((match[2] ?? '').padEnd(2, '0')) : 0;
        if (!match) problems[field.name] = 'Enter an amount in shillings, such as 17500.';
        else if (cents <= 0) problems[field.name] = 'Enter an amount above zero.';
        else payload[field.name] = cents;
        break;
      }
      case 'date':
        if (/^\d{4}-\d{2}-\d{2}$/.test(String(raw))) payload[field.name] = String(raw);
        else problems[field.name] = 'Enter a date.';
        break;
      case 'picker':
        payload[field.name] = (raw as Option[]).map((option) => option.value);
        break;
      case 'tags':
        payload[field.name] = (Array.isArray(raw) ? raw : String(raw).split(',')).map((tag) => String(tag).trim()).filter(Boolean);
        break;
      case 'datetime': {
        const date = new Date(String(raw));
        if (Number.isNaN(date.getTime())) problems[field.name] = 'Enter a date and time.';
        else payload[field.name] = date.toISOString();
        break;
      }
      default:
        payload[field.name] = typeof raw === 'string' ? raw.trim() : raw;
    }
  }
  return { payload, problems };
}

interface FormProps {
  ctx: Ctx;
  fields: FormField[];
  initial?: FormValues;
  confirm?: string;
  /** Asked before submitting; its answer must be confirmed. */
  preview?: (values: FormValues) => Promise<string>;
  submitLabel: string;
  danger?: boolean;
  onSubmit: (values: FormValues, idempotencyKey: string) => Promise<void>;
  onCancel?: () => void;
}

/** The edit step. One idempotency key lives as long as the form, so a retry repeats the same intent. */
export function ActionForm({ ctx, fields, initial, confirm, preview, submitLabel, danger, onSubmit, onCancel }: FormProps) {
  const [values, setValues] = useState<FormValues>(initial ?? {});
  const [problems, setProblems] = useState<Record<string, string>>({});
  const [failure, setFailure] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  const [previewed, setPreviewed] = useState<string>();
  const key = useRef(newIdempotencyKey());
  const helpId = useId();

  useEffect(() => () => { unsaved.dirty = false; }, []);

  const shown = fields.filter((field) => !field.showIf || field.showIf(values));

  async function submit(event: FormEvent) {
    event.preventDefault();
    const { payload, problems: found } = toPayload(fields, values);
    setProblems(found);
    if (Object.keys(found).length > 0) return;
    setBusy(true);
    setFailure(undefined);
    try {
      if (preview && previewed === undefined) {
        setPreviewed(await preview(payload));
        return;
      }
      await onSubmit(payload, key.current);
      unsaved.dirty = false;
    } catch (error) {
      setFailure(error);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={submit} noValidate className="space-y-4">
      {shown.map((field) => {
        const meter = field.meter?.(String(values[field.name] ?? ''));
        return (
          <div key={field.name}>
            <Control
              field={field}
              ctx={ctx}
              value={values[field.name]}
              describedBy={`${helpId}-${field.name}`}
              onChange={(value) => {
                unsaved.dirty = true;
                // What was confirmed no longer matches what is typed.
                setPreviewed(undefined);
                setValues((current) => ({ ...current, [field.name]: value }));
              }}
            />
            <p id={`${helpId}-${field.name}`} className={`mt-1 text-xs ${problems[field.name] ? 'font-semibold text-red-800' : 'text-gray-600'}`}>
              {problems[field.name] ?? [meter, field.help].filter(Boolean).join(' · ')}
            </p>
          </div>
        );
      })}
      {confirm && <p className="text-sm">{confirm}</p>}
      {previewed !== undefined && (
        <p role="status" className="rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-950">
          {previewed}
        </p>
      )}
      {failure !== undefined && <ErrorNote error={failure} />}
      <div className="flex flex-wrap gap-2">
        <Button type="submit" tone={danger ? 'danger' : 'primary'} disabled={busy}>
          {busy ? 'Working…' : preview && previewed === undefined ? 'Check and continue' : preview ? `Confirm: ${submitLabel}` : submitLabel}
        </Button>
        {onCancel && <Button onClick={onCancel} disabled={busy}>Cancel</Button>}
      </div>
    </form>
  );
}
