'use client';

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import BulkImportModal from '@/components/comms/BulkImportModal';
import { apiRequest, type Contact, type ContactInput, type ContactImportResult, type Message, type SMSTemplate } from '@/lib/api';
import { page, resource, type Ctx, type FormValues, type ModuleManifest, type Option } from '@/framework/types';
import { describeSegments } from '@/lib/sms';
import { formatPhone } from '@/lib/phone';
import { Button, Card, ErrorNote, SimpleTable, Spinner, Status, Tile, when } from '@/ui/kit';

const PAGE = 50;

// --- Messages ---------------------------------------------------------------

interface Stats {
  total: number;
  sent: number;
  delivered: number;
  failed: number;
  pending: number;
  delivery_rate: number;
}

/** A message as the view shows it: the record plus its delivery totals. */
type MessageRow = Message & { stats?: Stats };

interface DeliveryLog {
  id: string;
  recipient_name?: string;
  phone: string;
  status: string;
  sent_at?: string;
  delivered_at?: string;
  cost_cents?: number;
  error_code?: string;
  error_message?: string;
}

interface AudienceOptions {
  classes: { grade: string; stream: string }[];
  tags: string[];
  variables: string[];
  max_units: number;
}

interface Estimate {
  recipient_count: number;
  estimated_kes: number;
  sms_units: number;
  invalid_count: number;
  encoding: string;
}

const AUDIENCES: Option[] = [
  { value: 'all_parents', label: 'All parents' },
  { value: 'grade', label: 'Parents of one grade' },
  { value: 'stream', label: 'Parents of one class (grade and stream)' },
  { value: 'fee_defaulters', label: 'Parents with a fee balance' },
  { value: 'transport', label: 'Parents of learners on school transport' },
  { value: 'contacts', label: 'Saved contacts' },
  { value: 'custom', label: 'Parents I choose' },
];

interface GuardianEntry {
  id: string;
  full_name: string;
  phone: string;
  learner_count: number;
  is_sms_opted_out: boolean;
}

const VARIABLE_LABEL: Record<string, string> = {
  parent_name: 'Parent name',
  learner_name: 'Learner name',
  class: 'Class',
  school_name: 'School name',
};

function audienceLabel(row: Pick<Message, 'audience_type' | 'audience_filter'>): string {
  const filter = (row.audience_filter ?? {}) as Record<string, string>;
  switch (row.audience_type) {
    case 'all_parents': return 'All parents';
    case 'grade': return `Parents of ${filter.grade ?? 'a grade'}`;
    case 'stream': return `Parents of ${[filter.grade, filter.stream].filter(Boolean).join(' ')}`;
    case 'fee_defaulters': return 'Parents with a fee balance';
    case 'transport': return 'Transport parents';
    case 'contacts': return filter.tag ? `Contacts tagged "${filter.tag}"` : 'All saved contacts';
    case 'custom': return 'Selected parents';
    case 'resend': return 'Resend to failed recipients';
    default: return row.audience_type;
  }
}

function unique(values: string[]): string[] {
  return values.filter((value, index) => values.indexOf(value) === index);
}

const audienceOptions = () => apiRequest<AudienceOptions>('/messages/audience-options');

/** What the form collects, as the API takes it. */
function messageBody(values: FormValues) {
  const filter: Record<string, string> = {};
  if (values.audience_type === 'grade' || values.audience_type === 'stream') filter.grade = values.grade;
  if (values.audience_type === 'stream') filter.stream = values.stream;
  if (values.audience_type === 'contacts' && values.tag) filter.tag = values.tag;
  if (values.audience_type === 'custom') {
    return {
      channel: 'sms',
      content_type: 'text',
      audience_type: 'custom',
      audience_filter: { guardian_ids: values.guardian_ids ?? [] },
      content: values.content,
      scheduled_at: values.scheduled_at,
    };
  }
  return {
    channel: 'sms',
    content_type: 'text',
    audience_type: values.audience_type,
    audience_filter: filter,
    content: values.content,
    scheduled_at: values.scheduled_at,
  };
}

