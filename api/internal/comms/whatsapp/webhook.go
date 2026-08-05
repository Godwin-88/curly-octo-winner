package whatsapp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/pkg/httputil"
)

// WebhookHandler handles Meta WhatsApp Cloud API webhooks.
type WebhookHandler struct {
	verifyToken string
	pool        *pgxpool.Pool
	chatbot     *Chatbot
	waClient    *WAClient
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(verifyToken string, pool *pgxpool.Pool, chatbot *Chatbot, waClient *WAClient) *WebhookHandler {
	return &WebhookHandler{
		verifyToken: verifyToken,
		pool:        pool,
		chatbot:     chatbot,
		waClient:    waClient,
	}
}

// ServeHTTP handles both GET (verification) and POST (webhook events).
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleVerification(w, r)
	case http.MethodPost:
		h.handleEvent(w, r)
	default:
		httputil.RespondError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}
}

// handleVerification handles the Meta webhook verification challenge.
func (h *WebhookHandler) handleVerification(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode != "subscribe" {
		httputil.RespondBadRequest(w, "VERIFY_FAILED", "Invalid hub.mode")
		return
	}

	if token != h.verifyToken {
		httputil.RespondForbidden(w, "VERIFY_FAILED", "Invalid verify_token")
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(challenge))
}

// --- Payload types ---

type webhookPayload struct {
	Object string         `json:"object"`
	Entry  []webhookEntry `json:"entry"`
}

type webhookEntry struct {
	ID      string          `json:"id"`
	Changes []webhookChange `json:"changes"`
}

type webhookChange struct {
	Field string       `json:"field"`
	Value webhookValue `json:"value"`
}

type webhookValue struct {
	MessagingProduct string           `json:"messaging_product"`
	Metadata         webhookMetadata  `json:"metadata"`
	Contacts         []webhookContact `json:"contacts"`
	Messages         []webhookMessage `json:"messages"`
	Statuses         []webhookStatus  `json:"statuses"`
}

type webhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type webhookContact struct {
	Profile webhookProfile `json:"profile"`
	WaID    string         `json:"wa_id"`
}

type webhookProfile struct {
	Name string `json:"name"`
}

type webhookMessage struct {
	From      string                `json:"from"`
	ID        string                `json:"id"`
	Timestamp string                `json:"timestamp"`
	Type      string                `json:"type"`
	Text      *webhookText          `json:"text,omitempty"`
	Image     *webhookMedia         `json:"image,omitempty"`
	Document  *webhookDocument      `json:"document,omitempty"`
	Button    *webhookButtonPayload `json:"button,omitempty"`
}

type webhookText struct {
	Body string `json:"body"`
}

type webhookMedia struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
}

type webhookDocument struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
}

type webhookButtonPayload struct {
	Text string `json:"text"`
}

type webhookStatus struct {
	ID           string              `json:"id"`
	Status       string              `json:"status"`
	Timestamp    string              `json:"timestamp"`
	RecipientID  string              `json:"recipient_id"`
	Conversation webhookConversation `json:"conversation"`
}

type webhookConversation struct {
	ID string `json:"id"`
}

