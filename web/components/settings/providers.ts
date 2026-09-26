// Provider metadata for Settings → Integrations.
//
// `secret: true` marks a credential: it is sent once, stored encrypted, and
// never returned to the browser. Everything else is ordinary configuration that
// a principal can read back.

export interface IntegrationField {
  key: string;
  label: string;
  secret?: boolean;
  placeholder?: string;
  help?: string;
}

export interface ProviderSpec {
  id: string;
  name: string;
  purpose: string;
  docsHint: string;
  fields: IntegrationField[];
}

export const PROVIDERS: ProviderSpec[] = [
  {
    id: 'mpesa',
    name: 'M-Pesa (Safaricom Daraja)',
    purpose: 'Parents pay fee invoices by STK push and get a confirmation.',
    docsHint: 'Daraja portal → App → your sandbox/live app.',
    fields: [
      { key: 'consumer_key', label: 'Consumer key', secret: true },
      { key: 'consumer_secret', label: 'Consumer secret', secret: true },
      { key: 'passkey', label: 'Passkey (STK)', secret: true },
      { key: 'shortcode', label: 'Paybill / shortcode', placeholder: '174379' },
      { key: 'base_url', label: 'Base URL', placeholder: 'https://sandbox.safaricomm.co.ke' },
      {
        key: 'callback_url',
        label: 'Callback URL',
        placeholder: 'https://your-api.onrender.com/api/v1/webhooks/mpesa/callback',
        help: 'Register this exact URL with Safaricom — payment results arrive here.',
      },
    ],
  },
  {
    id: 'africastalking',
    name: "Africa's Talking (SMS)",
    purpose: 'Fee reminders, results and attendance SMS to parents.',
    docsHint: "Africa's Talking account → Developers → API key.",
    fields: [
      { key: 'api_key', label: 'API key', secret: true },
      { key: 'username', label: 'Username' },
      {
        key: 'sender_id',
        label: 'Sender ID',
        placeholder: 'SHULE360',
        help: 'Must be approved by Safaricom before messages go out.',
      },
    ],
  },
  {
    id: 'whatsapp',
    name: 'WhatsApp Cloud API',
    purpose: 'Two-way parent messaging and the school inbox.',
    docsHint: 'Meta for Developers → WhatsApp → API setup.',
    fields: [
      { key: 'access_token', label: 'Permanent access token', secret: true },
      { key: 'phone_number_id', label: 'Phone number ID' },
      { key: 'business_account_id', label: 'Business account ID' },
      {
        key: 'verify_token',
        label: 'Webhook verify token',
        secret: true,
        help: 'You choose this value; it secures the inbound webhook.',
      },
    ],
  },
  {
    id: 'backblaze',
    name: 'Backblaze B2 (documents)',
    purpose: 'Storage for learner documents and report-card PDFs.',
    docsHint: 'B2 → Account → Keys.',
    fields: [
      { key: 'account_id', label: 'Key ID', secret: true },
      { key: 'application_key', label: 'Application key', secret: true },
      { key: 'bucket_name', label: 'Bucket name' },
      { key: 'endpoint', label: 'Endpoint', placeholder: 'https://api.backblazeb2.com' },
    ],
  },
  {
    id: 'groq',
    name: 'Groq (AI assistant)',
    purpose: 'The AI assistant that drafts messages, reports and summaries.',
    docsHint: 'console.groq.com → API keys.',
    fields: [{ key: 'api_key', label: 'API key', secret: true }],
  },
  {
    id: 'upstash',
    name: 'Upstash Redis',
    purpose: 'Sign-in rate limiting and fast shared caches.',
    docsHint: 'Upstash console → Database → REST credentials.',
    fields: [
      { key: 'redis_url', label: 'REST URL', placeholder: 'https://db-name.upstash.io' },
      { key: 'redis_token', label: 'REST token', secret: true },
    ],
  },
];