const ERROR_TEXT: Record<string, string> = {
  INVALID_PHONE: 'The phone number on record cannot be texted.',
  OUTCOME_UNKNOWN: 'Sending was interrupted. This person may or may not have received it.',
  NOT_CONFIGURED: 'SMS is not set up for this school.',
  CANCELLED: 'Cancelled before sending.',
  AT_402: 'The sender name is not approved.',
  AT_403: 'The network rejected this phone number.',
  AT_405: 'The SMS account has run out of credit.',
  AT_406: 'This person has blocked messages from this sender.',
  AT_407: 'The network could not route the message.',
  DELIVERY_FAILED: 'The network could not deliver it.',
};

function DeliveryPanel({ row }: { row: MessageRow; ctx: Ctx }) {
  const [only, setOnly] = useState('');
  const live = row.status === 'sending' || (row.stats?.sent ?? 0) > 0;
  const logs = useQuery({
    queryKey: ['communications', 'messages', 'logs', row.id, only],
    queryFn: () => apiRequest<DeliveryLog[]>(`/messages/${row.id}/logs?limit=500${only ? `&status=${only}` : ''}`),
    // Delivery reports keep arriving for a while after a message is sent.
    refetchInterval: live ? 4000 : false,
  });
  const stats = row.stats;

  return (
    <div className="space-y-3">
      {stats && (
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          <Tile label="Recipients" value={stats.total} />
          <Tile label="Delivered" value={stats.delivered} note="Confirmed by the phone network" />
          <Tile label="Sent" value={stats.sent + stats.pending} note={stats.pending > 0 ? `${stats.pending} still queued` : 'Awaiting a delivery report'} />
          {/* The recipients of a cancelled message were never attempted: that is not a failure. */}
          <Tile label={row.status === 'cancelled' ? 'Not sent' : 'Failed'} value={stats.failed} />
        </div>
      )}
      <Card
        title="Recipients"
        actions={
          <label className="flex items-center gap-2 text-xs text-gray-600">
            Show
            <select
              className="rounded-lg border border-gray-300 bg-white px-2 py-1 text-xs text-gray-900"
              value={only}
              onChange={(event) => setOnly(event.target.value)}
            >
              <option value="">Everyone</option>
              <option value="failed">Failed</option>
              <option value="delivered">Delivered</option>
              <option value="sent">Sent, not yet confirmed</option>
              <option value="pending">Queued</option>
            </select>
          </label>
        }
      >
        {logs.isLoading && <Spinner />}
        {logs.error && <div className="p-4"><ErrorNote error={logs.error} /></div>}
        {logs.data && (
          <SimpleTable
            caption="Recipients of this message"
            rows={logs.data}
            empty="No recipients match."
            columns={[
              { header: 'Recipient', cell: (log) => log.recipient_name || '—' },
              { header: 'Phone', cell: (log) => <span className="whitespace-nowrap tabular-nums">{formatPhone(log.phone)}</span> },
              { header: 'Status', cell: (log) => <Status value={log.error_code === 'CANCELLED' ? 'cancelled' : log.status} /> },
              {
                header: 'Detail',
                cell: (log) =>
                  log.status === 'failed'
                    ? ERROR_TEXT[log.error_code ?? ''] ?? log.error_message ?? 'Failed.'
                    : log.delivered_at ? `Delivered ${when(log.delivered_at)}` : log.sent_at ? `Sent ${when(log.sent_at)}` : '',
              },
            ]}
          />
        )}
      </Card>
    </div>
  );
}

