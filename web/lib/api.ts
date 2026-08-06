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

// --- Learner types ---

export interface Learner {
  id: string;
  tenant_id: string;
  upi: string;
  full_name: string;
  date_of_birth?: string;
  grade: string;
  stream?: string;
  photo_url?: string;
  guardian_ids: string[];
  birth_cert_no?: string;
  entry_level?: string;
  special_needs: boolean;
  is_active: boolean;
  admission_date?: string;
  created_at: string;
  updated_at: string;
}

export interface GuardianBrief {
  id: string;
  full_name: string;
  phone: string;
  relation: string;
}

export interface CreateLearnerRequest {
  upi: string;
  full_name: string;
  date_of_birth?: string;
  grade: string;
  stream?: string;
  photo_url?: string;
  guardian_ids?: string[];
  birth_cert_no?: string;
  entry_level?: string;
  special_needs?: boolean;
  admission_date?: string;
}

export interface UpdateLearnerRequest {
  full_name?: string;
  date_of_birth?: string;
  grade?: string;
  stream?: string;
  photo_url?: string;
  guardian_ids?: string[];
  birth_cert_no?: string;
  entry_level?: string;
  special_needs?: boolean;
  admission_date?: string;
}

export interface LearnerDocument {
  id: string;
  tenant_id: string;
  learner_id: string;
  doc_type: string;
  file_name: string;
  file_url: string;
  mime_type?: string;
  file_size?: number;
  uploaded_by?: string;
  created_at: string;
  updated_at: string;
}

