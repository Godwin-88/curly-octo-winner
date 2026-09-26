'use client';

import { useEffect, useRef, useState } from 'react';
import { X } from 'lucide-react';

import { APIError, api, Contact, ContactInput } from '@/lib/api';
import { formatPhone, normalizePhone } from '@/lib/phone';
import { useAuth } from '@/lib/auth';

interface Props {
  /** null = create a new contact, a contact = edit that contact. */
  contact: Contact | null;
  onClose: () => void;
  onSaved: (contact: Contact, mode: 'created' | 'updated') => void;
}

const RELATIONSHIPS = [
  { value: 'parent', label: 'Parent / guardian' },
  { value: 'sponsor', label: 'Sponsor' },
  { value: 'staff', label: 'Staff' },
  { value: 'prospective', label: 'Prospective family' },
  { value: 'vendor', label: 'Vendor / supplier' },
  { value: 'alumni', label: 'Alumni' },
  { value: 'other', label: 'Other' },
];

const EMPTY = {
  full_name: '',
  phone: '',
  email: '',
  relationship: 'parent',
  grade_stream: '',
  tags: '',
  notes: '',
  is_opted_out: false,
};

/**
 * Create / edit dialog for a single contact.
 *
 * Field-level validation runs while the user types (a school should never have
 * to press Save to learn a number is wrong), but the server's message always
 * wins when it disagrees: the API is the authority on duplicates.
 */