const messages = resource<MessageRow>({
  id: 'messages',
  label: 'Messages',
  noun: 'message',
  scopes: ['school'],
  purpose: 'SMS sent to parents and contacts, with who received each one.',
  list: async (_ctx, filters, cursor) => {
    const offset = Number(cursor ?? 0);
    const query = new URLSearchParams({ channel: 'sms', limit: String(PAGE), offset: String(offset) });
    if (filters.status) query.set('status', filters.status);
    const items = await apiRequest<Message[]>(`/messages?${query}`);
    return { items, next: items.length === PAGE ? String(offset + PAGE) : undefined };
  },
  get: async (_ctx, id) => {
    const result = await apiRequest<{ message: Message; stats: Stats }>(`/messages/${id}`);
    return { ...result.message, stats: result.stats };
  },
  live: (row) => row.status === 'sending' || row.status === 'scheduled' || Boolean(row.stats && row.stats.sent + row.stats.pending > 0),
  rowId: (row) => row.id,
  title: (row) => audienceLabel(row),
  status: (row) => row.status,
  filters: [
    {
      name: 'status', label: 'Status', type: 'select',
      options: ['scheduled', 'sending', 'sent', 'failed', 'cancelled'].map((value) => ({ value, label: value.charAt(0).toUpperCase() + value.slice(1) })),
    },
  ],
  columns: [
    // The list sits beside the open record, so it keeps to what identifies a
    // message; who it went to and the counts are in the view.
    { header: 'Message', cell: (row) => (row.content.length > 36 ? `${row.content.slice(0, 36)}…` : row.content) },
    { header: 'Status', cell: (row) => <Status value={row.status} /> },
    { header: 'When', cell: (row) => <span className="whitespace-nowrap">{when(row.sent_at ?? row.scheduled_at ?? row.created_at)}</span> },
  ],
  fields: [
    { label: 'Message', value: (row) => row.content },
    { label: 'Sent to', value: (row) => audienceLabel(row) },
    { label: 'Length', value: (row) => describeSegments(row.content) },
    { label: 'Created', value: (row) => when(row.created_at) },
    { label: 'Sent / scheduled for', value: (row) => (row.status === 'scheduled' ? when(row.scheduled_at) : when(row.sent_at)) },
  ],
  extra: DeliveryPanel,
  create: {
    id: 'create',
    label: 'New SMS',
    submitLabel: 'Send',
    initial: () => ({ audience_type: 'all_parents' }),
    fields: [
      { name: 'audience_type', label: 'Send to', type: 'select', required: true, options: AUDIENCES },
      {
        name: 'grade', label: 'Grade', type: 'select', required: true,
        showIf: (values) => values.audience_type === 'grade' || values.audience_type === 'stream',
        options: async () => unique((await audienceOptions()).classes.map((c) => c.grade)).map((grade) => ({ value: grade, label: grade })),
      },
      {
        name: 'stream', label: 'Stream', type: 'select', required: true,
        showIf: (values) => values.audience_type === 'stream',
        options: async () => unique((await audienceOptions()).classes.map((c) => c.stream).filter(Boolean)).map((stream) => ({ value: stream, label: stream })),
      },
      {
        name: 'tag', label: 'Only contacts tagged', type: 'select',
        showIf: (values) => values.audience_type === 'contacts',
        options: async () => (await audienceOptions()).tags.map((tag) => ({ value: tag, label: tag })),
        help: 'Leave as "Any" to send to every saved contact.',
      },
      {
        name: 'guardian_ids', label: 'Parents', type: 'picker', required: true,
        showIf: (values) => values.audience_type === 'custom',
        placeholder: 'Search by name or phone number',
        help: 'Parents who have asked not to receive SMS are not listed.',
        search: async (_ctx, query) => {
          const found = await apiRequest<GuardianEntry[] | null>(`/learners/guardians?search=${encodeURIComponent(query)}`);
          return (found ?? [])
            .filter((guardian) => !guardian.is_sms_opted_out)
            .map((guardian) => ({ value: guardian.id, label: guardian.full_name, hint: formatPhone(guardian.phone) }));
        },
      },
      {
        name: 'content', label: 'Message', type: 'textarea', required: true,
        meter: describeSegments,
        inserts: Object.entries(VARIABLE_LABEL).map(([name, label]) => ({ value: `{{${name}}}`, label })),
        help: 'Inserted names are filled in for each recipient.',
      },
      {
        name: 'scheduled_at', label: 'Send later (optional)', type: 'datetime',
        help: 'Leave empty to send now. Uses this device’s clock.',
      },
    ],
    preview: async (_ctx, _row, values) => {
      const estimate = await apiRequest<Estimate>('/messages/estimate', { method: 'POST', body: messageBody(values) });
      if (estimate.recipient_count === 0) {
        throw new Error(estimate.invalid_count > 0
          ? `No one in this audience has a phone number that can be texted (${estimate.invalid_count} invalid).`
          : 'No one matches this audience, so there is nobody to send to.');
      }
      const skipped = estimate.invalid_count > 0 ? ` ${estimate.invalid_count} more cannot be texted (invalid number) and will be listed as failed.` : '';
      const timing = values.scheduled_at ? `It will be sent on ${when(values.scheduled_at)}.` : 'It will be sent now.';
      return `This goes to ${estimate.recipient_count} ${estimate.recipient_count === 1 ? 'person' : 'people'}, ${estimate.sms_units} SMS unit${estimate.sms_units === 1 ? '' : 's'} each: about KES ${estimate.estimated_kes.toFixed(2)}.${skipped} ${timing}`;
    },
    run: (_ctx, _row, values, key) =>
      apiRequest<Message>('/messages', { method: 'POST', body: { ...messageBody(values), idempotency_key: key } }),
    createdId: (message: Message) => message.id,
  },
  actions: [
    {
      id: 'cancel',
      label: 'Cancel this message',
      tone: 'danger',
      when: (row) => row.status === 'scheduled',
      confirm: 'It will not be sent. This cannot be undone; to send it after all, create a new message.',
      submitLabel: 'Cancel the message',
      run: (_ctx, row) => apiRequest<void>(`/messages/${row.id}`, { method: 'DELETE' }),
    },
    {
      id: 'resend',
      label: 'Resend to failed',
      when: (row) => (row.status === 'sent' || row.status === 'failed') && (row.stats?.failed ?? row.failed_count) > 0,
      fields: [
        {
          name: 'include_uncertain', label: 'Also resend where the outcome is unknown', type: 'checkbox',
          help: 'These people may already have received the message; ticking this may text them twice.',
        },
      ],
      confirm: 'A new message with the same text goes to the recipients this one failed to reach. Invalid phone numbers are skipped: correct them first.',
      submitLabel: 'Resend',
      run: (_ctx, row, values, key) =>
        apiRequest<Message>(`/messages/${row.id}/resend-failed`, {
          method: 'POST',
          body: { include_uncertain: Boolean(values.include_uncertain), idempotency_key: key },
        }),
      createdId: (message: Message) => message.id,
    },
  ],
});

