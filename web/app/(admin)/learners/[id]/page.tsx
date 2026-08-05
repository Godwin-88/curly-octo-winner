'use client';

import { use, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import {
  ArrowLeft, User, FileText, TrendingUp, CalendarCheck, Download,
  Trash2, ToggleLeft, ToggleRight, GraduationCap, RefreshCw, Send
} from 'lucide-react';
import { api, Learner, GuardianBrief, LearnerDocument, LearnerProgression, AttendanceRecord } from '@/lib/api';

const GRADES = ['PP1', 'PP2', 'Grade 1', 'Grade 2', 'Grade 3', 'Grade 4', 'Grade 5', 'Grade 6', 'Grade 7', 'Grade 8', 'Grade 9'];
const DOC_TYPES = ['birth_cert', 'report_card', 'medical', 'transfer', 'other'];

type Tab = 'overview' | 'documents' | 'progression' | 'attendance';

export default function LearnerDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const token = ''; // TODO: Get from auth context

  const [learner, setLearner] = useState<Learner | null>(null);
  const [guardians, setGuardians] = useState<GuardianBrief[]>([]);
  const [documents, setDocuments] = useState<LearnerDocument[]>([]);
  const [progressions, setProgressions] = useState<LearnerProgression[]>([]);
  const [attendance, setAttendance] = useState<AttendanceRecord[]>([]);
  const [tab, setTab] = useState<Tab>('overview');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [showPromote, setShowPromote] = useState(false);
  const [showRetain, setShowRetain] = useState(false);
  const [showTransferOut, setShowTransferOut] = useState(false);
  const [showTransferIn, setShowTransferIn] = useState(false);
  const [showDocUpload, setShowDocUpload] = useState(false);
  const [actionError, setActionError] = useState('');
  const [toGrade, setToGrade] = useState('');
  const [actionNotes, setActionNotes] = useState('');
  const [actionYear, setActionYear] = useState(String(new Date().getFullYear()));

  const [docForm, setDocForm] = useState({ doc_type: 'other', file_name: '', file_url: '', mime_type: '', file_size: '' });

  const loadAll = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [l, g, d, p, a] = await Promise.all([
        api.getLearner(id, token),
        api.listLearnerGuardians(id, token),
        api.listLearnerDocuments(id, token),
        api.listLearnerProgressions(id, token),
        api.listAttendanceByLearner(id, token),
      ]);
      setLearner(l);
      setGuardians(g);
      setDocuments(d);
      setProgressions(p);
      setAttendance(a);
    } catch (e: any) {
      setError(e.message || 'Failed to load learner');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, token]);

  const runAction = async (fn: () => Promise<unknown>) => {
    setActionError('');
    try {
      await fn();
      setShowPromote(false); setShowRetain(false);
      setShowTransferOut(false); setShowTransferIn(false);
      setToGrade(''); setActionNotes(''); setActionYear(String(new Date().getFullYear()));
      await loadAll();
    } catch (e: any) {
      setActionError(e.message || 'Action failed');
    }
  };

  const handleToggleActive = () => {
    if (!learner) return;
    runAction(async () => {
      if (learner.is_active) await api.deactivateLearner(id, token);
      else await api.reactivateLearner(id, token);
    });
  };

  const handleUploadDoc = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionError('');
    try {
      await api.uploadLearnerDocument(id, {
        doc_type: docForm.doc_type,
        file_name: docForm.file_name,
        file_url: docForm.file_url,
        mime_type: docForm.mime_type || undefined,
        file_size: docForm.file_size ? Number(docForm.file_size) : undefined,
      }, token);
      setShowDocUpload(false);
      setDocForm({ doc_type: 'other', file_name: '', file_url: '', mime_type: '', file_size: '' });
      setDocuments(await api.listLearnerDocuments(id, token));
    } catch (e: any) {
      setActionError(e.message || 'Upload failed');
    }
  };

  const handleDeleteDoc = (docId: string) => {
    if (!confirm('Delete this document?')) return;
    runAction(async () => {
      await api.deleteLearnerDocument(docId, token);
      setDocuments(await api.listLearnerDocuments(id, token));
    });
  };

  if (loading) return <div className="p-8 text-center text-gray-400">Loading learner...</div>;
  if (error && !learner) return <div className="bg-red-50 text-red-700 p-3 rounded-md text-sm">{error}</div>;
  if (!learner) return <div className="p-8 text-center">Learner not found</div>;

  const tabs: { key: Tab; label: string; icon: any }[] = [
    { key: 'overview', label: 'Overview', icon: User },
    { key: 'documents', label: `Documents (${documents.length})`, icon: FileText },
    { key: 'progression', label: `Progression (${progressions.length})`, icon: TrendingUp },
    { key: 'attendance', label: `Attendance (${attendance.length})`, icon: CalendarCheck },
  ];

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-4">
          <button onClick={() => router.push('/learners')} className="p-2 rounded-md hover:bg-gray-100">
            <ArrowLeft size={20} />
          </button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold">{learner.full_name}</h1>
              <span className={`px-2 py-0.5 rounded-full text-xs ${
                learner.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'
              }`}>
                {learner.is_active ? 'Active' : 'Inactive'}
              </span>
              {learner.special_needs && (
                <span className="px-2 py-0.5 rounded-full text-xs bg-yellow-100 text-yellow-700">Special Needs</span>
              )}
            </div>
            <p className="text-sm text-gray-500">
              {learner.grade} {learner.stream && `· Stream ${learner.stream}`} · {learner.upi}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Link href={`/academic/attendance`} className="btn-secondary flex items-center gap-2">
            <CalendarCheck size={16} /> Attendance
          </Link>
          <button onClick={() => setShowPromote(true)} className="btn-primary flex items-center gap-2">
            <GraduationCap size={16} /> Promote
          </button>
          <button
            onClick={handleToggleActive}
            className={`btn-secondary flex items-center gap-2 ${learner.is_active ? 'text-red-600' : 'text-green-600'}`}
          >
            {learner.is_active ? <ToggleLeft size={16} /> : <ToggleRight size={16} />}
            {learner.is_active ? 'Deactivate' : 'Reactivate'}
          </button>
        </div>
      </div>

      {actionError && <div className="bg-red-50 text-red-700 p-3 rounded-md mb-4 text-sm">{actionError}</div>}

      {/* Tabs */}
      <div className="flex gap-2 mb-6 border-b">
        {tabs.map((t) => {
          const Icon = t.icon;
          const active = tab === t.key;
          return (
            <button
              key={t.key}
              onClick={() => setTab(t.key)}
              className={`flex items-center gap-2 px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                active ? 'border-blue-600 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              <Icon size={16} />
              {t.label}
            </button>
          );
        })}
      </div>

      {/* Tab content */}
      {tab === 'overview' && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
            <div className="card p-6">
              <h3 className="font-semibold mb-4">Personal Information</h3>
              <div className="grid grid-cols-2 gap-4">
                <div><p className="text-xs text-gray-500">Full Name</p><p className="font-medium">{learner.full_name}</p></div>
                <div><p className="text-xs text-gray-500">UPI</p><p className="font-medium">{learner.upi}</p></div>
                <div><p className="text-xs text-gray-500">Date of Birth</p><p className="font-medium">{learner.date_of_birth ? new Date(learner.date_of_birth).toLocaleDateString() : '—'}</p></div>
                <div><p className="text-xs text-gray-500">Grade / Stream</p><p className="font-medium">{learner.grade} {learner.stream && `· ${learner.stream}`}</p></div>
                <div><p className="text-xs text-gray-500">Birth Certificate No.</p><p className="font-medium">{learner.birth_cert_no || '—'}</p></div>
                <div><p className="text-xs text-gray-500">Entry Level</p><p className="font-medium">{learner.entry_level || '—'}</p></div>
                <div><p className="text-xs text-gray-500">Admission Date</p><p className="font-medium">{learner.admission_date ? new Date(learner.admission_date).toLocaleDateString() : '—'}</p></div>
                <div><p className="text-xs text-gray-500">Special Needs</p><p className="font-medium">{learner.special_needs ? 'Yes' : 'No'}</p></div>
              </div>
            </div>

            <div className="card p-6">
              <h3 className="font-semibold mb-4">Guardians</h3>
              {guardians.length === 0 ? (
                <p className="text-sm text-gray-400">No guardians linked</p>
              ) : (
                <div className="space-y-3">
                  {guardians.map((g) => (
                    <div key={g.id} className="flex items-center justify-between p-3 border rounded-md">
                      <div>
                        <p className="font-medium text-sm">{g.full_name}</p>
                        <p className="text-xs text-gray-500">{g.relation}</p>
                      </div>
                      <p className="text-sm text-gray-600">{g.phone}</p>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          <div className="space-y-6">
            <div className="card p-6">
              <h3 className="font-semibold mb-4">Actions</h3>
              <div className="space-y-2">
                <button onClick={() => setShowRetain(true)} className="w-full btn-secondary flex items-center gap-2 text-sm">
                  <RefreshCw size={16} /> Retain in Grade
                </button>
                <button onClick={() => setShowTransferOut(true)} className="w-full btn-secondary flex items-center gap-2 text-sm text-red-600">
                  <Send size={16} /> Transfer Out
                </button>
                <button onClick={() => setShowTransferIn(true)} className="w-full btn-secondary flex items-center gap-2 text-sm text-green-600">
                  <Send size={16} /> Transfer In
                </button>
              </div>
            </div>

            <div className="card p-6">
              <h3 className="font-semibold mb-4">Quick Stats</h3>
              <div className="space-y-3">
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Total Documents</span>
                  <span className="font-medium">{documents.length}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Attendance Records</span>
                  <span className="font-medium">{attendance.length}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-gray-500">Progression Events</span>
                  <span className="font-medium">{progressions.length}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {tab === 'documents' && (
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold">Documents</h3>
            <button onClick={() => setShowDocUpload(true)} className="btn-primary text-sm flex items-center gap-2">
              <FileText size={16} /> Upload Document
            </button>
          </div>

          {showDocUpload && (
            <form onSubmit={handleUploadDoc} className="border rounded-lg p-4 mb-4 space-y-3">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-700 mb-1">Document Type</label>
                  <select value={docForm.doc_type} onChange={(e) => setDocForm({ ...docForm, doc_type: e.target.value })} className="w-full px-3 py-2 border rounded-md text-sm">
                    {DOC_TYPES.map((t) => (
                      <option key={t} value={t}>{t.replace('_', ' ').toUpperCase()}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-700 mb-1">File Name</label>
                  <input type="text" value={docForm.file_name} onChange={(e) => setDocForm({ ...docForm, file_name: e.target.value })} required className="w-full px-3 py-2 border rounded-md text-sm" placeholder="birth-certificate.pdf" />
                </div>
                <div className="sm:col-span-2">
                  <label className="block text-xs font-medium text-gray-700 mb-1">File URL</label>
                  <input type="text" value={docForm.file_url} onChange={(e) => setDocForm({ ...docForm, file_url: e.target.value })} required className="w-full px-3 py-2 border rounded-md text-sm" placeholder="https://..." />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-700 mb-1">MIME Type</label>
                  <input type="text" value={docForm.mime_type} onChange={(e) => setDocForm({ ...docForm, mime_type: e.target.value })} className="w-full px-3 py-2 border rounded-md text-sm" placeholder="application/pdf" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-700 mb-1">Size (bytes)</label>
                  <input type="number" value={docForm.file_size} onChange={(e) => setDocForm({ ...docForm, file_size: e.target.value })} className="w-full px-3 py-2 border rounded-md text-sm" />
                </div>
              </div>
              <div className="flex gap-2">
                <button type="submit" className="btn-primary text-sm">Save</button>
                <button type="button" onClick={() => setShowDocUpload(false)} className="btn-secondary text-sm">Cancel</button>
              </div>
            </form>
          )}

          {documents.length === 0 ? (
            <p className="text-sm text-gray-400">No documents uploaded</p>
          ) : (
            <div className="space-y-2">
              {documents.map((doc) => (
                <div key={doc.id} className="flex items-center justify-between p-3 border rounded-md">
                  <div className="flex items-center gap-3">
                    <FileText className="text-blue-500" size={20} />
                    <div>
                      <p className="font-medium text-sm">{doc.file_name}</p>
                      <p className="text-xs text-gray-500">
                        {doc.doc_type.replace('_', ' ').toUpperCase()} · {doc.mime_type || 'unknown'} · {doc.file_size ? `${(doc.file_size / 1024).toFixed(1)} KB` : '—'}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <a href={doc.file_url} target="_blank" rel="noopener noreferrer" className="p-2 rounded-md hover:bg-gray-100 text-blue-600">
                      <Download size={16} />
                    </a>
                    <button onClick={() => handleDeleteDoc(doc.id)} className="p-2 rounded-md hover:bg-red-50 text-red-500">
                      <Trash2 size={16} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {tab === 'progression' && (
        <div className="card p-6">
          <h3 className="font-semibold mb-4">Progression History</h3>
          {progressions.length === 0 ? (
            <p className="text-sm text-gray-400">No progression events recorded</p>
          ) : (
            <div className="space-y-2">
              {progressions.map((p) => (
                <div key={p.id} className="flex items-center justify-between p-3 border rounded-md">
                  <div className="flex items-center gap-3">
                    <TrendingUp className="text-blue-500" size={20} />
                    <div>
                      <p className="font-medium text-sm">
                        {p.action === 'promote' && `${p.from_grade} → ${p.to_grade}`}
                        {p.action === 'retain' && `${p.from_grade} (retained)`}
                        {p.action === 'transfer_out' && `Transfer out from ${p.from_grade}`}
                        {p.action === 'transfer_in' && `Transfer in to ${p.to_grade}`}
                      </p>
                      <p className="text-xs text-gray-500">
                        Term {p.term || '—'} · {p.year}
                        {p.notes && ` · ${p.notes}`}
                      </p>
                    </div>
                  </div>
                  <span className={`px-2 py-0.5 rounded-full text-xs ${
                    p.action === 'promote' ? 'bg-green-100 text-green-700' :
                    p.action === 'retain' ? 'bg-yellow-100 text-yellow-700' :
                    'bg-blue-100 text-blue-700'
                  }`}>
                    {p.action.replace('_', ' ').toUpperCase()}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {tab === 'attendance' && (
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold">Attendance History</h3>
            <Link href="/academic/attendance" className="btn-secondary text-sm">Daily Roll Call</Link>
          </div>
          {attendance.length === 0 ? (
            <p className="text-sm text-gray-400">No attendance records</p>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-gray-50 text-left text-gray-500">
                  <th className="px-4 py-3 font-medium">Date</th>
                  <th className="px-4 py-3 font-medium">Status</th>
                  <th className="px-4 py-3 font-medium">Reason</th>
                  <th className="px-4 py-3 font-medium">SMS Notified</th>
                </tr>
              </thead>
              <tbody>
                {attendance.slice(0, 20).map((a) => (
                  <tr key={a.id} className="border-t">
                    <td className="px-4 py-3">{new Date(a.date).toLocaleDateString()}</td>
                    <td className="px-4 py-3">
                      <span className={`px-2 py-0.5 rounded-full text-xs ${
                        a.status === 'present' ? 'bg-green-100 text-green-700' :
                        a.status === 'absent' ? 'bg-red-100 text-red-700' :
                        a.status === 'late' ? 'bg-yellow-100 text-yellow-700' :
                        'bg-blue-100 text-blue-700'
                      }`}>
                        {a.status.toUpperCase()}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-gray-600">{a.reason || '—'}</td>
                    <td className="px-4 py-3">{a.sms_notified ? 'Yes' : 'No'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {/* Action modals */}
      {(showPromote || showRetain || showTransferOut || showTransferIn) && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="font-bold text-lg mb-4">
              {showPromote && 'Promote Learner'}
              {showRetain && 'Retain in Grade'}
              {showTransferOut && 'Transfer Out'}
              {showTransferIn && 'Transfer In'}
            </h3>
            <div className="space-y-4">
              {showPromote && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">To Grade</label>
                  <select value={toGrade} onChange={(e) => setToGrade(e.target.value)} className="w-full px-3 py-2 border rounded-md text-sm">
                    <option value="">Select grade</option>
                    {GRADES.filter((g) => g !== learner.grade).map((g) => (
                      <option key={g} value={g}>{g}</option>
                    ))}
                  </select>
                </div>
              )}
              {showTransferIn && (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">To Grade</label>
                  <select value={toGrade} onChange={(e) => setToGrade(e.target.value)} className="w-full px-3 py-2 border rounded-md text-sm">
                    <option value="">Select grade</option>
                    {GRADES.map((g) => (
                      <option key={g} value={g}>{g}</option>
                    ))}
                  </select>
                </div>
              )}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Year</label>
                <input type="number" value={actionYear} onChange={(e) => setActionYear(e.target.value)} className="w-full px-3 py-2 border rounded-md text-sm" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Notes</label>
                <textarea value={actionNotes} onChange={(e) => setActionNotes(e.target.value)} rows={3} className="w-full px-3 py-2 border rounded-md text-sm" />
              </div>
              {actionError && <div className="bg-red-50 text-red-700 p-2 rounded text-xs">{actionError}</div>}
              <div className="flex justify-end gap-2">
                <button onClick={() => { setShowPromote(false); setShowRetain(false); setShowTransferOut(false); setShowTransferIn(false); }} className="btn-secondary text-sm">Cancel</button>
                <button
                  onClick={() => {
                    const year = Number(actionYear);
                    const notes = actionNotes;
                    if (showPromote) {
                      if (!toGrade) { setActionError('To grade is required'); return; }
                      runAction(() => api.promoteLearner(id, { to_grade: toGrade, year, notes }, token));
                    } else if (showRetain) {
                      runAction(() => api.retainLearner(id, { year, notes }, token));
                    } else if (showTransferOut) {
                      runAction(() => api.transferOutLearner(id, { year, notes }, token));
                    } else if (showTransferIn) {
                      if (!toGrade) { setActionError('To grade is required'); return; }
                      runAction(() => api.transferInLearner(id, { to_grade: toGrade, year, notes }, token));
                    }
                  }}
                  className="btn-primary text-sm"
                >
                  Confirm
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}