export interface LearnerProgression {
  id: string;
  tenant_id: string;
  learner_id: string;
  from_grade: string;
  to_grade: string;
  action: string;
  term?: number;
  year: number;
  approved_by?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

// --- Transport types ---

export interface Vehicle {
  id: string;
  tenant_id: string;
  registration: string;
  make: string;
  model: string;
  capacity: number;
  year?: number;
  status: 'active' | 'maintenance' | 'retired';
  insurance_expiry?: string;
  inspection_expiry?: string;
  driver_id?: string;
  driver_name?: string;
  driver_phone?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateVehicleRequest {
  registration: string;
  make: string;
  model: string;
  capacity: number;
  year?: number;
  status?: string;
  insurance_expiry?: string;
  inspection_expiry?: string;
  driver_id?: string;
  driver_name?: string;
  driver_phone?: string;
  notes?: string;
}

export interface UpdateVehicleRequest {
  registration?: string;
  make?: string;
  model?: string;
  capacity?: number;
  year?: number;
  status?: string;
  insurance_expiry?: string;
  inspection_expiry?: string;
  driver_id?: string;
  driver_name?: string;
  driver_phone?: string;
  notes?: string;
}

export interface Stop {
  id: string;
  tenant_id: string;
  route_id: string;
  name: string;
  sequence: number;
  latitude?: number;
  longitude?: number;
  landmark?: string;
  created_at: string;
}

export interface Route {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  vehicle_id?: string;
  active: boolean;
  created_at: string;
  updated_at: string;
  stops?: Stop[];
}

export interface CreateRouteRequest {
  name: string;
  description?: string;
  vehicle_id?: string;
  active?: boolean;
  stops?: Stop[];
}

export interface UpdateRouteRequest {
  name?: string;
  description?: string;
  vehicle_id?: string;
  active?: boolean;
}

export interface StopInput {
  name: string;
  sequence: number;
  latitude?: number;
  longitude?: number;
  landmark?: string;
}

export interface Assignment {
  id: string;
  tenant_id: string;
  route_id: string;
  learner_id: string;
  stop_id: string;
  direction: string;
  created_at: string;
  learner_name?: string;
  grade?: string;
  stream?: string;
  stop_name?: string;
}

export interface CreateAssignmentRequest {
  learner_id: string;
  stop_id: string;
  direction: string;
}

export interface Trip {
  id: string;
  tenant_id: string;
  route_id: string;
  route_name?: string;
  vehicle_id?: string;
  vehicle_registration?: string;
  direction: string;
  status: 'scheduled' | 'in_progress' | 'completed' | 'cancelled';
  scheduled_departure: string;
  actual_departure?: string;
  actual_arrival?: string;
  boarded_count: number;
  created_by?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
  last_latitude?: number;
  last_longitude?: number;
  last_reported?: string;
}

export interface CreateTripRequest {
  route_id: string;
  vehicle_id?: string;
  direction: string;
  scheduled_departure: string;
  notes?: string;
}

export interface UpdateTripRequest {
  route_id?: string;
  vehicle_id?: string;
  direction?: string;
  scheduled_departure?: string;
  notes?: string;
}

export interface TripCheckin {
  id: string;
  trip_id: string;
  learner_id: string;
  learner_name?: string;
  stop_id?: string;
  stop_name?: string;
  action: string;
  checked_at: string;
  sms_notified: boolean;
}

export interface CreateCheckinRequest {
  learner_id: string;
  stop_id?: string;
  action: string;
}

export interface TripPosition {
  id: string;
  trip_id: string;
  latitude: number;
  longitude: number;
  speed_kmh?: number;
  heading_deg?: number;
  odometer_km?: number;
  reported_at: string;
}

export interface ReportPositionRequest {
  latitude: number;
  longitude: number;
  speed_kmh?: number;
  heading_deg?: number;
  odometer_km?: number;
}

// --- Finance types ---

export interface FeeStructureItem {
  id: string;
  tenant_id: string;
  fee_structure_id: string;
  name: string;
  amount_cents: number;
  item_type: string;
  is_optional: boolean;
  sort_order: number;
  created_at: string;
}

export interface FeeStructure {
  id: string;
  tenant_id: string;
  name: string;
  grade: string;
  term: number;
  year: number;
  total_cents: number;
  active: boolean;
  notes?: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
  items?: FeeStructureItem[];
}

export interface FeeItemInput {
  name: string;
  amount_cents: number;
  item_type: string;
  is_optional?: boolean;
  sort_order?: number;
}

export interface CreateFeeStructureRequest {
  name: string;
  grade: string;
  term: number;
  year: number;
  active?: boolean;
  notes?: string;
  created_by?: string;
  items?: FeeItemInput[];
}

export interface UpdateFeeStructureRequest {
  name?: string;
  grade?: string;
  term?: number;
  year?: number;
  active?: boolean;
  notes?: string;
}

export interface InvoiceItem {
  id: string;
  tenant_id: string;
  invoice_id: string;
  name: string;
  amount_cents: number;
  item_type: string;
  is_optional: boolean;
  sort_order: number;
  created_at: string;
}

export interface Invoice {
  id: string;
  tenant_id: string;
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
  status: 'draft' | 'unpaid' | 'partially_paid' | 'paid' | 'overdue' | 'void';
  notes?: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
  learner_name?: string;
  grade?: string;
  stream?: string;
  balance_cents?: number;
  items?: InvoiceItem[];
}

export interface CreateInvoiceRequest {
  learner_id: string;
  fee_structure_id?: string;
  term: number;
  year: number;
  issue_date?: string;
  due_date?: string;
  notes?: string;
  created_by?: string;
  items?: FeeItemInput[];
}

export interface UpdateInvoiceRequest {
  due_date?: string;
  status?: string;
  notes?: string;
}

export interface Discount {
  id: string;
  tenant_id: string;
  invoice_id: string;
  amount_cents: number;
  discount_type: string;
  reason?: string;
  approved_by?: string;
  created_at: string;
}

export interface CreateDiscountRequest {
  amount_cents: number;
  discount_type: string;
  reason?: string;
  approved_by?: string;
}

export interface Payment {
  id: string;
  tenant_id: string;
  invoice_id: string;
  amount_cents: number;
  channel: 'mpesa' | 'bank' | 'cash' | 'cheque';
  status: 'pending' | 'completed' | 'failed' | 'reversed';
  reference?: string;
  paid_by?: string;
  phone?: string;
  paid_at?: string;
  received_by?: string;
  notes?: string;
  checkout_request_id?: string;
  merchant_request_id?: string;
  mpesa_receipt?: string;
  mpesa_result_code?: string;
  mpesa_result_desc?: string;
  created_at: string;
  updated_at: string;
  invoice_number?: string;
  learner_name?: string;
  grade?: string;
}

export interface CreatePaymentRequest {
  invoice_id: string;
  amount_cents: number;
  channel: string;
  reference?: string;
  paid_by?: string;
  phone?: string;
  paid_at?: string;
  received_by?: string;
  notes?: string;
}

export interface MpesaStkRequest {
  invoice_id: string;
  phone: string;
  amount_cents: number;
  paid_by?: string;
}

// --- Reports & Analytics types ---

export interface ReportCardItem {
  id: string;
  tenant_id: string;
  report_card_id: string;
  learning_area_id?: string;
  strand_id?: string;
  sub_strand_id?: string;
  learning_area?: string;
  strand_name?: string;
  sub_strand_name?: string;
  rubric_level?: number;
  rubric_label?: string;
  comment?: string;
  sort_order: number;
  created_at: string;
}

export interface ReportCard {
  id: string;
  tenant_id: string;
  learner_id: string;
  learner_name?: string;
  grade?: string;
  stream?: string;
  upi?: string;
  term: number;
  year: number;
  status: 'draft' | 'final';
  overall_rating?: number;
  core_competency_remarks?: Record<string, string>;
  teacher_comments?: Record<string, string>;
  attendance_summary?: Record<string, any>;
  generated_by?: string;
  generated_at: string;
  created_at: string;
  updated_at: string;
  items?: ReportCardItem[];
}

export interface GenerateReportCardRequest {
  status?: string;
  overall_rating?: number;
  core_competency_remarks?: Record<string, string>;
  teacher_comments?: Record<string, string>;
  generated_by?: string;
}

export interface UpdateReportCardRequest {
  status?: string;
  overall_rating?: number;
  core_competency_remarks?: Record<string, string>;
  teacher_comments?: Record<string, string>;
}

export interface SchoolOverview {
  tenant_id: string;
  learner_count: number;
}

export interface AlertLearner {
  learner_id: string;
  learner_name: string;
  grade: string;
  stream: string;
  term: number;
  year: number;
  overall_avg_rubric: number;
  attendance_rate: number;
  assessed_areas: number;
}

export interface StrandCoverage {
  tenant_id: string;
  grade: string;
  stream: string;
  learning_area_id: string;
  learning_area: string;
  strand_id: string;
  strand_name: string;
  term: number;
  year: number;
  sub_strands_assessed: number;
  learners_assessed: number;
}

export interface CompetencyDistribution {
  tenant_id: string;
  grade: string;
  stream: string;
  strand_id: string;
  strand_name: string;
  term: number;
  year: number;
  rubric_level: number;
  learner_count: number;
}

export interface TeacherVelocity {
  tenant_id: string;
  teacher_id: string;
  teacher_name: string;
  term: number;
  year: number;
  week_start: string;
  assessment_count: number;
}

export interface LearnerPortfolio {
  tenant_id: string;
  learner_id: string;
  learner_name: string;
  grade: string;
  stream: string;
  term: number;
  year: number;
  learning_areas_assessed: number;
  overall_avg_rubric: number;
  attendance_rate: number;
}

export interface LearningAreaPerformance {
  tenant_id: string;
  learner_id: string;
  term: number;
  year: number;
  learning_area_id: string;
  learning_area: string;
  assessment_count: number;
  avg_rubric_level: number;
}

// --- HR types ---

export interface StaffProfile {
  id: string;
  tenant_id: string;
  full_name: string;
  email: string;
  phone?: string;
  role: string;
  is_active: boolean;
  tsc_number?: string;
  national_id?: string;
  kra_pin?: string;
  date_of_birth?: string;
  gender?: string;
  department?: string;
  job_title?: string;
  employment_type: string;
  hire_date?: string;
  qualifications?: any[];
  subjects?: any[];
  employment_history?: any[];
  emergency_contact?: Record<string, any>;
  bank_details?: Record<string, any>;
  photo_url?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateStaffRequest {
  full_name: string;
  email: string;
  phone?: string;
  role?: string;
  is_active?: boolean;
  tsc_number?: string;
  national_id?: string;
  kra_pin?: string;
  date_of_birth?: string;
  gender?: string;
  department?: string;
  job_title?: string;
  employment_type?: string;
  hire_date?: string;
  qualifications?: any[];
  subjects?: any[];
  employment_history?: any[];
  emergency_contact?: Record<string, any>;
  bank_details?: Record<string, any>;
  photo_url?: string;
}

export interface UpdateStaffRequest {
  full_name?: string;
  email?: string;
  phone?: string;
  role?: string;
  is_active?: boolean;
  tsc_number?: string;
  national_id?: string;
  kra_pin?: string;
  date_of_birth?: string;
  gender?: string;
  department?: string;
  job_title?: string;
  employment_type?: string;
  hire_date?: string;
  qualifications?: any[];
  subjects?: any[];
  employment_history?: any[];
  emergency_contact?: Record<string, any>;
  bank_details?: Record<string, any>;
  photo_url?: string;
}

export interface StaffDocument {
  id: string;
  tenant_id: string;
  staff_id: string;
  doc_type: string;
  file_name: string;
  file_url: string;
  mime_type?: string;
  file_size?: number;
  uploaded_by?: string;
  created_at: string;
}

export interface CreateStaffDocumentRequest {
  doc_type: string;
  file_name: string;
  file_url: string;
  mime_type?: string;
  file_size?: number;
  uploaded_by?: string;
}

export interface PayrollItem {
  id: string;
  tenant_id: string;
  payroll_run_id: string;
  item_type: 'earning' | 'deduction';
  name: string;
  amount_cents: number;
  sort_order: number;
  created_at: string;
}

export interface PayrollRun {
  id: string;
  tenant_id: string;
  staff_id: string;
  staff_name?: string;
  month: number;
  year: number;
  basic_salary_cents: number;
  allowances_cents: number;
  gross_cents: number;
  paye_cents: number;
  nhif_cents: number;
  nssf_cents: number;
  other_deductions_cents: number;
  net_cents: number;
  status: 'draft' | 'approved' | 'paid';
  paid_at?: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
  items?: PayrollItem[];
}

export interface PayrollItemInput {
  item_type: string;
  name: string;
  amount_cents: number;
  sort_order?: number;
}

export interface CreatePayrollRunRequest {
  staff_id: string;
  month: number;
  year: number;
  basic_salary_cents: number;
  allowances_cents: number;
  paye_cents: number;
  nhif_cents: number;
  nssf_cents: number;
  other_deductions_cents: number;
  created_by?: string;
  items?: PayrollItemInput[];
}

export interface UpdatePayrollRunRequest {
  basic_salary_cents?: number;
  allowances_cents?: number;
  paye_cents?: number;
  nhif_cents?: number;
  nssf_cents?: number;
  other_deductions_cents?: number;
  status?: string;
}

export interface LeaveRequest {
  id: string;
  tenant_id: string;
  staff_id: string;
  staff_name?: string;
  leave_type: string;
  start_date: string;
  end_date: string;
  days: number;
  reason?: string;
  status: 'pending' | 'approved' | 'denied' | 'cancelled';
  approved_by?: string;
  approved_at?: string;
  denial_reason?: string;
  substitute_id?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateLeaveRequest {
  staff_id: string;
  leave_type: string;
  start_date: string;
  end_date: string;
  reason?: string;
  substitute_id?: string;
}

export interface ApproveLeaveRequest {
  approved_by?: string;
  substitute_id?: string;
}

export interface DenyLeaveRequest {
  approved_by?: string;
  denial_reason?: string;
}

export interface StaffAttendance {
  id: string;
  tenant_id: string;
  staff_id: string;
  staff_name?: string;
  date: string;
  clock_in?: string;
  clock_out?: string;
  status: string;
  notes?: string;
  marked_by?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateStaffAttendanceRequest {
  staff_id: string;
  date: string;
  clock_in?: string;
  clock_out?: string;
  status?: string;
  notes?: string;
  marked_by?: string;
}

export interface UpdateStaffAttendanceRequest {
  clock_in?: string;
  clock_out?: string;
  status?: string;
  notes?: string;
}

export interface StaffAppraisal {
  id: string;
  tenant_id: string;
  staff_id: string;
  staff_name?: string;
  year: number;
  term?: number;
  appraiser_id?: string;
  scores?: Record<string, any>;
  overall_score?: number;
  rating?: string;
  comments?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface CreateAppraisalRequest {
  staff_id: string;
  year: number;
  term?: number;
  appraiser_id?: string;
  scores?: Record<string, any>;
  comments?: string;
}

export interface UpdateAppraisalRequest {
  scores?: Record<string, any>;
  overall_score?: number;
  rating?: string;
  comments?: string;
  status?: string;
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

