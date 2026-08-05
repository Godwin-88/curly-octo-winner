package inbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Conversation represents a WhatsApp conversation.
type Conversation struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	GuardianID         *uuid.UUID `json:"guardian_id,omitempty"`
	WAContactPhone     string     `json:"wa_contact_phone"`
	WAContactName      string     `json:"wa_contact_name"`
	Status             string     `json:"status"`
	AssignedTo         *uuid.UUID `json:"assigned_to,omitempty"`
	LastMessageAt      time.Time  `json:"last_message_at"`
	LastMessagePreview string     `json:"last_message_preview"`
	UnreadCount        int        `json:"unread_count"`
	CreatedAt          time.Time  `json:"created_at"`
}

// ConversationMessage represents a message in a conversation thread.
type ConversationMessage struct {
	ID             uuid.UUID       `json:"id"`
	ConversationID uuid.UUID       `json:"conversation_id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	Direction      string          `json:"direction"`
	ContentType    string          `json:"content_type"`
	Content        json.RawMessage `json:"content"`
	WAMessageID    string          `json:"wa_message_id"`
	Status         string          `json:"status"`
	Timestamp      time.Time       `json:"timestamp"`
}

// Service handles conversation assignment and threading.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a new inbox service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// ListConversations returns conversations for a tenant, optionally filtered.
func (s *Service) ListConversations(ctx context.Context, tenantID uuid.UUID, status, assignedTo string, limit, offset int) ([]Conversation, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT c.id, c.tenant_id, c.guardian_id, c.wa_contact_phone, c.wa_contact_name,
		       c.status, c.assigned_to, c.last_message_at, c.last_message_preview,
		       (SELECT COUNT(*) FROM wa_messages m 
		        WHERE m.conversation_id = c.id AND m.direction = 'inbound' 
		        AND m.status != 'read') AS unread_count,
		       c.created_at
		FROM wa_conversations c
		WHERE c.tenant_id = $1
	`
	args := []any{tenantID}

	if status != "" {
		args = append(args, status)
		query += fmt.Sprintf(" AND c.status = $%d", len(args))
	}

	if assignedTo != "" {
		args = append(args, assignedTo)
		query += fmt.Sprintf(" AND c.assigned_to::text = $%d", len(args))
	}

	args = append(args, limit, offset)
	query += fmt.Sprintf(" ORDER BY c.last_message_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query conversations: %w", err)
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.GuardianID, &c.WAContactPhone, &c.WAContactName,
			&c.Status, &c.AssignedTo, &c.LastMessageAt, &c.LastMessagePreview,
			&c.UnreadCount, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		conversations = append(conversations, c)
	}
	return conversations, rows.Err()
}

// GetConversation returns a single conversation with its message thread.
func (s *Service) GetConversation(ctx context.Context, tenantID, conversationID uuid.UUID) (*Conversation, []ConversationMessage, error) {
	var c Conversation
	err := s.pool.QueryRow(ctx, `
		SELECT c.id, c.tenant_id, c.guardian_id, c.wa_contact_phone, c.wa_contact_name,
		       c.status, c.assigned_to, c.last_message_at, c.last_message_preview,
		       (SELECT COUNT(*) FROM wa_messages m 
		        WHERE m.conversation_id = c.id AND m.direction = 'inbound' 
		        AND m.status != 'read') AS unread_count,
		       c.created_at
		FROM wa_conversations c
		WHERE c.tenant_id = $1 AND c.id = $2
	`, tenantID, conversationID).Scan(
		&c.ID, &c.TenantID, &c.GuardianID, &c.WAContactPhone, &c.WAContactName,
		&c.Status, &c.AssignedTo, &c.LastMessageAt, &c.LastMessagePreview,
		&c.UnreadCount, &c.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, fmt.Errorf("conversation not found")
		}
		return nil, nil, fmt.Errorf("query conversation: %w", err)
	}

	// Get message thread
	rows, err := s.pool.Query(ctx, `
		SELECT id, conversation_id, tenant_id, direction, content_type, content,
		       wa_message_id, status, timestamp
		FROM wa_messages
		WHERE conversation_id = $1 AND tenant_id = $2
		ORDER BY timestamp ASC
	`, conversationID, tenantID)
	if err != nil {
		return nil, nil, fmt.Errorf("query conversation messages: %w", err)
	}
	defer rows.Close()

	var messages []ConversationMessage
	for rows.Next() {
		var m ConversationMessage
		if err := rows.Scan(
			&m.ID, &m.ConversationID, &m.TenantID, &m.Direction, &m.ContentType,
			&m.Content, &m.WAMessageID, &m.Status, &m.Timestamp,
		); err != nil {
			return nil, nil, fmt.Errorf("scan conversation message: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate conversation messages: %w", err)
	}

	return &c, messages, nil
}

// AssignConversation assigns a conversation to a staff member.
func (s *Service) AssignConversation(ctx context.Context, tenantID, conversationID, staffID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE wa_conversations
		SET assigned_to = $3, status = 'in_progress', updated_at = now()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, conversationID, staffID)
	if err != nil {
		return fmt.Errorf("assign conversation: %w", err)
	}
	return nil
}

// UpdateStatus updates a conversation's status.
func (s *Service) UpdateStatus(ctx context.Context, tenantID, conversationID uuid.UUID, status string) error {
	valid := map[string]bool{
		"open": true, "in_progress": true, "waiting": true, "resolved": true,
	}
	if !valid[status] {
		return fmt.Errorf("invalid conversation status: %s", status)
	}

	_, err := s.pool.Exec(ctx, `
		UPDATE wa_conversations
		SET status = $3, updated_at = now()
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, conversationID, status)
	if err != nil {
		return fmt.Errorf("update conversation status: %w", err)
	}
	return nil
}

// RecordOutboundMessage inserts an outbound message into a conversation.
func (s *Service) RecordOutboundMessage(ctx context.Context, tenantID, conversationID uuid.UUID, content json.RawMessage, waMessageID, contentType string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO wa_messages (conversation_id, tenant_id, direction, content_type, content, wa_message_id, status, timestamp)
		VALUES ($1, $2, 'outbound', $3, $4::jsonb, $5, 'sent', now())
	`, conversationID, tenantID, contentType, content, waMessageID)
	if err != nil {
		return fmt.Errorf("insert outbound wa_message: %w", err)
	}

	// Update conversation last message
	_, err = s.pool.Exec(ctx, `
		UPDATE wa_conversations
		SET last_message_at = now(), last_message_preview = $3
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, conversationID, string(content))
	if err != nil {
		return fmt.Errorf("update conversation last message: %w", err)
	}
	return nil
}