// --- Contacts ---------------------------------------------------------------

const contactFields = [
  { name: 'full_name', label: 'Full name', type: 'text' as const, required: true },
  { name: 'phone', label: 'Phone', type: 'text' as const, required: true, placeholder: '0712 345 678', help: 'A Kenyan mobile number; saved as +254…' },
  { name: 'email', label: 'Email', type: 'text' as const },
  { name: 'relationship', label: 'Relationship', type: 'text' as const, placeholder: 'parent, sponsor, supplier…' },
  { name: 'grade_stream', label: 'Class', type: 'text' as const, placeholder: 'Grade 4 North' },
  { name: 'tags', label: 'Tags', type: 'tags' as const, help: 'Separate with commas. A message can be sent to one tag.' },
  { name: 'notes', label: 'Notes', type: 'textarea' as const },
  { name: 'is_opted_out', label: 'Has asked not to receive messages', type: 'checkbox' as const },
];

function contactInput(values: FormValues): ContactInput {
  return {
    full_name: values.full_name,
    phone: values.phone,
    email: values.email ?? '',
    relationship: values.relationship ?? '',
    grade_stream: values.grade_stream ?? '',
    tags: values.tags ?? [],
    notes: values.notes ?? '',
    is_opted_out: Boolean(values.is_opted_out),
  };
}