  // Learners
  listLearners: (params: { grade?: string; stream?: string; search?: string; include_inactive?: boolean }, token: string) => {
    const qs = new URLSearchParams();
    if (params.grade) qs.set('grade', params.grade);
    if (params.stream) qs.set('stream', params.stream);
    if (params.search) qs.set('search', params.search);
    if (params.include_inactive) qs.set('include_inactive', 'true');
    return request<Learner[]>(`/learners?${qs.toString()}`, { token });
  },

  createLearner: (data: CreateLearnerRequest, token: string) =>
    request<Learner>('/learners', { method: 'POST', body: data, token }),

  getLearner: (id: string, token: string) =>
    request<Learner>(`/learners/${id}`, { token }),

  updateLearner: (id: string, data: UpdateLearnerRequest, token: string) =>
    request<Learner>(`/learners/${id}`, { method: 'PATCH', body: data, token }),

  deactivateLearner: (id: string, token: string) =>
    request<void>(`/learners/${id}`, { method: 'DELETE', token }),

  reactivateLearner: (id: string, token: string) =>
    request<Learner>(`/learners/${id}/reactivate`, { method: 'POST', token }),

  listLearnerGuardians: (id: string, token: string) =>
    request<GuardianBrief[]>(`/learners/${id}/guardians`, { token }),

