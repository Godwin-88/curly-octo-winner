package sms

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

type Handler struct {
	service *SMSService
}

func NewHandler(service *SMSService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Mount(r chi.Router) {
	r.Route("/sms", func(r chi.Router) {
		// Sending lives at POST /messages (comms.CommsService): one path that
		// records every recipient before the provider is called. The former
		// /sms/send and /sms/campaign routes wrote to columns that do not
		// exist and were removed rather than repaired.
		r.Get("/templates", h.listTemplates)
		r.Post("/templates", h.createTemplate)
		r.Get("/templates/{id}", h.getTemplate)
		r.Patch("/templates/{id}", h.updateTemplate)
		r.Delete("/templates/{id}", h.deleteTemplate)
	})
}

func (h *Handler) listTemplates(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	rows, err := h.service.pool.Query(r.Context(), `
		SELECT id, tenant_id, name, content, variables, category, created_at
		FROM sms_templates
		WHERE tenant_id = $1
		ORDER BY category, name
	`, tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type Template struct {
		ID        uuid.UUID `json:"id"`
		TenantID  uuid.UUID `json:"tenant_id"`
		Name      string    `json:"name"`
		Content   string    `json:"content"`
		Variables []string  `json:"variables"`
		Category  string    `json:"category"`
		CreatedAt time.Time `json:"created_at"`
	}

	// created_at was scanned into a string, which pgx refuses for timestamptz:
	// the list answered 500 as soon as the school saved its first template.
	templates := []Template{}
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.Content, &t.Variables, &t.Category, &t.CreatedAt); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		templates = append(templates, t)
	}
	httputil.RespondOK(w, templates)
}

func (h *Handler) createTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var req struct {
		Name      string   `json:"name"`
		Content   string   `json:"content"`
		Variables []string `json:"variables"`
		Category  string   `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	if req.Name == "" || req.Content == "" {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "name and content are required")
		return
	}

	var id uuid.UUID
	err := h.service.pool.QueryRow(r.Context(), `
		INSERT INTO sms_templates (tenant_id, name, content, variables, category)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, tenantID, req.Name, req.Content, req.Variables, req.Category).Scan(&id)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}

	httputil.RespondCreated(w, map[string]any{
		"id":        id,
		"name":      req.Name,
		"content":   req.Content,
		"variables": req.Variables,
		"category":  req.Category,
	})
}

func (h *Handler) getTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid template ID")
		return
	}

	var t struct {
		ID        uuid.UUID `json:"id"`
		TenantID  uuid.UUID `json:"tenant_id"`
		Name      string    `json:"name"`
		Content   string    `json:"content"`
		Variables []string  `json:"variables"`
		Category  string    `json:"category"`
		CreatedAt time.Time `json:"created_at"`
	}
	err = h.service.pool.QueryRow(r.Context(), `
		SELECT id, tenant_id, name, content, variables, category, created_at
		FROM sms_templates
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id).Scan(&t.ID, &t.TenantID, &t.Name, &t.Content, &t.Variables, &t.Category, &t.CreatedAt)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, t)
}

func (h *Handler) updateTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid template ID")
		return
	}

	var req struct {
		Name      *string   `json:"name,omitempty"`
		Content   *string   `json:"content,omitempty"`
		Variables *[]string `json:"variables,omitempty"`
		Category  *string   `json:"category,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	_, err = h.service.pool.Exec(r.Context(), `
		UPDATE sms_templates
		SET name = COALESCE($3, name), content = COALESCE($4, content),
		    variables = COALESCE($5, variables), category = COALESCE($6, category),
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id, req.Name, req.Content, req.Variables, req.Category)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, map[string]string{"status": "updated"})
}

func (h *Handler) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid template ID")
		return
	}

	_, err = h.service.pool.Exec(r.Context(), `
		DELETE FROM sms_templates WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
