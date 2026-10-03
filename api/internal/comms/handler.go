package comms

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/comms/contacts"
	"github.com/shule360/api/internal/comms/inbox"
	"github.com/shule360/api/internal/comms/sms"
	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the communications API.
type Handler struct {
	service      *CommsService
	inboxService interface {
		ListConversations(ctx context.Context, tenantID uuid.UUID, status, assignedTo string, limit, offset int) ([]inbox.Conversation, error)
	}
	smsService      *sms.SMSService
	contactsHandler *contacts.Handler
}

// NewHandler creates a new communications handler.
func NewHandler(service *CommsService) *Handler {
	return &Handler{service: service}
}

// NewHandlerWithSMS creates a new communications handler with SMS support.
func NewHandlerWithSMS(service *CommsService, smsService *sms.SMSService) *Handler {
	return &Handler{service: service, smsService: smsService}
}

// SetContactsHandler wires the contact book routes. Kept as a setter because
// the comms handler is constructed before the contacts service exists.
func (h *Handler) SetContactsHandler(ch *contacts.Handler) {
	h.contactsHandler = ch
}

// Mount registers all comms routes under the provided router.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/messages", func(r chi.Router) {
		r.Post("/", h.createMessage)
		r.Get("/", h.listMessages)
		// The estimate takes an audience + body, so it is a POST (this is what
		// the spec and the web client use). The GET below is kept as an alias
		// for older clients.
		r.Post("/estimate", h.estimateReach)
		r.Get("/estimate", h.estimateReach)
		r.Get("/audience-options", h.audienceOptions)
		r.Get("/{id}", h.getMessage)
		r.Get("/{id}/logs", h.getMessageLogs)
		r.Delete("/{id}", h.cancelMessage)
		r.Post("/{id}/resend-failed", h.resendFailed)
	})

	r.Route("/conversations", func(r chi.Router) {
		r.Get("/", h.listConversations)
		r.Post("/{id}/reply", h.sendReply)
		r.Patch("/{id}/assign", h.assignConversation)
		r.Patch("/{id}/status", h.updateConversationStatus)
		r.Get("/{id}", h.getConversation)
	})

	if h.smsService != nil {
		smsHandler := sms.NewHandler(h.smsService)
		smsHandler.Mount(r)
	}

	// Contact book: the people a school can actually message. Mounted here so
	// it inherits the same all-staff role group as the rest of Communications.
	if h.contactsHandler != nil {
		h.contactsHandler.Mount(r)
	}
}

// respondError maps a service error to the response the caller can act on.
func respondError(w http.ResponseWriter, err error, code string) {
	var validation *ValidationError
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.RespondNotFound(w, "NOT_FOUND", "Message not found")
	case errors.As(err, &validation):
		httputil.RespondBadRequest(w, code, validation.Message)
	default:
		httputil.RespondInternalError(w, err)
	}
}

// actor returns who is making the request: a member of the school's staff, or
// a platform/group user (who has no row in the school's staff table).
func actor(r *http.Request) Actor {
	var a Actor
	if id, ok := middleware.GetStaffID(r); ok && id != uuid.Nil {
		a.StaffID = &id
	}
	if session := middleware.GetSession(r); session.OperatorID != uuid.Nil {
		a.OperatorID = &session.OperatorID
	}
	return a
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
	// The header form is what generic HTTP clients send; the body field is
	// what the web app sends.
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = r.Header.Get("Idempotency-Key")
	}

	msg, err := h.service.CreateAndSend(r.Context(), tenantID, actor(r), req)
	if err != nil {
		respondError(w, err, "CREATE_FAILED")
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
		respondError(w, err, "ESTIMATE_FAILED")
		return
	}

	httputil.RespondOK(w, estimate)
}

// audienceOptions handles GET /messages/audience-options
func (h *Handler) audienceOptions(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	options, err := h.service.GetAudienceOptions(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, options)
}

// messageRef reads the school and message id of a /messages/{id} request.
func messageRef(w http.ResponseWriter, r *http.Request) (tenantID, id uuid.UUID, ok bool) {
	tenantID, ok = middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid message ID")
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, id, true
}

// getMessage handles GET /messages/{id}
func (h *Handler) getMessage(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := messageRef(w, r)
	if !ok {
		return
	}

	msg, stats, err := h.service.GetMessage(r.Context(), tenantID, id)
	if err != nil {
		respondError(w, err, "GET_FAILED")
		return
	}

	httputil.RespondOK(w, map[string]any{
		"message": msg,
		"stats":   stats,
	})
}

// getMessageLogs handles GET /messages/{id}/logs
func (h *Handler) getMessageLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := messageRef(w, r)
	if !ok {
		return
	}

	limit := parseIntDefault(r.URL.Query().Get("limit"), 100)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	logs, err := h.service.GetMessageLogs(r.Context(), tenantID, id, r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		respondError(w, err, "GET_FAILED")
		return
	}

	httputil.RespondOK(w, logs)
}

// cancelMessage handles DELETE /messages/{id}
func (h *Handler) cancelMessage(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := messageRef(w, r)
	if !ok {
		return
	}

	if err := h.service.CancelScheduled(r.Context(), tenantID, id); err != nil {
		respondError(w, err, "CANCEL_FAILED")
		return
	}

	httputil.RespondNoContent(w)
}

// resendFailed handles POST /messages/{id}/resend-failed
func (h *Handler) resendFailed(w http.ResponseWriter, r *http.Request) {
	tenantID, id, ok := messageRef(w, r)
	if !ok {
		return
	}

	var req struct {
		IncludeUncertain bool   `json:"include_uncertain"`
		IdempotencyKey   string `json:"idempotency_key"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
			return
		}
	}
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = r.Header.Get("Idempotency-Key")
	}

	msg, err := h.service.ResendFailed(r.Context(), tenantID, actor(r), id, req.IncludeUncertain, req.IdempotencyKey)
	if err != nil {
		respondError(w, err, "RESEND_FAILED")
		return
	}

	httputil.RespondCreated(w, msg)
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