  listLearnerProgressions: (id: string, token: string) =>
    request<LearnerProgression[]>(`/learners/${id}/progressions`, { token }),

  // Learner documents
  listLearnerDocuments: (id: string, token: string) =>
    request<LearnerDocument[]>(`/learners/${id}/documents`, { token }),

  uploadLearnerDocument: (id: string, data: { doc_type: string; file_name: string; file_url: string; mime_type?: string; file_size?: number }, token: string) =>
    request<LearnerDocument>(`/learners/${id}/documents`, { method: 'POST', body: data, token }),

  deleteLearnerDocument: (docId: string, token: string) =>
    request<void>(`/learners/documents/${docId}`, { method: 'DELETE', token }),

  // Learner progression
  promoteLearner: (id: string, data: { to_grade: string; term?: number; year?: number; notes?: string }, token: string) =>
    request<LearnerProgression>(`/learners/${id}/promote`, { method: 'POST', body: data, token }),

  retainLearner: (id: string, data: { term?: number; year?: number; notes?: string }, token: string) =>
    request<LearnerProgression>(`/learners/${id}/retain`, { method: 'POST', body: data, token }),

  transferOutLearner: (id: string, data: { term?: number; year?: number; notes?: string }, token: string) =>
    request<LearnerProgression>(`/learners/${id}/transfer-out`, { method: 'POST', body: data, token }),

