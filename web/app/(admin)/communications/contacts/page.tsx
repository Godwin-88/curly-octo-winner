'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  AlertTriangle,
  CheckCircle2,
  Contact2,
  Pencil,
  Search,
  Upload,
  UserPlus,
  Archive,
  RotateCcw,
} from 'lucide-react';

import { api, Contact, ContactSummary } from '@/lib/api';
import { formatPhone } from '@/lib/phone';
import { useAuth } from '@/lib/auth';
import ContactFormModal from '@/components/comms/ContactFormModal';
import BulkImportModal from '@/components/comms/BulkImportModal';

const PAGE_SIZE = 50;

type StatusFilter = 'active' | 'opted_out' | 'inactive' | 'all';

/**
 * The school contact book — the curated list of people staff can message.
 *
 * This is deliberately a first-class screen rather than a field on the SMS
 * composer: numbers are entered and corrected here, then reused as the
 * "Saved contacts" audience every time a campaign is built.
 */
export default function ContactsPage() {
  const { token } = useAuth();

  const [contacts, setContacts] = useState<Contact[]>([]);
  const [summary, setSummary] = useState<ContactSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const [search, setSearch] = useState('');
  const [tag, setTag] = useState('');
  const [status, setStatus] = useState<StatusFilter>('active');
  const [page, setPage] = useState(0);

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Contact | null>(null);
  const [importOpen, setImportOpen] = useState(false);
  const [confirmArchive, setConfirmArchive] = useState<Contact | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const [rows, stats] = await Promise.all([
        api.listContacts(
          { search, tag, status, limit: PAGE_SIZE, offset: page * PAGE_SIZE },
          token
        ),
        api.getContactSummary(token),
      ]);
      setContacts(rows);
      setSummary(stats);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not load contacts');
    } finally {
      setLoading(false);
    }
  }, [search, tag, status, page, token]);

  // Debounce the search box so typing does not fire a request per keystroke.
  useEffect(() => {
    const t = setTimeout(load, search ? 250 : 0);
    return () => clearTimeout(t);
  }, [load, search]);

  // Filters reset the pager directly in their onChange handlers (rather than
  // in an effect) so the list never flashes the old page first.

  const flash = (message: string) => {
    setNotice(message);
    setTimeout(() => setNotice(null), 4000);
  };

  const openCreate = () => {
    setEditing(null);
    setFormOpen(true);
  };

  // Each filter change returns to the first page of results.
  const changeSearch = (value: string) => {
    setSearch(value);
    setPage(0);
  };
  const changeTag = (value: string) => {
    setTag(value);
    setPage(0);
  };
  const changeStatus = (value: StatusFilter) => {
    setStatus(value);
    setPage(0);
  };

  const openEdit = (contact: Contact) => {
    setEditing(contact);
    setFormOpen(true);
  };

  const onSaved = (contact: Contact, mode: 'created' | 'updated') => {
    setFormOpen(false);
    setEditing(null);
    flash(`${contact.full_name} ${mode === 'created' ? 'saved' : 'updated'}.`);
    load();
  };

  const onImported = () => {
    load();
  };

  const archive = async (contact: Contact) => {
    setBusyId(contact.id);
    try {
      await api.archiveContact(contact.id, token);
      flash(`${contact.full_name} archived.`);
      setConfirmArchive(null);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not archive the contact');
    } finally {
      setBusyId(null);
    }
  };

  const restore = async (contact: Contact) => {
    setBusyId(contact.id);
    try {
      await api.restoreContact(contact.id, token);
      flash(`${contact.full_name} restored.`);
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not restore the contact');
    } finally {
      setBusyId(null);
    }
  };

  const total = summary?.total ?? 0;

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-3 mb-6">
        <div>
          <h1 className="text-2xl font-bold">Contacts</h1>
          <p className="text-sm text-gray-500 mt-1">
            The people this school can message. Curate the list here, then pick{' '}
            <span className="font-medium">Saved contacts</span> as the audience when sending.
          </p>
        </div>
        <div className="flex gap-2">
          <button className="btn-secondary" onClick={() => setImportOpen(true)}>
            <Upload size={16} className="mr-2 inline" />
            Import in bulk
          </button>
          <button className="btn-primary" onClick={openCreate}>
            <UserPlus size={16} className="mr-2 inline" />
            Add contact
          </button>
        </div>
      </div>

      {notice && (
        <p
          role="status"
          className="mb-4 flex items-center gap-2 bg-green-50 border border-green-200 text-green-800 rounded-md p-3 text-sm"
        >
          <CheckCircle2 size={16} />
          {notice}
        </p>
      )}
      {error && (
        <p
          role="alert"
          className="mb-4 flex items-center gap-2 bg-red-50 border border-red-200 text-red-700 rounded-md p-3 text-sm"
        >
          <AlertTriangle size={16} />
          {error}
        </p>
      )}

      {summary && (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
          <div className="card p-4">
            <p className="text-xs text-gray-500 uppercase tracking-wide">Saved contacts</p>
            <p className="text-2xl font-semibold mt-1">{summary.active}</p>
          </div>
          <div className="card p-4">
            <p className="text-xs text-gray-500 uppercase tracking-wide">With phone number</p>
            <p className="text-2xl font-semibold mt-1">{summary.with_phone}</p>
          </div>
          <div className="card p-4">
            <p className="text-xs text-gray-500 uppercase tracking-wide">Opted out</p>
            <p className="text-2xl font-semibold mt-1">{summary.opted_out}</p>
          </div>
          <div className="card p-4">
            <p className="text-xs text-gray-500 uppercase tracking-wide">Segments (tags)</p>
            <p className="text-2xl font-semibold mt-1">{Object.keys(summary.by_tag).length}</p>
          </div>
        </div>
      )}

      <div className="card p-4 mb-4">
        <div className="flex flex-wrap items-end gap-3">
          <div className="flex-1 min-w-[220px]">
            <label className="label" htmlFor="contacts-search">
              Search
            </label>
            <div className="relative">
              <Search
                size={16}
                className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
                aria-hidden="true"
              />
              <input
                id="contacts-search"
                className="input pl-9"
                placeholder="Name, phone or email"
                value={search}
                onChange={(e) => changeSearch(e.target.value)}
              />
            </div>
          </div>

          <div className="w-44">
            <label className="label" htmlFor="contacts-status">
              Status
            </label>
            <select
              id="contacts-status"
              className="input"
              value={status}
              onChange={(e) => changeStatus(e.target.value as StatusFilter)}
            >
              <option value="active">Active</option>
              <option value="opted_out">Opted out</option>
              <option value="inactive">Archived</option>
              <option value="all">All</option>
            </select>
          </div>

          <div className="w-44">
            <label className="label" htmlFor="contacts-tag">
              Tag
            </label>
            <select
              id="contacts-tag"
              className="input"
              value={tag}
              onChange={(e) => changeTag(e.target.value)}
            >
              <option value="">All tags</option>
              {summary &&
                Object.entries(summary.by_tag).map(([t, n]) => (
                  <option key={t} value={t}>
                    {t} ({n})
                  </option>
                ))}
            </select>
          </div>
        </div>
      </div>


      {loading ? (
        <p className="text-center py-12 text-gray-500">Loading contacts…</p>
      ) : contacts.length === 0 ? (
        <div className="card p-10 text-center">
          <Contact2 size={32} className="mx-auto text-gray-400" />
          <h2 className="font-semibold mt-3">
            {total === 0 ? 'No contacts yet' : 'No contacts match these filters'}
          </h2>
          <p className="text-sm text-gray-600 mt-1 max-w-md mx-auto">
            {total === 0
              ? 'Add the people you message — parents, sponsors, staff. You can add them one at a time or paste a whole list from Excel.'
              : 'Try a different search term, tag or status.'}
          </p>
          {total === 0 && (
            <div className="flex justify-center gap-2 mt-5">
              <button className="btn-primary" onClick={openCreate}>
                <UserPlus size={16} className="mr-2 inline" />
                Add your first contact
              </button>
              <button className="btn-secondary" onClick={() => setImportOpen(true)}>
                <Upload size={16} className="mr-2 inline" />
                Import in bulk
              </button>
            </div>
          )}
        </div>
      ) : (
        <>
          <div className="card overflow-x-auto">
            <table className="w-full">
              <caption className="sr-only">Saved contacts</caption>
              <thead className="bg-gray-50 text-left text-sm">
                <tr>
                  <th scope="col" className="px-4 py-3 font-medium text-gray-600">Name</th>
                  <th scope="col" className="px-4 py-3 font-medium text-gray-600">Phone</th>
                  <th scope="col" className="px-4 py-3 font-medium text-gray-600">Relationship</th>
                  <th scope="col" className="px-4 py-3 font-medium text-gray-600">Class</th>
                  <th scope="col" className="px-4 py-3 font-medium text-gray-600">Tags</th>
                  <th scope="col" className="px-4 py-3 font-medium text-gray-600">
                    <span className="sr-only">Actions</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {contacts.map((c) => (
                  <tr key={c.id} className="border-t hover:bg-gray-50">
                    <td className="px-4 py-3">
                      <p className="font-medium text-sm">{c.full_name}</p>
                      {c.email && <p className="text-xs text-gray-500">{c.email}</p>}
                      {c.is_opted_out && (
                        <span className="inline-block mt-1 text-xs font-medium text-amber-700 bg-amber-50 rounded-full px-2 py-0.5">
                          opted out
                        </span>
                      )}
                      {!c.is_active && (
                        <span className="inline-block mt-1 text-xs font-medium text-gray-600 bg-gray-100 rounded-full px-2 py-0.5">
                          archived
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-sm whitespace-nowrap">
                      {formatPhone(c.phone)}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 capitalize">
                      {c.relationship || '—'}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600">{c.grade_stream || '—'}</td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {(c.tags ?? []).map((t) => (
                          <span
                            key={t}
                            className="text-xs font-medium text-gray-700 bg-gray-100 rounded-full px-2 py-0.5"
                          >
                            {t}
                          </span>
                        ))}
                        {(c.tags ?? []).length === 0 && (
                          <span className="text-sm text-gray-400">—</span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-right whitespace-nowrap">
                      {c.is_active ? (
                        <>
                          <button
                            className="p-2 text-gray-500 hover:text-blue-600"
                            onClick={() => openEdit(c)}
                            aria-label={`Edit ${c.full_name}`}
                            title="Edit"
                          >
                            <Pencil size={16} />
                          </button>
                          <button
                            className="p-2 text-gray-500 hover:text-amber-600"
                            onClick={() => setConfirmArchive(c)}
                            aria-label={`Archive ${c.full_name}`}
                            title="Archive"
                          >
                            <Archive size={16} />
                          </button>
                        </>
                      ) : (
                        <button
                          className="p-2 text-gray-500 hover:text-green-600"
                          onClick={() => restore(c)}
                          disabled={busyId === c.id}
                          aria-label={`Restore ${c.full_name}`}
                          title="Restore"
                        >
                          <RotateCcw size={16} />
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>


          {total > PAGE_SIZE && (
            <nav className="flex items-center justify-between mt-4" aria-label="Pagination">
              <p className="text-sm text-gray-600">
                Page {page + 1} of {Math.ceil(total / PAGE_SIZE)}
              </p>
              <div className="flex gap-2">
                <button
                  className="btn-secondary"
                  onClick={() => setPage((p) => Math.max(0, p - 1))}
                  disabled={page === 0}
                >
                  Previous
                </button>
                <button
                  className="btn-secondary"
                  onClick={() => setPage((p) => p + 1)}
                  disabled={(page + 1) * PAGE_SIZE >= total}
                >
                  Next
                </button>
              </div>
            </nav>
          )}
        </>
      )}

      {formOpen && (
        <ContactFormModal
          contact={editing}
          onClose={() => {
            setFormOpen(false);
            setEditing(null);
          }}
          onSaved={onSaved}
        />
      )}

      {importOpen && (
        <BulkImportModal
          onClose={() => setImportOpen(false)}
          onImported={onImported}
        />
      )}

      {confirmArchive && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="archive-title"
            className="bg-white rounded-lg shadow-xl p-6 w-full max-w-md"
          >
            <h2 id="archive-title" className="text-lg font-semibold">
              Archive {confirmArchive.full_name}?
            </h2>
            <p className="text-sm text-gray-600 mt-2">
              They will be removed from your contact list and from every campaign audience. Their
              message history is kept, and you can restore them later.
            </p>
            <div className="flex justify-end gap-3 mt-6">
              <button
                className="btn-secondary"
                onClick={() => setConfirmArchive(null)}
                disabled={busyId === confirmArchive.id}
              >
                Cancel
              </button>
              <button
                className="btn-primary"
                onClick={() => archive(confirmArchive)}
                disabled={busyId === confirmArchive.id}
              >
                {busyId === confirmArchive.id ? 'Archiving…' : 'Archive contact'}
              </button>
            </div>
          </div>
        </div>
      )}

      {!loading && total > 0 && (
        <p className="text-xs text-gray-500 mt-6">
          Contacts are scoped to this school. Editing a number here updates every future campaign
          that uses it.
        </p>
      )}
    </div>
  );
}