function contactStatus(row: Contact): string {
  if (!row.is_active) return 'archived';
  return row.is_opted_out ? 'opted out' : 'active';
}

const contacts = resource<Contact>({
  id: 'contacts',
  label: 'Contacts',
  noun: 'contact',
  scopes: ['school'],
  purpose: 'People the school can message who are not parents on a learner record: sponsors, suppliers, prospective parents.',
  list: async (_ctx, filters, cursor) => {
    const offset = Number(cursor ?? 0);
    const query = new URLSearchParams({ limit: String(PAGE), offset: String(offset) });
    if (filters.search) query.set('search', filters.search);
    if (filters.tag) query.set('tag', filters.tag);
    if (filters.status) query.set('status', filters.status);
    const items = await apiRequest<Contact[] | null>(`/contacts?${query}`);
    return { items: items ?? [], next: items && items.length === PAGE ? String(offset + PAGE) : undefined };
  },
  get: (_ctx, id) => apiRequest<Contact>(`/contacts/${id}`),
  rowId: (row) => row.id,
  title: (row) => row.full_name,
  status: contactStatus,
  filters: [
    { name: 'search', label: 'Search', type: 'text', placeholder: 'Name or phone' },
    {
      name: 'status', label: 'Show', type: 'select',
      options: [{ value: 'opted_out', label: 'Opted out' }, { value: 'archived', label: 'Archived' }, { value: 'all', label: 'Everyone' }],
      help: '"Any" shows active contacts.',
    },
  ],
  columns: [
    { header: 'Name', cell: (row) => row.full_name },
    { header: 'Phone', cell: (row) => <span className="whitespace-nowrap tabular-nums">{formatPhone(row.phone)}</span> },
    { header: 'Tags', cell: (row) => row.tags?.join(', ') || '—' },
    { header: 'Status', cell: (row) => <Status value={contactStatus(row)} /> },
  ],
  fields: [
    { label: 'Phone', value: (row) => formatPhone(row.phone) },
    { label: 'Email', value: (row) => row.email || '—' },
    { label: 'Relationship', value: (row) => row.relationship || '—' },
    { label: 'Class', value: (row) => row.grade_stream || '—' },
    { label: 'Tags', value: (row) => row.tags?.join(', ') || '—' },
    { label: 'Added', value: (row) => `${when(row.created_at)} (${row.source})` },
    { label: 'Notes', value: (row) => row.notes || '—' },
  ],
  create: {
    id: 'create',
    label: 'Add contact',
    fields: contactFields,
    run: (_ctx, _row, values) => apiRequest<Contact>('/contacts', { method: 'POST', body: contactInput(values) }),
    createdId: (contact: Contact) => contact.id,
  },
  actions: [
    {
      id: 'edit',
      label: 'Edit',
      when: (row) => row.is_active,
      fields: contactFields,
      initial: (row) => ({ ...row, tags: row.tags?.join(', ') ?? '' }),
      submitLabel: 'Save changes',
      run: (_ctx, row, values) => apiRequest<Contact>(`/contacts/${row.id}`, { method: 'PATCH', body: contactInput(values) }),
    },
    {
      id: 'archive',
      label: 'Archive',
      tone: 'danger',
      when: (row) => row.is_active,
      confirm: 'An archived contact is no longer messaged. Their past messages stay on record, and they can be restored.',
      run: (_ctx, row) => apiRequest<void>(`/contacts/${row.id}`, { method: 'DELETE' }),
    },
    {
      id: 'restore',
      label: 'Restore',
      when: (row) => !row.is_active,
      run: (_ctx, row) => apiRequest<void>(`/contacts/${row.id}/restore`, { method: 'POST' }),
    },
  ],
});