  transferInLearner: (id: string, data: { to_grade: string; term?: number; year?: number; notes?: string }, token: string) =>
    request<LearnerProgression>(`/learners/${id}/transfer-in`, { method: 'POST', body: data, token }),

  // Vehicles
  listVehicles: (params: { status?: string }, token: string) => {
    const qs = new URLSearchParams();
    if (params.status) qs.set('status', params.status);
    return request<Vehicle[]>(`/vehicles?${qs.toString()}`, { token });
  },

  createVehicle: (data: CreateVehicleRequest, token: string) =>
    request<Vehicle>('/vehicles', { method: 'POST', body: data, token }),

  getVehicle: (id: string, token: string) =>
    request<Vehicle>(`/vehicles/${id}`, { token }),

  updateVehicle: (id: string, data: UpdateVehicleRequest, token: string) =>
    request<Vehicle>(`/vehicles/${id}`, { method: 'PATCH', body: data, token }),

  deleteVehicle: (id: string, token: string) =>
    request<void>(`/vehicles/${id}`, { method: 'DELETE', token }),

  // Routes
  listRoutes: (token: string) =>
    request<Route[]>('/routes', { token }),

  createRoute: (data: CreateRouteRequest, token: string) =>
    request<Route>('/routes', { method: 'POST', body: data, token }),

  getRoute: (id: string, token: string) =>
    request<Route>(`/routes/${id}`, { token }),

  updateRoute: (id: string, data: UpdateRouteRequest, token: string) =>
    request<Route>(`/routes/${id}`, { method: 'PATCH', body: data, token }),

  deleteRoute: (id: string, token: string) =>
    request<void>(`/routes/${id}`, { method: 'DELETE', token }),

  // Stops
  listRouteStops: (routeId: string, token: string) =>
    request<Stop[]>(`/routes/${routeId}/stops`, { token }),

  createRouteStop: (routeId: string, data: StopInput, token: string) =>
    request<Stop>(`/routes/${routeId}/stops`, { method: 'POST', body: data, token }),

  deleteRouteStop: (stopId: string, token: string) =>
    request<void>(`/stops/${stopId}`, { method: 'DELETE', token }),

  // Assignments
  listRouteAssignments: (routeId: string, token: string) =>
    request<Assignment[]>(`/routes/${routeId}/assignments`, { token }),

  assignLearnerToRoute: (routeId: string, data: CreateAssignmentRequest, token: string) =>
    request<Assignment>(`/routes/${routeId}/assignments`, { method: 'POST', body: data, token }),

  removeRouteAssignment: (assignmentId: string, token: string) =>
    request<void>(`/assignments/${assignmentId}`, { method: 'DELETE', token }),

  // Trips
  listTrips: (params: { status?: string; on_date?: string }, token: string) => {
    const qs = new URLSearchParams();
    if (params.status) qs.set('status', params.status);
    if (params.on_date) qs.set('on_date', params.on_date);
    return request<Trip[]>(`/trips?${qs.toString()}`, { token });
  },

  createTrip: (data: CreateTripRequest, token: string) =>
    request<Trip>('/trips', { method: 'POST', body: data, token }),

  getTrip: (id: string, token: string) =>
    request<Trip>(`/trips/${id}`, { token }),

  updateTrip: (id: string, data: UpdateTripRequest, token: string) =>
    request<Trip>(`/trips/${id}`, { method: 'PATCH', body: data, token }),

  deleteTrip: (id: string, token: string) =>
    request<void>(`/trips/${id}`, { method: 'DELETE', token }),

  startTrip: (id: string, token: string) =>
    request<Trip>(`/trips/${id}/start`, { method: 'POST', token }),

