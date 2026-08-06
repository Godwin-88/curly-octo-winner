'use client';

import { useEffect, useState } from 'react';
import { FileBarChart, Plus, Trash2, CheckCircle2 } from 'lucide-react';
import { api, DataProcessingRecord, ErasureRequest } from '@/lib/api';

export default function DataProtectionPage() {
  const token = ''; // TODO: Get from auth context
  const [records, setRecords] = useState<DataProcessingRecord[]>([]);
  const [erasure, setErasure] = useState<ErasureRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // New record form
  const [showForm, setShowForm] = useState(false);
  const [activity, setActivity] = useState('');
  const [purpose, setPurpose] = useState('');
  const [legalBasis, setLegalBasis] = useState('Consent');
  const [dataSubjects, setDataSubjects] = useState('');
  const [retention, setRetention] = useState('');
  const [thirdParties, setThirdParties] = useState('');

  // New erasure request form
  const [showErasureForm, setShowErasureForm] = useState(false);
  const [subjectType, setSubjectType] = useState('guardian');
  const [subjectId, setSubjectId] = useState('');
  const [requestedBy, setRequestedBy] = useState('');
  const [requestType, setRequestType] = useState('erasure');

  const load = async () => {
    if (!token) return;
    setLoading(true);
    setError('');
    try {
      const [r, e] = await Promise.all([
        api.listProcessingRecords(token),
        api.listErasureRequests({}, token),
      ]);
      setRecords(r);
      setErasure(e);
    } catch (e: any) {
      setError(e.message || 'Failed to load data protection records');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const createRecord = async () => {
    setError('');
    try {
      await api.createProcessingRecord({
        activity,
        purpose,
        legal_basis: legalBasis,
        data_subjects: dataSubjects,
        retention_period: retention || undefined,
        transfer_to_third_parties: !!thirdParties,
        third_parties: thirdParties || undefined,
      }, token);
      setShowForm(false);
      setActivity(''); setPurpose(''); setDataSubjects(''); setRetention(''); setThirdParties('');
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to create record');
    }
  };

  const deleteRecord = async (id: string) => {
    setError('');
    try {
      await api.deleteProcessingRecord(id, token);
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to delete record');
    }
  };

  const createErasure = async () => {
    setError('');
    try {
      await api.createErasureRequest({
        subject_type: subjectType,
        subject_id: subjectId,
        requested_by: requestedBy,
        request_type: requestType,
      }, token);
      setShowErasureForm(false);
      setSubjectId(''); setRequestedBy('');
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to create erasure request');
    }
  };

  const updateErasureStatus = async (id: string, status: string) => {
    setError('');
    try {
      await api.updateErasureRequestStatus(id, { status }, token);
      await load();
    } catch (e: any) {
      setError(e.message || 'Failed to update status');
    }
  };

  return (
    <div className="p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Data Protection</h1>
          <p className="text-gray-500">Kenya Data Protection Act 2019 — processing register & data subject rights</p>
        </div>
      </div>

      {error && <div className="mt-4 p-3 bg-red-50 text-red-700 rounded-md">{error}</div>}

      {loading ? (
        <p className="text-gray-500 mt-4">Loading...</p>
      ) : (
        <>
          {/* Data Processing Register */}
          <div className="mt-6 bg-white rounded-lg shadow border">
            <div className="p-6 pb-4 flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold flex items-center gap-2">
                  <FileBarChart size={18} className="text-purple-600" /> Data Processing Register
                </h2>
                <p className="text-sm text-gray-500">Documented processing activities, legal basis, and retention</p>
              </div>
              <button onClick={() => setShowForm(!showForm)} className="px-3 py-2 bg-purple-600 text-white rounded-md text-sm hover:bg-purple-700 flex items-center gap-2">
                <Plus size={16} /> Add Activity
              </button>
            </div>

            {showForm && (
              <div className="p-6 pt-0 grid grid-cols-1 md:grid-cols-2 gap-3">
                <input value={activity} onChange={(e) => setActivity(e.target.value)} placeholder="Activity (e.g. SMS fee reminders)" className="px-3 py-2 border rounded-md text-sm" />
                <input value={purpose} onChange={(e) => setPurpose(e.target.value)} placeholder="Purpose" className="px-3 py-2 border rounded-md text-sm" />
                <select value={legalBasis} onChange={(e) => setLegalBasis(e.target.value)} className="px-3 py-2 border rounded-md text-sm">
                  <option>Consent</option>
                  <option>Contract</option>
                  <option>Legal obligation</option>
                  <option>Legitimate interest</option>
                </select>
                <input value={dataSubjects} onChange={(e) => setDataSubjects(e.target.value)} placeholder="Data subjects (e.g. Guardians, Learners)" className="px-3 py-2 border rounded-md text-sm" />
                <input value={retention} onChange={(e) => setRetention(e.target.value)} placeholder="Retention period (e.g. 7 years)" className="px-3 py-2 border rounded-md text-sm" />
                <input value={thirdParties} onChange={(e) => setThirdParties(e.target.value)} placeholder="Third parties (e.g. Africa's Talking)" className="px-3 py-2 border rounded-md text-sm" />
                <button onClick={createRecord} className="px-4 py-2 bg-purple-600 text-white rounded-md text-sm hover:bg-purple-700 md:col-span-2">
                  Save Activity
                </button>
              </div>
            )}

            <div className="p-6 pt-0">
              {records.length === 0 ? (
                <p className="text-gray-400 text-sm">No processing activities registered.</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-left text-gray-500">
                    <tr>
                      <th className="px-3 py-2">Activity</th>
                      <th className="px-3 py-2">Legal Basis</th>
                      <th className="px-3 py-2">Data Subjects</th>
                      <th className="px-3 py-2">Retention</th>
                      <th className="px-3 py-2">Third Parties</th>
                      <th className="px-3 py-2"></th>
                    </tr>
                  </thead>
                  <tbody>
                    {records.map((r) => (
                      <tr key={r.id} className="border-t">
                        <td className="px-3 py-2">
                          <p className="font-medium">{r.activity}</p>
                          <p className="text-xs text-gray-400">{r.purpose}</p>
                        </td>
                        <td className="px-3 py-2">{r.legal_basis}</td>
                        <td className="px-3 py-2">{r.data_subjects}</td>
                        <td className="px-3 py-2">{r.retention_period || '-'}</td>
                        <td className="px-3 py-2">{r.third_parties || '-'}</td>
                        <td className="px-3 py-2">
                          <button onClick={() => deleteRecord(r.id)} className="text-red-500 hover:text-red-700">
                            <Trash2 size={16} />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>

          {/* Erasure / Data Subject Rights */}
          <div className="mt-6 bg-white rounded-lg shadow border">
            <div className="p-6 pb-4 flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold flex items-center gap-2">
                  <CheckCircle2 size={18} className="text-green-600" /> Data Subject Rights
                </h2>
                <p className="text-sm text-gray-500">Erasure, access, rectification, and restriction requests</p>
              </div>
              <button onClick={() => setShowErasureForm(!showErasureForm)} className="px-3 py-2 bg-green-600 text-white rounded-md text-sm hover:bg-green-700 flex items-center gap-2">
                <Plus size={16} /> New Request
              </button>
            </div>

            {showErasureForm && (
              <div className="p-6 pt-0 grid grid-cols-1 md:grid-cols-2 gap-3">
                <select value={subjectType} onChange={(e) => setSubjectType(e.target.value)} className="px-3 py-2 border rounded-md text-sm">
                  <option value="guardian">Guardian</option>
                  <option value="learner">Learner</option>
                </select>
                <input value={subjectId} onChange={(e) => setSubjectId(e.target.value)} placeholder="Subject ID (UUID)" className="px-3 py-2 border rounded-md text-sm" />
                <input value={requestedBy} onChange={(e) => setRequestedBy(e.target.value)} placeholder="Requested by (name/contact)" className="px-3 py-2 border rounded-md text-sm" />
                <select value={requestType} onChange={(e) => setRequestType(e.target.value)} className="px-3 py-2 border rounded-md text-sm">
                  <option value="erasure">Erasure</option>
                  <option value="access">Access</option>
                  <option value="rectification">Rectification</option>
                  <option value="restriction">Restriction</option>
                </select>
                <button onClick={createErasure} className="px-4 py-2 bg-green-600 text-white rounded-md text-sm hover:bg-green-700 md:col-span-2">
                  Submit Request
                </button>
              </div>
            )}

            <div className="p-6 pt-0">
              {erasure.length === 0 ? (
                <p className="text-gray-400 text-sm">No data subject requests.</p>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 text-left text-gray-500">
                    <tr>
                      <th className="px-3 py-2">Subject</th>
                      <th className="px-3 py-2">Type</th>
                      <th className="px-3 py-2">Requested By</th>
                      <th className="px-3 py-2">Status</th>
                      <th className="px-3 py-2">Created</th>
                      <th className="px-3 py-2">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {erasure.map((e) => (
                      <tr key={e.id} className="border-t">
                        <td className="px-3 py-2 font-mono text-xs">{e.subject_id.slice(0, 8)}</td>
                        <td className="px-3 py-2 capitalize">{e.request_type}</td>
                        <td className="px-3 py-2">{e.requested_by}</td>
                        <td className="px-3 py-2">
                          <span className={`px-2 py-1 rounded-full text-xs ${
                            e.status === 'completed' ? 'bg-green-50 text-green-700' :
                            e.status === 'denied' ? 'bg-red-50 text-red-700' :
                            'bg-yellow-50 text-yellow-700'
                          }`}>
                            {e.status}
                          </span>
                        </td>
                        <td className="px-3 py-2 whitespace-nowrap">{new Date(e.created_at).toLocaleDateString()}</td>
                        <td className="px-3 py-2">
                          {e.status === 'pending' && (
                            <div className="flex gap-2">
                              <button onClick={() => updateErasureStatus(e.id, 'in_progress')} className="px-2 py-1 bg-blue-50 text-blue-700 rounded-md text-xs hover:bg-blue-100">
                                Start
                              </button>
                              <button onClick={() => updateErasureStatus(e.id, 'completed')} className="px-2 py-1 bg-green-50 text-green-700 rounded-md text-xs hover:bg-green-100">
                                Complete
                              </button>
                            </div>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}