function ImportContacts() {
  const [open, setOpen] = useState(false);
  const [result, setResult] = useState<ContactImportResult>();
  const queryClient = useQueryClient();
  return (
    <Card title="Import contacts">
      <div className="space-y-3 p-4 text-sm">
        <p className="text-gray-700">
          Paste or upload a list (name, phone, and optionally email, relationship, class, tags). Every row is checked
          first and nothing is saved until you confirm.
        </p>
        <Button tone="primary" onClick={() => { setResult(undefined); setOpen(true); }}>Start an import</Button>
        {result && (
          <p role="status" className="rounded-lg bg-emerald-50 p-3 font-semibold text-emerald-900">
            Import finished: {result.created} added, {result.updated} updated, {result.skipped} skipped.
          </p>
        )}
      </div>
      {open && (
        <BulkImportModal
          onClose={() => setOpen(false)}
          onImported={(outcome) => {
            setResult(outcome);
            setOpen(false);
            void queryClient.invalidateQueries({ queryKey: ['communications', 'contacts'] });
          }}
        />
      )}
    </Card>
  );
}

// --- SMS templates ----------------------------------------------------------

const templateFields = [
  { name: 'name', label: 'Name', type: 'text' as const, required: true, placeholder: 'Closing day notice' },
  { name: 'category', label: 'Category', type: 'text' as const, placeholder: 'general, fees, attendance…' },
  {
    name: 'content', label: 'Message', type: 'textarea' as const, required: true,
    meter: describeSegments,
    inserts: Object.entries(VARIABLE_LABEL).map(([name, label]) => ({ value: `{{${name}}}`, label })),
  },
];

function templateBody(values: FormValues) {
  const content = String(values.content ?? '');
  const variables = unique((content.match(/\{\{\s*[a-zA-Z_]+\s*\}\}/g) ?? []).map((match) => match.replace(/[{}\s]/g, '').toLowerCase()));
  return { name: values.name, category: values.category || 'general', content, variables };
}

const templates = resource<SMSTemplate>({
  id: 'templates',
  label: 'SMS templates',
  noun: 'template',
  scopes: ['school'],
  purpose: 'Wording the school reuses. Copy a template into a new SMS.',
  list: async () => ({ items: (await apiRequest<SMSTemplate[] | null>('/sms/templates')) ?? [] }),
  rowId: (row) => row.id,
  title: (row) => row.name,
  columns: [
    { header: 'Name', cell: (row) => row.name },
    { header: 'Category', cell: (row) => row.category },
    { header: 'Message', cell: (row) => (row.content.length > 50 ? `${row.content.slice(0, 50)}…` : row.content) },
  ],
  fields: [
    { label: 'Message', value: (row) => row.content },
    { label: 'Length', value: (row) => describeSegments(row.content) },
    { label: 'Category', value: (row) => row.category },
    { label: 'Fills in', value: (row) => row.variables?.map((name) => VARIABLE_LABEL[name] ?? name).join(', ') || 'Nothing: the same text for everyone' },
  ],
  create: {
    id: 'create',
    label: 'New template',
    fields: templateFields,
    run: (_ctx, _row, values) => apiRequest<SMSTemplate>('/sms/templates', { method: 'POST', body: templateBody(values) }),
    createdId: (template: SMSTemplate) => template.id,
  },
  actions: [
    {
      id: 'edit',
      label: 'Edit',
      fields: templateFields,
      initial: (row) => ({ name: row.name, category: row.category, content: row.content }),
      submitLabel: 'Save changes',
      run: (_ctx, row, values) => apiRequest(`/sms/templates/${row.id}`, { method: 'PATCH', body: templateBody(values) }),
    },
    {
      id: 'delete',
      label: 'Delete',
      tone: 'danger',
      confirm: 'The template is removed. Messages already sent from it are not affected.',
      removes: true,
      run: (_ctx, row) => apiRequest<void>(`/sms/templates/${row.id}`, { method: 'DELETE' }),
    },
  ],
});

export const communications: ModuleManifest = {
  id: 'communications',
  label: 'Communications',
  sections: [
    messages,
    contacts,
    page({
      id: 'import',
      label: 'Import contacts',
      scopes: ['school'],
      purpose: 'Add many contacts at once from a spreadsheet.',
      page: ImportContacts,
    }),
    templates,
  ],
};
