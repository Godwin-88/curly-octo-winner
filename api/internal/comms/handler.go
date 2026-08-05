package comms

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/comms/inbox"
	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the communications API.
type Handler struct {
	service      *CommsService
	inboxService interface {
		ListConversations(ctx context.Context, tenantID uuid.UUID, status, assignedTo string, limit, offset int) ([]inbox.Conversation, error)
	}
}

// NewHandler creates a new communications handler.
func NewHandler(service *CommsService) *Handler {
	return &Handler{service: service}
}

// Mount registers all comms routes under the provided router.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/messages", func(r chi.Router) {
		r.Post("/", h.createMessage)
		r.Get("/", h.listMessages)
		r.Get("/estimate", h.estimateReach)
		r.Get("/{id}", h.getMessage)
		r.Get("/{id}/logs", h.getMessageLogs)
		r.Delete("/{id}", h.cancelMessage)
	})

	r.Route("/conversations", func(r chi.Router) {
		r.Get("/", h.listConversations)
		r.Post("/{id}/reply", h.sendReply)
		r.Patch("/{id}/assign", h.assignConversation)
		r.Patch("/{id}/status", h.updateConversationStatus)
		r.Get("/{id}", h.getConversation)
	})
}

// createMessage handles POST /messages
func (h *Handler) createMessage(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var req CreateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	msg, err := h.service.CreateAndSend(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}

	httputil.RespondCreated(w, msg)
}

// listMessages handles GET /messages
func (h *Handler) listMessages(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	status := r.URL.Query().Get("status")
	channel := r.URL.Query().Get("channel")
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	messages, err := h.service.ListMessages(r.Context(), tenantID, status, channel, limit, offset)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}

	httputil.RespondOK(w, messages)
}

// estimateReach handles POST /messages/estimate
func (h *Handler) estimateReach(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var req CreateMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	estimate, err := h.service.EstimateReach(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "ESTIMATE_FAILED", err.Error())
		return
	}

	httputil.RespondOK(w, estimate)
}

// getMessage handles GET /messages/{id}
func (h *Handler) getMessage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid message ID")
		return
	}

	msg, stats, err := h.service.GetMessage(r.Context(), id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}

	httputil.RespondOK(w, map[string]any{
		"message": msg,
		"stats":   stats,
	})
}

// getMessageLogs handles GET /messages/{id}/logs
func (h *Handler) getMessageLogs(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid message ID")
		return
	}

	limit := parseIntDefault(r.URL.Query().Get("limit"), 100)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	logs, err := h.service.GetMessageLogs(r.Context(), id, limit, offset)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}

	httputil.RespondOK(w, logs)
}

// cancelMessage handles DELETE /messages/{id}
func (h *Handler) cancelMessage(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid message ID")
		return
	}

	if err := h.service.CancelScheduled(r.Context(), id); err != nil {
		httputil.RespondBadRequest(w, "CANCEL_FAILED", err.Error())
		return
	}

	httputil.RespondNoContent(w)
}

// listConversations handles GET /conversations
func (h *Handler) listConversations(w http.ResponseWriter, r *http.Request) {
	// Handled in inbox handler
	httputil.RespondOK(w, []inbox.Conversation{})
}

// getConversation handles GET /conversations/{id}
func (h *Handler) getConversation(w http.ResponseWriter, r *http.Request) {
	httputil.RespondOK(w, map[string]any{})
}

// sendReply handles POST /conversations/{id}/reply
func (h *Handler) sendReply(w http.ResponseWriter, r *http.Request) {
	httputil.RespondOK(w, map[string]any{})
}

// assignConversation handles PATCH /conversations/{id}/assign
func (h *Handler) assignConversation(w http.ResponseWriter, r *http.Request) {
	httputil.RespondOK(w, map[string]any{})
}

// updateConversationStatus handles PATCH /conversations/{id}/status
func (h *Handler) updateConversationStatus(w http.ResponseWriter, r *http.Request) {
	httputil.RespondOK(w, map[string]any{})
}

// parseIntDefault parses an int with a default fallback.
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

// errorCodeFor converts an error to an HTTP status + code.
func errorCodeFor(err error) (int, string) {
	return http.StatusInternalServerError, "INTERNAL_ERROR"
}