// handleEvent processes incoming messages and status updates.
func (h *WebhookHandler) handleEvent(w http.ResponseWriter, r *http.Request) {
	var payload webhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputil.RespondBadRequest(w, "INVALID_PAYLOAD", "Invalid webhook payload: "+err.Error())
		return
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}

			// Process statuses (delivery receipts)
			for _, status := range change.Value.Statuses {
				if err := h.processStatusUpdate(r, status.ID, status.Status, status.Timestamp); err != nil {
					slog.Error("process status update", "error", err)
				}
			}

			// Process incoming messages
			for _, msg := range change.Value.Messages {
				if err := h.processInboundMessage(r, msg, change.Value); err != nil {
					slog.Error("process inbound message", "error", err, "message_id", msg.ID)
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

// processInboundMessage handles a single inbound WhatsApp message.
func (h *WebhookHandler) processInboundMessage(r *http.Request, msg webhookMessage, value webhookValue) error {
	from := msg.From
	text := ""
	if msg.Text != nil {
		text = msg.Text.Body
	}
	msgID := msg.ID
	ts := parseTimestamp(msg.Timestamp)

	// Upsert conversation
	conversationID, err := h.upsertConversation(r, from, value.Metadata.PhoneNumberID, value.Contacts)
	if err != nil {
		return fmt.Errorf("upsert conversation: %w", err)
	}

	// Get contact name
	contactName := ""
	if len(value.Contacts) > 0 {
		contactName = value.Contacts[0].Profile.Name
	}

	// Insert wa_messages row
	err = h.insertMessage(r, conversationID, from, text, msgID, msg.Type, ts, contactName)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	// Chatbot processing
	if text != "" {
		tenant := Tenant{ID: uuid.Nil, Name: "Sandbox School"}

		guardian := Guardian{
			ID:           uuid.Nil,
			FullName:     contactName,
			PhonePrimary: from,
			PhoneWA:      from,
		}

		reply, matched, err := h.chatbot.Handle(r.Context(), tenant, guardian, text)
		if err != nil {
			return fmt.Errorf("chatbot handle: %w", err)
		}

		if matched && reply != "" && h.waClient != nil {
			// Send auto-reply via WhatsApp
			if _, err := h.waClient.SendText(r.Context(), from, reply); err != nil {
				slog.Error("send chatbot reply", "error", err, "from", from)
			}

			// Insert outbound reply as wa_message
			err = h.insertOutboundMessage(r, conversationID, from, reply, msgID+"_reply", time.Now(), contactName)
			if err != nil {
				slog.Error("insert outbound reply", "error", err)
			}
		} else if !matched {
			// Route to inbox: set conversation status to open
			_, err := h.pool.Exec(r.Context(), `
				UPDATE wa_conversations
				SET status = 'open', last_message_at = now()
				WHERE id = $1
			`, conversationID)
			if err != nil {
				slog.Error("set conversation open", "error", err)
			}
		}
	}

	return nil
}

// upsertConversation creates or updates a wa_conversations record.
func (h *WebhookHandler) upsertConversation(r *http.Request, from, phoneNumberID string, contacts []webhookContact) (uuid.UUID, error) {
	contactName := ""
	if len(contacts) > 0 {
		contactName = contacts[0].Profile.Name
	}

	var convID uuid.UUID
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO wa_conversations (tenant_id, wa_contact_phone, wa_contact_name, status, last_message_at)
		SELECT t.id, $1, $2, 'open', now()
		FROM tenants t
		WHERE t.wa_phone_number_id = $3
		ON CONFLICT (tenant_id, wa_contact_phone)
		DO UPDATE SET status = 'open', last_message_at = now(), wa_contact_name = $2
		RETURNING id
	`, from, contactName, phoneNumberID).Scan(&convID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert conversation query: %w", err)
	}
	return convID, nil
}

// insertMessage inserts an inbound WhatsApp message.
func (h *WebhookHandler) insertMessage(r *http.Request, convID uuid.UUID, from, text, msgID, msgType string, ts time.Time, contactName string) error {
	var content map[string]any
	switch msgType {
	case "text":
		content = map[string]any{"text": text}
	case "image":
		content = map[string]any{"image": map[string]any{"id": msgID}}
	case "document":
		content = map[string]any{"document": map[string]any{"id": msgID}}
	case "button":
		content = map[string]any{"button": map[string]any{"text": text}}
	default:
		content = map[string]any{"text": text}
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("marshal content: %w", err)
	}

	_, err = h.pool.Exec(r.Context(), `
		INSERT INTO wa_messages (conversation_id, tenant_id, direction, content_type, content, wa_message_id, status, timestamp)
		SELECT $1, c.tenant_id, 'inbound', $3, $4::jsonb, $5, 'delivered', $6
		FROM wa_conversations c
		WHERE c.id = $1
	`, convID, from, msgType, contentJSON, msgID, ts)
	if err != nil {
		return fmt.Errorf("insert wa_message: %w", err)
	}

	// Update conversation preview
	_, err = h.pool.Exec(r.Context(), `
		UPDATE wa_conversations
		SET last_message_preview = $2, last_message_at = $3
		WHERE id = $1
	`, convID, text, ts)
	if err != nil {
		return fmt.Errorf("update conversation preview: %w", err)
	}

	return nil
}

// insertOutboundMessage inserts an outbound reply (chatbot auto-reply).
func (h *WebhookHandler) insertOutboundMessage(r *http.Request, convID uuid.UUID, to, text, msgID string, ts time.Time, contactName string) error {
	content, err := json.Marshal(map[string]any{"text": text})
	if err != nil {
		return fmt.Errorf("marshal outbound content: %w", err)
	}

	_, err = h.pool.Exec(r.Context(), `
		INSERT INTO wa_messages (conversation_id, tenant_id, direction, content_type, content, wa_message_id, status, timestamp)
		SELECT $1, c.tenant_id, 'outbound', 'text', $3::jsonb, $4, 'sent', $5
		FROM wa_conversations c
		WHERE c.id = $1
	`, convID, to, content, msgID, ts)
	if err != nil {
		return fmt.Errorf("insert outbound wa_message: %w", err)
	}
	return nil
}

// processStatusUpdate updates message status from Meta status webhooks.
func (h *WebhookHandler) processStatusUpdate(r *http.Request, waMessageID, status, timestamp string) error {
	ts := parseTimestamp(timestamp)

	// Update message_logs by provider_message_id
	_, err := h.pool.Exec(r.Context(), `
		UPDATE message_logs
		SET status = $2,
		    delivered_at = CASE WHEN $2 = 'delivered' THEN $3 ELSE delivered_at END,
		    read_at = CASE WHEN $2 = 'read' THEN $3 ELSE read_at END,
		    updated_at = now()
		WHERE provider_message_id = $1
	`, waMessageID, status, ts)
	if err != nil {
		return fmt.Errorf("update message log status: %w", err)
	}

	// Also update wa_messages
	_, err = h.pool.Exec(r.Context(), `
		UPDATE wa_messages
		SET status = $2
		WHERE wa_message_id = $1
	`, waMessageID, status)
	if err != nil {
		return fmt.Errorf("update wa_message status: %w", err)
	}

	return nil
}

// parseTimestamp converts a Unix timestamp string to time.Time.
func parseTimestamp(s string) time.Time {
	// Meta sends Unix seconds
	if sec, err := strconv.ParseInt(s, 10, 64); err == nil && sec > 0 {
		return time.Unix(sec, 0)
	}
	return time.Now()
}