  completeTrip: (id: string, token: string) =>
    request<Trip>(`/trips/${id}/complete`, { method: 'POST', token }),

  cancelTrip: (id: string, token: string) =>
    request<Trip>(`/trips/${id}/cancel`, { method: 'POST', token }),

  // Tracking
  listTripPositions: (id: string, params: { limit?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.limit) qs.set('limit', String(params.limit));
    return request<TripPosition[]>(`/trips/${id}/positions?${qs.toString()}`, { token });
  },

  reportTripPosition: (id: string, data: ReportPositionRequest, token: string) =>
    request<TripPosition>(`/trips/${id}/positions`, { method: 'POST', body: data, token }),

  // Check-ins
  listTripCheckins: (id: string, token: string) =>
    request<TripCheckin[]>(`/trips/${id}/checkins`, { token }),

  checkInLearner: (id: string, data: CreateCheckinRequest, token: string) =>
    request<TripCheckin>(`/trips/${id}/checkins`, { method: 'POST', body: data, token }),

  // Fee structures
  listFeeStructures: (params: { grade?: string; term?: number; year?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.grade) qs.set('grade', params.grade);
    if (params.term) qs.set('term', String(params.term));
    if (params.year) qs.set('year', String(params.year));
    return request<FeeStructure[]>(`/fee-structures?${qs.toString()}`, { token });
  },

  createFeeStructure: (data: CreateFeeStructureRequest, token: string) =>
    request<FeeStructure>('/fee-structures', { method: 'POST', body: data, token }),

  getFeeStructure: (id: string, token: string) =>
    request<FeeStructure>(`/fee-structures/${id}`, { token }),

  updateFeeStructure: (id: string, data: UpdateFeeStructureRequest, token: string) =>
    request<FeeStructure>(`/fee-structures/${id}`, { method: 'PATCH', body: data, token }),

  deleteFeeStructure: (id: string, token: string) =>
    request<void>(`/fee-structures/${id}`, { method: 'DELETE', token }),

  addFeeStructureItem: (id: string, data: FeeItemInput, token: string) =>
    request<FeeStructureItem>(`/fee-structures/${id}/items`, { method: 'POST', body: data, token }),

  deleteFeeStructureItem: (itemId: string, token: string) =>
    request<void>(`/fee-structures/items/${itemId}`, { method: 'DELETE', token }),

  // Invoices
  listInvoices: (params: { status?: string; learner_id?: string; term?: number; year?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.status) qs.set('status', params.status);
    if (params.learner_id) qs.set('learner_id', params.learner_id);
    if (params.term) qs.set('term', String(params.term));
    if (params.year) qs.set('year', String(params.year));
    return request<Invoice[]>(`/invoices?${qs.toString()}`, { token });
  },

  createInvoice: (data: CreateInvoiceRequest, token: string) =>
    request<Invoice>('/invoices', { method: 'POST', body: data, token }),

  getInvoice: (id: string, token: string) =>
    request<Invoice>(`/invoices/${id}`, { token }),

  updateInvoice: (id: string, data: UpdateInvoiceRequest, token: string) =>
    request<Invoice>(`/invoices/${id}`, { method: 'PATCH', body: data, token }),

  deleteInvoice: (id: string, token: string) =>
    request<void>(`/invoices/${id}`, { method: 'DELETE', token }),

  listInvoicePayments: (id: string, token: string) =>
    request<Payment[]>(`/invoices/${id}/payments`, { token }),

  listInvoiceDiscounts: (id: string, token: string) =>
    request<Discount[]>(`/invoices/${id}/discounts`, { token }),

  createInvoiceDiscount: (id: string, data: CreateDiscountRequest, token: string) =>
    request<Discount>(`/invoices/${id}/discounts`, { method: 'POST', body: data, token }),

  deleteInvoiceDiscount: (discountId: string, token: string) =>
    request<void>(`/invoices/discounts/${discountId}`, { method: 'DELETE', token }),

  // Payments
  listPayments: (params: { status?: string; channel?: string; term?: number; year?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.status) qs.set('status', params.status);
    if (params.channel) qs.set('channel', params.channel);
    if (params.term) qs.set('term', String(params.term));
    if (params.year) qs.set('year', String(params.year));
    return request<Payment[]>(`/payments?${qs.toString()}`, { token });
  },

  createPayment: (data: CreatePaymentRequest, token: string) =>
    request<Payment>('/payments', { method: 'POST', body: data, token }),

  getPayment: (id: string, token: string) =>
    request<Payment>(`/payments/${id}`, { token }),

  reversePayment: (id: string, token: string) =>
    request<Payment>(`/payments/${id}/reverse`, { method: 'POST', token }),

