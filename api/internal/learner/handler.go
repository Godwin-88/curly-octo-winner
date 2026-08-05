package learner

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the learner API.
type Handler struct {
	service *Service
}

// NewHandler creates a new learner handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Mount registers all learner routes under the provided router.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/learners", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.deactivate)
		r.Post("/{id}/reactivate", h.reactivate)
		r.Get("/{id}/guardians", h.listGuardians)
		r.Get("/{id}/progressions", h.listProgressions)

		// Documents
		r.Get("/{id}/documents", h.listDocuments)
		r.Post("/{id}/documents", h.uploadDocument)
		r.Get("/documents/{docId}", h.getDocument)
		r.Delete("/documents/{docId}", h.deleteDocument)

		// Progression
		r.Post("/{id}/promote", h.promote)
		r.Post("/{id}/retain", h.retain)
		r.Post("/{id}/transfer-out", h.transferOut)
		r.Post("/{id}/transfer-in", h.transferIn)
	})
}

// list handles GET /learners
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	grade := r.URL.Query().Get("grade")
	stream := r.URL.Query().Get("stream")
	search := r.URL.Query().Get("search")
	includeInactive := boolQuery(r.URL.Query().Get("include_inactive"))

	learners, err := h.service.List(r.Context(), tenantID, grade, stream, search, includeInactive)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, learners)
}

// create handles POST /learners
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var req CreateLearnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	learner, err := h.service.Create(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, learner)
}

// get handles GET /learners/{id}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	learner, err := h.service.GetByID(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, learner)
}

// update handles PATCH /learners/{id}
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	var req UpdateLearnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	learner, err := h.service.Update(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, learner)
}

// deactivate handles DELETE /learners/{id}
func (h *Handler) deactivate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	if err := h.service.Deactivate(r.Context(), tenantID, id); err != nil {
		httputil.RespondBadRequest(w, "DEACTIVATE_FAILED", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}

// reactivate handles POST /learners/{id}/reactivate
func (h *Handler) reactivate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	learner, err := h.service.GetByID(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	if err := h.service.Reactivate(r.Context(), tenantID, id); err != nil {
		httputil.RespondBadRequest(w, "REACTIVATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, learner)
}

// listGuardians handles GET /learners/{id}/guardians
func (h *Handler) listGuardians(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	guardians, err := h.service.ListGuardians(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, guardians)
}

// --- Document handlers ---

// listDocuments handles GET /learners/{id}/documents
func (h *Handler) listDocuments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	docs, err := h.service.ListDocuments(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, docs)
}

// uploadDocument handles POST /learners/{id}/documents
func (h *Handler) uploadDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	var req CreateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	req.LearnerID = id

	doc, err := h.service.UploadDocument(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPLOAD_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, doc)
}

// getDocument handles GET /learners/documents/{docId}
func (h *Handler) getDocument(w http.ResponseWriter, r *http.Request) {
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

	doc, err := h.service.GetDocument(r.Context(), tenantID, docID)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, doc)
}

// deleteDocument handles DELETE /learners/documents/{docId}
func (h *Handler) deleteDocument(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.DeleteDocument(r.Context(), tenantID, docID); err != nil {
		httputil.RespondBadRequest(w, "DELETE_FAILED", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}

// --- Progression handlers ---

// listProgressions handles GET /learners/{id}/progressions
func (h *Handler) listProgressions(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	progressions, err := h.service.ListProgressions(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, progressions)
}

// promote handles POST /learners/{id}/promote
func (h *Handler) promote(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	var req PromoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	req.LearnerID = id

	p, err := h.service.Promote(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "PROMOTE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, p)
}

// retain handles POST /learners/{id}/retain
func (h *Handler) retain(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	var req RetainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	req.LearnerID = id

	p, err := h.service.Retain(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "RETAIN_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, p)
}

// transferOut handles POST /learners/{id}/transfer-out
func (h *Handler) transferOut(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	var req struct {
		Term  *int   `json:"term,omitempty"`
		Year  int    `json:"year"`
		Notes string `json:"notes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	p, err := h.service.TransferOut(r.Context(), tenantID, id, req.Term, req.Year, nil, req.Notes)
	if err != nil {
		httputil.RespondBadRequest(w, "TRANSFER_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, p)
}

// transferIn handles POST /learners/{id}/transfer-in
func (h *Handler) transferIn(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid learner ID")
		return
	}

	var req struct {
		ToGrade string `json:"to_grade"`
		Term    *int   `json:"term,omitempty"`
		Year    int    `json:"year"`
		Notes   string `json:"notes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	p, err := h.service.TransferIn(r.Context(), tenantID, id, req.ToGrade, req.Term, req.Year, nil, req.Notes)
	if err != nil {
		httputil.RespondBadRequest(w, "TRANSFER_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, p)
}

// boolQuery parses a boolean query parameter with a default of false.
func boolQuery(s string) bool {
	if s == "" {
		return false
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}
	return b
}
