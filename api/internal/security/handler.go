package security

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the security & compliance API.
type Handler struct {
	service *Service
}

// NewHandler creates a new security handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Mount registers all security routes under the provided router.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/security", func(r chi.Router) {
		// RBAC
		r.Get("/permissions", h.listPermissions)
		r.Get("/roles", h.listRoles)
		r.Get("/roles/{role}", h.getRolePermissions)
		r.Post("/roles/{role}/permissions", h.grantRolePermission)
		r.Delete("/roles/{role}/permissions/{code}", h.revokeRolePermission)

		// Sessions
		r.Get("/sessions", h.listSessions)
		r.Delete("/sessions/{id}", h.revokeSession)

		// Audit log
		r.Get("/audit", h.listAuditLogs)

		// Data processing register (KDPA)
		r.Get("/data-processing", h.listProcessingRecords)
		r.Post("/data-processing", h.createProcessingRecord)
		r.Get("/data-processing/{id}", h.getProcessingRecord)
		r.Patch("/data-processing/{id}", h.updateProcessingRecord)
		r.Delete("/data-processing/{id}", h.deleteProcessingRecord)

		// Consent management
		r.Get("/consent", h.listConsentAgreements)
		r.Post("/consent/grant", h.grantConsent)
		r.Post("/consent/revoke", h.revokeConsent)

		// Erasure / data subject rights
		r.Get("/erasure", h.listErasureRequests)
		r.Post("/erasure", h.createErasureRequest)
		r.Patch("/erasure/{id}/status", h.updateErasureRequestStatus)

		// Summary
		r.Get("/summary", h.getSummary)
	})
}

// --- RBAC handlers ---

func (h *Handler) listPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.service.ListPermissions(r.Context())
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, perms)
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.service.ListRoles(r.Context())
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, roles)
}

func (h *Handler) getRolePermissions(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	perms, err := h.service.GetRolePermissions(r.Context(), role)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, RolePermissionsResponse{Role: role, Permissions: perms})
}

func (h *Handler) grantRolePermission(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	var req struct {
		PermissionCode string `json:"permission_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.PermissionCode == "" {
		httputil.RespondBadRequest(w, "MISSING_FIELD", "permission_code is required")
		return
	}
	if err := h.service.GrantRolePermission(r.Context(), role, req.PermissionCode); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, map[string]string{"status": "granted"})
}

func (h *Handler) revokeRolePermission(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	code := chi.URLParam(r, "code")
	if err := h.service.RevokeRolePermission(r.Context(), role, code); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// --- Session handlers ---

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	staffID, ok := middleware.GetStaffID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Staff ID not found")
		return
	}
	sessions, err := h.service.ListActiveSessions(r.Context(), tenantID, staffID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, sessions)
}

func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	staffID, ok := middleware.GetStaffID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Staff ID not found")
		return
	}
	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid session ID")
		return
	}
	// Revoke by session ID (only own sessions)
	_, err = h.service.pool.Exec(r.Context(), `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE id = $1 AND tenant_id = $2 AND staff_id = $3 AND revoked_at IS NULL
	`, sessionID, tenantID, staffID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// --- Audit log handlers ---

func (h *Handler) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	entityType := r.URL.Query().Get("entity_type")
	action := r.URL.Query().Get("action")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	logs, err := h.service.ListAuditLogs(r.Context(), tenantID, entityType, action, limit, offset)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, logs)
}

// --- Data processing register handlers ---

func (h *Handler) listProcessingRecords(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	records, err := h.service.ListProcessingRecords(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, records)
}

func (h *Handler) getProcessingRecord(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid record ID")
		return
	}
	record, err := h.service.GetProcessingRecord(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, record)
}

func (h *Handler) createProcessingRecord(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	staffID, _ := middleware.GetStaffID(r)
	var input CreateProcessingRecordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondBadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	if input.Activity == "" || input.Purpose == "" || input.LegalBasis == "" || input.DataSubjects == "" {
		httputil.RespondBadRequest(w, "MISSING_FIELD", "activity, purpose, legal_basis, and data_subjects are required")
		return
	}
	record, err := h.service.CreateProcessingRecord(r.Context(), tenantID, input, &staffID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondCreated(w, record)
}

func (h *Handler) updateProcessingRecord(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid record ID")
		return
	}
	var input UpdateProcessingRecordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondBadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	record, err := h.service.UpdateProcessingRecord(r.Context(), tenantID, id, input)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, record)
}

func (h *Handler) deleteProcessingRecord(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid record ID")
		return
	}
	if err := h.service.DeleteProcessingRecord(r.Context(), tenantID, id); err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// --- Consent handlers ---

func (h *Handler) listConsentAgreements(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var guardianID *uuid.UUID
	if g := r.URL.Query().Get("guardian_id"); g != "" {
		id, err := uuid.Parse(g)
		if err != nil {
			httputil.RespondBadRequest(w, "INVALID_ID", "Invalid guardian_id")
			return
		}
		guardianID = &id
	}
	consents, err := h.service.ListConsentAgreements(r.Context(), tenantID, guardianID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, consents)
}

func (h *Handler) grantConsent(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var input GrantConsentInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondBadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	if input.GuardianID == uuid.Nil || input.ConsentType == "" {
		httputil.RespondBadRequest(w, "MISSING_FIELD", "guardian_id and consent_type are required")
		return
	}
	consent, err := h.service.GrantConsent(r.Context(), tenantID, input)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, consent)
}

func (h *Handler) revokeConsent(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req struct {
		GuardianID  uuid.UUID `json:"guardian_id"`
		ConsentType string    `json:"consent_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.GuardianID == uuid.Nil || req.ConsentType == "" {
		httputil.RespondBadRequest(w, "MISSING_FIELD", "guardian_id and consent_type are required")
		return
	}
	consent, err := h.service.RevokeConsent(r.Context(), tenantID, req.GuardianID, req.ConsentType)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, consent)
}

// --- Erasure / data subject rights handlers ---

func (h *Handler) listErasureRequests(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	status := r.URL.Query().Get("status")
	requests, err := h.service.ListErasureRequests(r.Context(), tenantID, status)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, requests)
}

func (h *Handler) createErasureRequest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var input CreateErasureRequestInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		httputil.RespondBadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	if input.SubjectID == uuid.Nil || input.SubjectType == "" || input.RequestedBy == "" {
		httputil.RespondBadRequest(w, "MISSING_FIELD", "subject_type, subject_id, and requested_by are required")
		return
	}
	request, err := h.service.CreateErasureRequest(r.Context(), tenantID, input)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondCreated(w, request)
}

func (h *Handler) updateErasureRequestStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid request ID")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	if req.Status == "" {
		httputil.RespondBadRequest(w, "MISSING_FIELD", "status is required")
		return
	}
	request, err := h.service.UpdateErasureRequestStatus(r.Context(), tenantID, id, req.Status)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, request)
}

// --- Summary handler ---

func (h *Handler) getSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	summary, err := h.service.GetSummary(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, summary)
}
