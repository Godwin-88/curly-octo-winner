// Typed fetch wrapper for the Shule360 Go API

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export class APIError extends Error {
  code: string;
  status: number;

  constructor(message: string, code: string, status: number) {
    super(message);
    this.code = code;
    this.status = status;
  }
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  token?: string;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, token } = options;

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Unknown error', code: 'UNKNOWN' }));
    throw new APIError(errorData.error || 'Request failed', errorData.code || 'UNKNOWN', res.status);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}

// --- Message types ---

export interface Message {
  id: string;
  tenant_id: string;
  channel: 'sms' | 'whatsapp' | 'both';
  audience_type: string;
  audience_filter: Record<string, unknown>;
  content_type: 'text' | 'template' | 'media';
  content: string;
  template_id?: string;
  media_url?: string;
  status: 'draft' | 'scheduled' | 'sending' | 'sent' | 'failed';
  scheduled_at?: string;
  sent_at?: string;
  sent_by: string;
  recipient_count: number;
  delivered_count: number;
  failed_count: number;
  created_at: string;
}

export interface CreateMessageRequest {
  channel: 'sms' | 'whatsapp' | 'both';
  audience_type: string;
  audience_filter: Record<string, unknown>;
  content_type: 'text' | 'template' | 'media';
  content: string;
  template_id?: string;
  media_url?: string;
  scheduled_at?: string;
}

export interface ReachEstimate {
  recipient_count: number;
  estimated_kes: number;
  sms_units: number;
}

export interface DeliveryStats {
  total: number;
  sent: number;
  delivered: number;
  failed: number;
  pending: number;
  delivery_rate: number;
}

export interface MessageLog {
  id: string;
  message_id: string;
  recipient_type: string;
  recipient_id?: string;
  phone: string;
  channel: string;
  status: string;
  provider_message_id?: string;
  delivered_at?: string;
  read_at?: string;
  error_code?: string;
  error_message?: string;
  created_at: string;
}

export interface Conversation {
  id: string;
  tenant_id: string;
  guardian_id?: string;
  wa_contact_phone: string;
  wa_contact_name: string;
  status: 'open' | 'in_progress' | 'waiting' | 'resolved';
  assigned_to?: string;
  last_message_at: string;
  last_message_preview: string;
  unread_count: number;
  created_at: string;
}

export interface ConversationMessage {
  id: string;
  conversation_id: string;
  tenant_id: string;
  direction: 'inbound' | 'outbound';
  content_type: string;
  content: Record<string, unknown>;
  wa_message_id: string;
  status: string;
  timestamp: string;
}

// --- Academic types ---