  initiateMpesaStk: (data: MpesaStkRequest, token: string) =>
    request<Payment>('/payments/mpesa/stk', { method: 'POST', body: data, token }),

  // Report cards
  listReportCards: (params: { learner_id?: string; term?: number; year?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.learner_id) qs.set('learner_id', params.learner_id);
    if (params.term) qs.set('term', String(params.term));
    if (params.year) qs.set('year', String(params.year));
    return request<ReportCard[]>(`/reports?${qs.toString()}`, { token });
  },

  getReportCard: (id: string, token: string) =>
    request<ReportCard>(`/reports/${id}`, { token }),

  generateReportCard: (params: { learner_id: string; term: number; year: number }, data: GenerateReportCardRequest, token: string) =>
    request<ReportCard>(`/reports/generate?learner_id=${params.learner_id}&term=${params.term}&year=${params.year}`, { method: 'POST', body: data, token }),

  updateReportCard: (id: string, data: UpdateReportCardRequest, token: string) =>
    request<ReportCard>(`/reports/${id}`, { method: 'PATCH', body: data, token }),

  deleteReportCard: (id: string, token: string) =>
    request<void>(`/reports/${id}`, { method: 'DELETE', token }),

  // Analytics
  getSchoolOverview: (token: string) =>
    request<SchoolOverview>('/analytics/overview', { token }),

  getStrandCoverage: (params: { grade?: string; stream?: string; term?: number; year?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.grade) qs.set('grade', params.grade);
    if (params.stream) qs.set('stream', params.stream);
    if (params.term) qs.set('term', String(params.term));
    if (params.year) qs.set('year', String(params.year));
    return request<StrandCoverage[]>(`/analytics/strand-coverage?${qs.toString()}`, { token });
  },

  getCompetencyDistribution: (params: { strand_id?: string; grade?: string; stream?: string; term?: number; year?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.strand_id) qs.set('strand_id', params.strand_id);
    if (params.grade) qs.set('grade', params.grade);
    if (params.stream) qs.set('stream', params.stream);
    if (params.term) qs.set('term', String(params.term));
    if (params.year) qs.set('year', String(params.year));
    return request<CompetencyDistribution[]>(`/analytics/competency-distribution?${qs.toString()}`, { token });
  },

  getTeacherVelocity: (params: { term?: number; year?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.term) qs.set('term', String(params.term));
    if (params.year) qs.set('year', String(params.year));
    return request<TeacherVelocity[]>(`/analytics/teacher-velocity?${qs.toString()}`, { token });
  },

  getLearnerPortfolio: (params: { grade?: string; stream?: string; term?: number; year?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.grade) qs.set('grade', params.grade);
    if (params.stream) qs.set('stream', params.stream);
    if (params.term) qs.set('term', String(params.term));
    if (params.year) qs.set('year', String(params.year));
    return request<LearnerPortfolio[]>(`/analytics/learner-portfolio?${qs.toString()}`, { token });
  },

  getAtRiskLearners: (params: { term?: number; year?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.term) qs.set('term', String(params.term));
    if (params.year) qs.set('year', String(params.year));
    return request<AlertLearner[]>(`/analytics/at-risk?${qs.toString()}`, { token });
  },

  getLearningAreaPerformance: (learnerId: string, params: { term?: number; year?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.term) qs.set('term', String(params.term));
    if (params.year) qs.set('year', String(params.year));
    return request<LearningAreaPerformance[]>(`/analytics/learners/${learnerId}/performance?${qs.toString()}`, { token });
  },

  // Staff
  listStaff: (params: { role?: string; department?: string; employment_type?: string; include_inactive?: boolean }, token: string) => {
    const qs = new URLSearchParams();
    if (params.role) qs.set('role', params.role);
    if (params.department) qs.set('department', params.department);
    if (params.employment_type) qs.set('employment_type', params.employment_type);
    if (params.include_inactive) qs.set('include_inactive', 'true');
    return request<StaffProfile[]>(`/staff?${qs.toString()}`, { token });
  },

  createStaff: (data: CreateStaffRequest, token: string) =>
    request<StaffProfile>('/staff', { method: 'POST', body: data, token }),

  getStaff: (id: string, token: string) =>
    request<StaffProfile>(`/staff/${id}`, { token }),

  updateStaff: (id: string, data: UpdateStaffRequest, token: string) =>
    request<StaffProfile>(`/staff/${id}`, { method: 'PATCH', body: data, token }),

  deleteStaff: (id: string, token: string) =>
    request<void>(`/staff/${id}`, { method: 'DELETE', token }),

  listStaffDocuments: (id: string, token: string) =>
    request<StaffDocument[]>(`/staff/${id}/documents`, { token }),

