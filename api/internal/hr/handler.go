package hr

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the HR API.
type Handler struct {
	service *Service
}

// NewHandler creates a new HR handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Mount registers all HR routes under the provided router.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/staff", func(r chi.Router) {
		r.Get("/", h.listStaff)
		r.Post("/", h.createStaff)
		r.Get("/{id}", h.getStaff)
		r.Patch("/{id}", h.updateStaff)
		r.Delete("/{id}", h.deleteStaff)
		r.Get("/{id}/documents", h.listStaffDocuments)
		r.Post("/{id}/documents", h.createStaffDocument)
		r.Delete("/documents/{docId}", h.deleteStaffDocument)
	})
	r.Route("/payroll", func(r chi.Router) {
		r.Get("/", h.listPayrollRuns)
		r.Post("/", h.createPayrollRun)
		r.Get("/{id}", h.getPayrollRun)
		r.Patch("/{id}", h.updatePayrollRun)
		r.Delete("/{id}", h.deletePayrollRun)
	})
	r.Route("/leave", func(r chi.Router) {
		r.Get("/", h.listLeaveRequests)
		r.Post("/", h.createLeaveRequest)
		r.Get("/{id}", h.getLeaveRequest)
		r.Post("/{id}/approve", h.approveLeaveRequest)
		r.Post("/{id}/deny", h.denyLeaveRequest)
		r.Post("/{id}/cancel", h.cancelLeaveRequest)
	})
	r.Route("/staff-attendance", func(r chi.Router) {
		r.Get("/", h.listStaffAttendance)
		r.Post("/", h.createStaffAttendance)
		r.Patch("/{id}", h.updateStaffAttendance)
		r.Delete("/{id}", h.deleteStaffAttendance)
	})
	r.Route("/appraisals", func(r chi.Router) {
		r.Get("/", h.listAppraisals)
		r.Post("/", h.createAppraisal)
		r.Get("/{id}", h.getAppraisal)
		r.Patch("/{id}", h.updateAppraisal)
		r.Delete("/{id}", h.deleteAppraisal)
	})
}

// --- Staff handlers ---

func (h *Handler) listStaff(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	role := r.URL.Query().Get("role")
	department := r.URL.Query().Get("department")
	employmentType := r.URL.Query().Get("employment_type")
	includeInactive := r.URL.Query().Get("include_inactive") == "true"
	staff, err := h.service.ListStaff(r.Context(), tenantID, role, department, employmentType, includeInactive)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, staff)
}

func (h *Handler) createStaff(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	staff, err := h.service.CreateStaff(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, staff)
}

func (h *Handler) getStaff(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid staff ID")
		return
	}
	staff, err := h.service.GetStaff(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, staff)
}

func (h *Handler) updateStaff(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid staff ID")
		return
	}
	var req UpdateStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	staff, err := h.service.UpdateStaff(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, staff)
}

func (h *Handler) deleteStaff(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid staff ID")
		return
	}
	if err := h.service.DeleteStaff(r.Context(), tenantID, id); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// --- Staff document handlers ---

func (h *Handler) listStaffDocuments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	staffID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid staff ID")
		return
	}
	docs, err := h.service.ListStaffDocuments(r.Context(), tenantID, staffID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, docs)
}

func (h *Handler) createStaffDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	staffID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid staff ID")
		return
	}
	var req CreateStaffDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	doc, err := h.service.CreateStaffDocument(r.Context(), tenantID, staffID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, doc)
}

func (h *Handler) deleteStaffDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	docID, err := uuid.Parse(chi.URLParam(r, "docId"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid document ID")
		return
	}
	if err := h.service.DeleteStaffDocument(r.Context(), tenantID, docID); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// --- Payroll handlers ---

func (h *Handler) listPayrollRuns(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	month, _ := strconv.Atoi(r.URL.Query().Get("month"))
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	status := r.URL.Query().Get("status")
	runs, err := h.service.ListPayrollRuns(r.Context(), tenantID, month, year, status)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, runs)
}

func (h *Handler) createPayrollRun(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreatePayrollRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	run, err := h.service.CreatePayrollRun(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, run)
}

func (h *Handler) getPayrollRun(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid payroll run ID")
		return
	}
	run, err := h.service.GetPayrollRun(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, run)
}

func (h *Handler) updatePayrollRun(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid payroll run ID")
		return
	}
	var req UpdatePayrollRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	run, err := h.service.UpdatePayrollRun(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, run)
}

func (h *Handler) deletePayrollRun(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid payroll run ID")
		return
	}
	if err := h.service.DeletePayrollRun(r.Context(), tenantID, id); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// --- Leave handlers ---

func (h *Handler) listLeaveRequests(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	status := r.URL.Query().Get("status")
	staffID := r.URL.Query().Get("staff_id")
	leaveType := r.URL.Query().Get("leave_type")
	leaves, err := h.service.ListLeaveRequests(r.Context(), tenantID, status, staffID, leaveType)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, leaves)
}

func (h *Handler) createLeaveRequest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	leave, err := h.service.CreateLeaveRequest(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, leave)
}

func (h *Handler) getLeaveRequest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid leave request ID")
		return
	}
	leave, err := h.service.GetLeaveRequest(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, leave)
}

func (h *Handler) approveLeaveRequest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid leave request ID")
		return
	}
	var req ApproveLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	leave, err := h.service.ApproveLeaveRequest(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "APPROVE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, leave)
}

func (h *Handler) denyLeaveRequest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid leave request ID")
		return
	}
	var req DenyLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	leave, err := h.service.DenyLeaveRequest(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "DENY_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, leave)
}

func (h *Handler) cancelLeaveRequest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid leave request ID")
		return
	}
	leave, err := h.service.CancelLeaveRequest(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondBadRequest(w, "CANCEL_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, leave)
}

// --- Staff attendance handlers ---

func (h *Handler) listStaffAttendance(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	date := r.URL.Query().Get("date")
	staffID := r.URL.Query().Get("staff_id")
	status := r.URL.Query().Get("status")
	records, err := h.service.ListStaffAttendance(r.Context(), tenantID, date, staffID, status)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, records)
}

func (h *Handler) createStaffAttendance(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateStaffAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	record, err := h.service.CreateStaffAttendance(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, record)
}

func (h *Handler) updateStaffAttendance(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid attendance record ID")
		return
	}
	var req UpdateStaffAttendanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	record, err := h.service.UpdateStaffAttendance(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, record)
}

func (h *Handler) deleteStaffAttendance(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid attendance record ID")
		return
	}
	if err := h.service.DeleteStaffAttendance(r.Context(), tenantID, id); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// --- Appraisal handlers ---

func (h *Handler) listAppraisals(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	staffID := r.URL.Query().Get("staff_id")
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	term, _ := strconv.Atoi(r.URL.Query().Get("term"))
	appraisals, err := h.service.ListAppraisals(r.Context(), tenantID, staffID, year, term)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, appraisals)
}

func (h *Handler) createAppraisal(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateAppraisalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	appraisal, err := h.service.CreateAppraisal(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, appraisal)
}

func (h *Handler) getAppraisal(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid appraisal ID")
		return
	}
	appraisal, err := h.service.GetAppraisal(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, appraisal)
}

func (h *Handler) updateAppraisal(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid appraisal ID")
		return
	}
	var req UpdateAppraisalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	appraisal, err := h.service.UpdateAppraisal(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, appraisal)
}

func (h *Handler) deleteAppraisal(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid appraisal ID")
		return
	}
	if err := h.service.DeleteAppraisal(r.Context(), tenantID, id); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}