export interface LearningArea {
  id: string;
  tenant_id: string;
  name: string;
  kicd_code: string;
  grade_level: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface Strand {
  id: string;
  tenant_id: string;
  learning_area_id: string;
  name: string;
  kicd_code: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface SubStrand {
  id: string;
  tenant_id: string;
  strand_id: string;
  name: string;
  kicd_code: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface LearningOutcome {
  id: string;
  tenant_id: string;
  sub_strand_id: string;
  description: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface CoreCompetency {
  id: string;
  tenant_id: string;
  name: string;
  kicd_code: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface Value {
  id: string;
  tenant_id: string;
  name: string;
  kicd_code: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface Assessment {
  id: string;
  tenant_id: string;
  learner_id: string;
  sub_strand_id: string;
  rubric_level: number;
  note: string;
  evidence_urls: string[];
  teacher_id: string;
  term: number;
  year: number;
  created_at: string;
  updated_at: string;
}

export interface CreateAssessmentRequest {
  learner_id: string;
  sub_strand_id: string;
  rubric_level: number;
  note?: string;
  evidence_urls?: string[];
  term: number;
  year: number;
}

export interface AssessmentSummary {
  id: string;
  learner_id: string;
  learner_name: string;
  grade: string;
  stream: string;
  sub_strand_id: string;
  sub_strand_name: string;
  sub_strand_code: string;
  strand_name: string;
  learning_area: string;
  rubric_level: number;
  rubric_label: string;
  note: string;
  term: number;
  year: number;
  teacher_id: string;
  created_at: string;
}

export interface AttendanceRecord {
  id: string;
  tenant_id: string;
  learner_id: string;
  date: string;
  status: 'present' | 'absent' | 'late' | 'excused';
  marked_by?: string;
  reason?: string;
  sms_notified: boolean;
  created_at: string;
  updated_at: string;
}

export interface AttendanceSummary {
  id: string;
  learner_id: string;
  learner_name: string;
  grade: string;
  stream: string;
  date: string;
  status: 'present' | 'absent' | 'late' | 'excused';
  reason?: string;
  sms_notified: boolean;
  created_at: string;
}

// --- API functions ---

export const api = {
  // Messages
  createMessage: (data: CreateMessageRequest, token: string) =>
    request<Message>('/messages', { method: 'POST', body: data, token }),

  listMessages: (params: { status?: string; channel?: string; limit?: number; offset?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.status) qs.set('status', params.status);
    if (params.channel) qs.set('channel', params.channel);
    if (params.limit) qs.set('limit', String(params.limit));
    if (params.offset) qs.set('offset', String(params.offset));
    return request<Message[]>(`/messages?${qs.toString()}`, { token });
  },

  getMessage: (id: string, token: string) =>
    request<{ message: Message; stats: DeliveryStats }>(`/messages/${id}`, { token }),

  getMessageLogs: (id: string, params: { limit?: number; offset?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.limit) qs.set('limit', String(params.limit));
    if (params.offset) qs.set('offset', String(params.offset));
    return request<MessageLog[]>(`/messages/${id}/logs?${qs.toString()}`, { token });
  },

  cancelMessage: (id: string, token: string) =>
    request<void>(`/messages/${id}`, { method: 'DELETE', token }),

  estimateReach: (data: CreateMessageRequest, token: string) =>
    request<ReachEstimate>('/messages/estimate', { method: 'POST', body: data, token }),

  // Conversations
  listConversations: (params: { status?: string; assigned_to?: string; limit?: number; offset?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.status) qs.set('status', params.status);
    if (params.assigned_to) qs.set('assigned_to', params.assigned_to);
    if (params.limit) qs.set('limit', String(params.limit));
    if (params.offset) qs.set('offset', String(params.offset));
    return request<Conversation[]>(`/conversations?${qs.toString()}`, { token });
  },

  getConversation: (id: string, token: string) =>
    request<{ conversation: Conversation; messages: ConversationMessage[] }>(`/conversations/${id}`, { token }),

  sendReply: (id: string, data: { content: string; content_type?: string }, token: string) =>
    request<{ message_id: string }>(`/conversations/${id}/reply`, { method: 'POST', body: data, token }),

  assignConversation: (id: string, staffId: string, token: string) =>
    request<void>(`/conversations/${id}/assign`, { method: 'PATCH', body: { staff_id: staffId }, token }),

  updateConversationStatus: (id: string, status: string, token: string) =>
    request<void>(`/conversations/${id}/status`, { method: 'PATCH', body: { status }, token }),

  // Curriculum
  listLearningAreas: (token: string) =>
    request<LearningArea[]>('/curriculum/learning-areas', { token }),

  createLearningArea: (data: LearningArea, token: string) =>
    request<LearningArea>('/curriculum/learning-areas', { method: 'POST', body: data, token }),

  listStrands: (learningAreaId: string, token: string) =>
    request<Strand[]>(`/curriculum/learning-areas/${learningAreaId}/strands`, { token }),

  createStrand: (data: Strand, token: string) =>
    request<Strand>('/curriculum/strands', { method: 'POST', body: data, token }),

  listSubStrands: (strandId: string, token: string) =>
    request<SubStrand[]>(`/curriculum/strands/${strandId}/sub-strands`, { token }),

  createSubStrand: (data: SubStrand, token: string) =>
    request<SubStrand>('/curriculum/sub-strands', { method: 'POST', body: data, token }),

  listLearningOutcomes: (subStrandId: string, token: string) =>
    request<LearningOutcome[]>(`/curriculum/sub-strands/${subStrandId}/learning-outcomes`, { token }),

  createLearningOutcome: (data: LearningOutcome, token: string) =>
    request<LearningOutcome>('/curriculum/learning-outcomes', { method: 'POST', body: data, token }),

  listCoreCompetencies: (token: string) =>
    request<CoreCompetency[]>('/curriculum/core-competencies', { token }),

  createCoreCompetency: (data: CoreCompetency, token: string) =>
    request<CoreCompetency>('/curriculum/core-competencies', { method: 'POST', body: data, token }),

  listValues: (token: string) =>
    request<Value[]>('/curriculum/values', { token }),

  createValue: (data: Value, token: string) =>
    request<Value>('/curriculum/values', { method: 'POST', body: data, token }),

  // Assessments
  createAssessment: (data: CreateAssessmentRequest, token: string) =>
    request<Assessment>('/assessments', { method: 'POST', body: data, token }),

  getAssessment: (id: string, token: string) =>
    request<Assessment>(`/assessments/${id}`, { token }),

  listAssessmentsByLearner: (learnerId: string, params: { term?: number; year?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.term) qs.set('term', String(params.term));
    if (params.year) qs.set('year', String(params.year));
    return request<Assessment[]>(`/assessments/learner/${learnerId}?${qs.toString()}`, { token });
  },

  listTermSummaries: (params: { learner_id: string; term?: number; year?: number }, token: string) => {
    const qs = new URLSearchParams();
    qs.set('learner_id', params.learner_id);
    if (params.term) qs.set('term', String(params.term));
    if (params.year) qs.set('year', String(params.year));
    return request<AssessmentSummary[]>(`/assessments/term-summary?${qs.toString()}`, { token });
  },

  deleteAssessment: (id: string, token: string) =>
    request<void>(`/assessments/${id}`, { method: 'DELETE', token }),

  // Attendance
  markAttendance: (data: AttendanceRecord, token: string) =>
    request<AttendanceRecord>('/attendance', { method: 'POST', body: data, token }),

  listAttendanceByDate: (date: string, token: string) =>
    request<AttendanceSummary[]>(`/attendance/date?date=${date}`, { token }),

  listAttendanceByLearner: (learnerId: string, token: string) =>
    request<AttendanceRecord[]>(`/attendance/learner/${learnerId}`, { token }),

  getAttendance: (id: string, token: string) =>
    request<AttendanceRecord>(`/attendance/${id}`, { token }),

  deleteAttendance: (id: string, token: string) =>
    request<void>(`/attendance/${id}`, { method: 'DELETE', token }),
};