  createStaffDocument: (id: string, data: CreateStaffDocumentRequest, token: string) =>
    request<StaffDocument>(`/staff/${id}/documents`, { method: 'POST', body: data, token }),

  deleteStaffDocument: (docId: string, token: string) =>
    request<void>(`/staff/documents/${docId}`, { method: 'DELETE', token }),

  // Payroll
  listPayrollRuns: (params: { month?: number; year?: number; status?: string }, token: string) => {
    const qs = new URLSearchParams();
    if (params.month) qs.set('month', String(params.month));
    if (params.year) qs.set('year', String(params.year));
    if (params.status) qs.set('status', params.status);
    return request<PayrollRun[]>(`/payroll?${qs.toString()}`, { token });
  },

  createPayrollRun: (data: CreatePayrollRunRequest, token: string) =>
    request<PayrollRun>('/payroll', { method: 'POST', body: data, token }),

  getPayrollRun: (id: string, token: string) =>
    request<PayrollRun>(`/payroll/${id}`, { token }),

  updatePayrollRun: (id: string, data: UpdatePayrollRunRequest, token: string) =>
    request<PayrollRun>(`/payroll/${id}`, { method: 'PATCH', body: data, token }),

  deletePayrollRun: (id: string, token: string) =>
    request<void>(`/payroll/${id}`, { method: 'DELETE', token }),

  // Leave
  listLeaveRequests: (params: { status?: string; staff_id?: string; leave_type?: string }, token: string) => {
    const qs = new URLSearchParams();
    if (params.status) qs.set('status', params.status);
    if (params.staff_id) qs.set('staff_id', params.staff_id);
    if (params.leave_type) qs.set('leave_type', params.leave_type);
    return request<LeaveRequest[]>(`/leave?${qs.toString()}`, { token });
  },

  createLeaveRequest: (data: CreateLeaveRequest, token: string) =>
    request<LeaveRequest>('/leave', { method: 'POST', body: data, token }),

  getLeaveRequest: (id: string, token: string) =>
    request<LeaveRequest>(`/leave/${id}`, { token }),

  approveLeaveRequest: (id: string, data: ApproveLeaveRequest, token: string) =>
    request<LeaveRequest>(`/leave/${id}/approve`, { method: 'POST', body: data, token }),

  denyLeaveRequest: (id: string, data: DenyLeaveRequest, token: string) =>
    request<LeaveRequest>(`/leave/${id}/deny`, { method: 'POST', body: data, token }),

  cancelLeaveRequest: (id: string, token: string) =>
    request<LeaveRequest>(`/leave/${id}/cancel`, { method: 'POST', token }),

  // Staff attendance
  listStaffAttendance: (params: { date?: string; staff_id?: string; status?: string }, token: string) => {
    const qs = new URLSearchParams();
    if (params.date) qs.set('date', params.date);
    if (params.staff_id) qs.set('staff_id', params.staff_id);
    if (params.status) qs.set('status', params.status);
    return request<StaffAttendance[]>(`/staff-attendance?${qs.toString()}`, { token });
  },

  createStaffAttendance: (data: CreateStaffAttendanceRequest, token: string) =>
    request<StaffAttendance>('/staff-attendance', { method: 'POST', body: data, token }),

  updateStaffAttendance: (id: string, data: UpdateStaffAttendanceRequest, token: string) =>
    request<StaffAttendance>(`/staff-attendance/${id}`, { method: 'PATCH', body: data, token }),

  deleteStaffAttendance: (id: string, token: string) =>
    request<void>(`/staff-attendance/${id}`, { method: 'DELETE', token }),

  // Appraisals
  listAppraisals: (params: { staff_id?: string; year?: number; term?: number }, token: string) => {
    const qs = new URLSearchParams();
    if (params.staff_id) qs.set('staff_id', params.staff_id);
    if (params.year) qs.set('year', String(params.year));
    if (params.term) qs.set('term', String(params.term));
    return request<StaffAppraisal[]>(`/appraisals?${qs.toString()}`, { token });
  },

  createAppraisal: (data: CreateAppraisalRequest, token: string) =>
    request<StaffAppraisal>('/appraisals', { method: 'POST', body: data, token }),

  getAppraisal: (id: string, token: string) =>
    request<StaffAppraisal>(`/appraisals/${id}`, { token }),

  updateAppraisal: (id: string, data: UpdateAppraisalRequest, token: string) =>
    request<StaffAppraisal>(`/appraisals/${id}`, { method: 'PATCH', body: data, token }),

  deleteAppraisal: (id: string, token: string) =>
    request<void>(`/appraisals/${id}`, { method: 'DELETE', token }),
};