export default function ContactFormModal({ contact, onClose, onSaved }: Props) {
  const { token } = useAuth();
  const editing = Boolean(contact);

  const [form, setForm] = useState({ ...EMPTY });
  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [saving, setSaving] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);
  const [duplicate, setDuplicate] = useState<string | null>(null);

  const nameRef = useRef<HTMLInputElement>(null);

  // Seed the form from the contact being edited.
  useEffect(() => {
    if (contact) {
      setForm({
        full_name: contact.full_name,
        phone: contact.phone,
        email: contact.email ?? '',
        relationship: contact.relationship ?? 'parent',
        grade_stream: contact.grade_stream ?? '',
        tags: (contact.tags ?? []).join(', '),
        notes: contact.notes ?? '',
        is_opted_out: contact.is_opted_out,
      });
    } else {
      setForm({ ...EMPTY });
    }
    setServerError(null);
    setDuplicate(null);
    setTouched({});
    nameRef.current?.focus();
  }, [contact]);

  // Escape closes the dialog.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !saving) onClose();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose, saving]);

  const set = (key: keyof typeof EMPTY, value: string | boolean) =>
    setForm((f) => ({ ...f, [key]: value }));

  const nameError = touched.full_name && !form.full_name.trim() ? 'Name is required' : '';
  const phonePreview = form.phone ? normalizePhone(form.phone) : null;
  const phoneError =
    touched.phone && form.phone.trim() && !phonePreview
      ? 'Enter a Kenyan mobile number, e.g. 0712345678'
      : '';
  const emailError =
    touched.email && form.email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email.trim())
      ? 'That does not look like an email address'
      : '';

  const invalid = !form.full_name.trim() || !phonePreview;
  const canSubmit = !invalid && !saving;

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setTouched({ full_name: true, phone: true, email: true });
    if (invalid) return;

    const payload: ContactInput = {
      full_name: form.full_name.trim(),
      phone: form.phone.trim(),
      email: form.email.trim() || undefined,
      relationship: form.relationship || undefined,
      grade_stream: form.grade_stream.trim() || undefined,
      tags: form.tags
        .split(',')
        .map((t) => t.trim().toLowerCase())
        .filter(Boolean),
      notes: form.notes.trim() || undefined,
      is_opted_out: form.is_opted_out,
    };

    setSaving(true);
    setServerError(null);
    setDuplicate(null);
    try {
      const saved = contact
        ? await api.updateContact(contact.id, payload, token)
        : await api.createContact(payload, token);
      onSaved(saved, contact ? 'updated' : 'created');
    } catch (err) {
      if (err instanceof APIError && err.status === 409) {
        setDuplicate(err.message);
      } else {
        setServerError(err instanceof Error ? err.message : 'Could not save the contact');
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/50 p-4 overflow-y-auto"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget && !saving) onClose();
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="contact-form-title"
        className="bg-white rounded-lg shadow-xl w-full max-w-2xl my-8"
      >
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 id="contact-form-title" className="text-lg font-semibold">
            {editing ? 'Edit contact' : 'Add contact'}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded-md text-gray-500 hover:bg-gray-100"
            aria-label="Close"
            disabled={saving}
          >
            <X size={18} />
          </button>
        </div>

        <form onSubmit={submit} className="px-6 py-5 space-y-4" noValidate>
          {serverError && (
            <p
              role="alert"
              className="bg-red-50 border border-red-200 text-red-700 rounded-md p-3 text-sm"
            >
              {serverError}
            </p>
          )}
          {duplicate && (
            <p
              role="alert"
              className="bg-amber-50 border border-amber-200 text-amber-800 rounded-md p-3 text-sm"
            >
              {duplicate} Close this dialog and edit the existing contact instead.
            </p>
          )}

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="label" htmlFor="contact-name">
                Full name <span className="text-red-600">*</span>
              </label>
              <input
                id="contact-name"
                ref={nameRef}
                className="input"
                value={form.full_name}
                onChange={(e) => set('full_name', e.target.value)}
                onBlur={() => setTouched((t) => ({ ...t, full_name: true }))}
                aria-invalid={Boolean(nameError)}
                aria-describedby={nameError ? 'contact-name-error' : undefined}
                required
              />
              {nameError && (
                <p id="contact-name-error" className="text-sm text-red-600 mt-1">
                  {nameError}
                </p>
              )}
            </div>

            <div>
              <label className="label" htmlFor="contact-phone">
                Phone <span className="text-red-600">*</span>
              </label>
              <input
                id="contact-phone"
                className="input"
                type="tel"
                inputMode="tel"
                placeholder="0712345678"
                value={form.phone}
                onChange={(e) => set('phone', e.target.value)}
                onBlur={() => setTouched((t) => ({ ...t, phone: true }))}
                aria-invalid={Boolean(phoneError)}
                aria-describedby="contact-phone-help"
                required
              />
              {phoneError ? (
                <p id="contact-phone-help" className="text-sm text-red-600 mt-1">
                  {phoneError}
                </p>
              ) : phonePreview && phonePreview !== form.phone ? (
                <p id="contact-phone-help" className="text-sm text-gray-500 mt-1">
                  Saved as {formatPhone(phonePreview)}
                </p>
              ) : (
                <p id="contact-phone-help" className="text-sm text-gray-500 mt-1">
                  0712345678 or +254712345678
                </p>
              )}
            </div>

            <div>
              <label className="label" htmlFor="contact-email">
                Email
              </label>
              <input
                id="contact-email"
                className="input"
                type="email"
                value={form.email}
                onChange={(e) => set('email', e.target.value)}
                onBlur={() => setTouched((t) => ({ ...t, email: true }))}
                aria-invalid={Boolean(emailError)}
              />
              {emailError && <p className="text-sm text-red-600 mt-1">{emailError}</p>}
            </div>

            <div>
              <label className="label" htmlFor="contact-relationship">
                Relationship
              </label>
              <select
                id="contact-relationship"
                className="input"
                value={form.relationship}
                onChange={(e) => set('relationship', e.target.value)}
              >
                {RELATIONSHIPS.map((r) => (
                  <option key={r.value} value={r.value}>
                    {r.label}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="label" htmlFor="contact-class">
                Class / grade
              </label>
              <input
                id="contact-class"
                className="input"
                placeholder="Grade 4 North"
                value={form.grade_stream}
                onChange={(e) => set('grade_stream', e.target.value)}
              />
            </div>

            <div>
              <label className="label" htmlFor="contact-tags">
                Tags
              </label>
              <input
                id="contact-tags"
                className="input"
                placeholder="grade 4, boarding"
                value={form.tags}
                onChange={(e) => set('tags', e.target.value)}
                aria-describedby="contact-tags-help"
              />
              <p id="contact-tags-help" className="text-sm text-gray-500 mt-1">
                Comma separated. Tags become audience segments when sending.
              </p>
            </div>
          </div>


          <div>
            <label className="label" htmlFor="contact-notes">
              Notes
            </label>
            <textarea
              id="contact-notes"
              className="input"
              rows={2}
              value={form.notes}
              onChange={(e) => set('notes', e.target.value)}
            />
          </div>

          <label className="flex items-start gap-2 text-sm">
            <input
              type="checkbox"
              className="mt-0.5"
              checked={form.is_opted_out}
              onChange={(e) => set('is_opted_out', e.target.checked)}
            />
            <span>
              Opted out of messages
              <span className="block text-xs text-gray-500">
                Opted-out contacts are never included in a campaign.
              </span>
            </span>
          </label>

          <div className="flex justify-end gap-3 pt-4 border-t">
            <button type="button" className="btn-secondary" onClick={onClose} disabled={saving}>
              Cancel
            </button>
            <button type="submit" className="btn-primary" disabled={!canSubmit}>
              {saving ? 'Saving…' : editing ? 'Save changes' : 'Save contact'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
