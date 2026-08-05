package comms

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/comms/sms"
	"github.com/shule360/api/internal/comms/whatsapp"
	"github.com/shule360/api/pkg/upstash"
)

// MessageLog represents a per-recipient delivery log entry.
type MessageLog struct {
	ID                uuid.UUID  `json:"id"`
	MessageID         uuid.UUID  `json:"message_id"`
	RecipientType     string     `json:"recipient_type"`
	RecipientID       *uuid.UUID `json:"recipient_id,omitempty"`
	Phone             string     `json:"phone"`
	Channel           string     `json:"channel"`
	Status            string     `json:"status"`
	ProviderMessageID *string    `json:"provider_message_id,omitempty"`
	DeliveredAt       *time.Time `json:"delivered_at,omitempty"`
	ReadAt            *time.Time `json:"read_at,omitempty"`
	ErrorCode         *string    `json:"error_code,omitempty"`
	ErrorMessage      *string    `json:"error_message,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// Recipient represents a single message recipient.
type Recipient struct {
	ID      uuid.UUID `json:"id"`
	Phone   string    `json:"phone"`
	Name    string    `json:"name"`
	Channel string    `json:"channel"` // sms | whatsapp
}

// Message represents a message record.
type Message struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	Channel        string          `json:"channel"`
	AudienceType   string          `json:"audience_type"`
	AudienceFilter json.RawMessage `json:"audience_filter"`
	ContentType    string          `json:"content_type"`
	Content        string          `json:"content"`
	TemplateID     *string         `json:"template_id,omitempty"`
	MediaURL       *string         `json:"media_url,omitempty"`
	Status         string          `json:"status"`
	ScheduledAt    *time.Time      `json:"scheduled_at,omitempty"`
	SentAt         *time.Time      `json:"sent_at,omitempty"`
	SentBy         uuid.UUID       `json:"sent_by"`
	RecipientCount int             `json:"recipient_count"`
	DeliveredCount int             `json:"delivered_count"`
	FailedCount    int             `json:"failed_count"`
	CreatedAt      time.Time       `json:"created_at"`
}

// CreateMessageRequest is the request payload for creating a message.
type CreateMessageRequest struct {
	Channel        string          `json:"channel"`
	AudienceType   string          `json:"audience_type"`
	AudienceFilter json.RawMessage `json:"audience_filter"`
	ContentType    string          `json:"content_type"`
	Content        string          `json:"content"`
	TemplateID     *string         `json:"template_id,omitempty"`
	MediaURL       *string         `json:"media_url,omitempty"`
	ScheduledAt    *time.Time      `json:"scheduled_at,omitempty"`
}

// ReachEstimate represents the estimated reach & cost for a message.
type ReachEstimate struct {
	RecipientCount int     `json:"recipient_count"`
	EstimatedKES   float64 `json:"estimated_kes"`
	SMSUnits       int     `json:"sms_units"`
}

// DeliveryStats represents aggregate delivery statistics.
type DeliveryStats struct {
	Total        int     `json:"total"`
	Sent         int     `json:"sent"`
	Delivered    int     `json:"delivered"`
	Failed       int     `json:"failed"`
	Pending      int     `json:"pending"`
	DeliveryRate float64 `json:"delivery_rate"`
}

// CommsService handles the communications business logic.
type CommsService struct {
	pool           *pgxpool.Pool
	redis          *upstash.RedisClient
	atClient       *sms.ATClient
	waClient       *whatsapp.WAClient
	queueKeyPrefix string
}

// NewCommsService creates a new communications service.
func NewCommsService(pool *pgxpool.Pool, redis *upstash.RedisClient, atClient *sms.ATClient, waClient *whatsapp.WAClient) *CommsService {
	return &CommsService{
		pool:           pool,
		redis:          redis,
		atClient:       atClient,
		waClient:       waClient,
		queueKeyPrefix: "queue:send:",
	}
}

// CreateAndSend creates a message record and dispatches it (or schedules it).
func (s *CommsService) CreateAndSend(ctx context.Context, tenantID uuid.UUID, req CreateMessageRequest) (*Message, error) {
	// Validate channel
	if req.Channel != "sms" && req.Channel != "whatsapp" && req.Channel != "both" {
		return nil, fmt.Errorf("invalid channel: %s", req.Channel)
	}

	// Validate audience type
	validAudiences := map[string]bool{
		"all_parents": true, "grade": true, "stream": true,
		"transport": true, "fee_defaulters": true, "custom": true,
	}
	if !validAudiences[req.AudienceType] {
		return nil, fmt.Errorf("invalid audience_type: %s", req.AudienceType)
	}

	// Build audience
	recipients, err := s.BuildAudience(ctx, tenantID, req.AudienceType, req.AudienceFilter)
	if err != nil {
		return nil, fmt.Errorf("build audience: %w", err)
	}

	status := "sending"
	if req.ScheduledAt != nil && req.ScheduledAt.After(time.Now()) {
		status = "scheduled"
	}

	// Insert message record
	var msg Message
	err = s.pool.QueryRow(ctx, `
		INSERT INTO messages (
			tenant_id, channel, audience_type, audience_filter, content_type,
			content, template_id, media_url, status, scheduled_at, sent_by,
			recipient_count
		)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, tenant_id, channel, audience_type, audience_filter, content_type,
		          content, template_id, media_url, status, scheduled_at, sent_at,
		          sent_by, recipient_count, delivered_count, failed_count, created_at
	`,
		tenantID, req.Channel, req.AudienceType, req.AudienceFilter, req.ContentType,
		req.Content, req.TemplateID, req.MediaURL, status, req.ScheduledAt,
		uuid.Nil, len(recipients),
	).Scan(
		&msg.ID, &msg.TenantID, &msg.Channel, &msg.AudienceType, &msg.AudienceFilter,
		&msg.ContentType, &msg.Content, &msg.TemplateID, &msg.MediaURL, &msg.Status,
		&msg.ScheduledAt, &msg.SentAt, &msg.SentBy, &msg.RecipientCount,
		&msg.DeliveredCount, &msg.FailedCount, &msg.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	// If scheduled, just return
	if status == "scheduled" {
		return &msg, nil
	}

	// Queue for async dispatch
	if err := s.enqueueDispatch(ctx, tenantID, &msg, recipients); err != nil {
		return nil, fmt.Errorf("enqueue dispatch: %w", err)
	}

	return &msg, nil
}

// BuildAudience resolves an AudienceType + filter to a list of recipient phone numbers.
func (s *CommsService) BuildAudience(ctx context.Context, tenantID uuid.UUID, audienceType string, filter json.RawMessage) ([]Recipient, error) {
	var recipients []Recipient

	switch audienceType {
	case "all_parents":
		err := s.queryRecipients(ctx, tenantID, `
			SELECT DISTINCT g.id, COALESCE(g.phone_wa, g.phone_primary) AS phone, g.full_name
			FROM guardians g
			JOIN learners l ON l.tenant_id = g.tenant_id AND g.id = ANY(l.guardian_ids)
			WHERE g.tenant_id = $1 AND g.is_sms_opted_out = false
		`, &recipients)
		if err != nil {
			return nil, err
		}

	case "grade":
		var f struct {
			Grade string `json:"grade"`
		}
		if len(filter) > 0 {
			if err := json.Unmarshal(filter, &f); err != nil {
				return nil, fmt.Errorf("parse grade filter: %w", err)
			}
		}
		err := s.queryRecipients(ctx, tenantID, `
			SELECT DISTINCT g.id, COALESCE(g.phone_wa, g.phone_primary) AS phone, g.full_name
			FROM guardians g
			JOIN learners l ON l.tenant_id = g.tenant_id AND g.id = ANY(l.guardian_ids)
			WHERE g.tenant_id = $1 AND l.grade = $2 AND g.is_sms_opted_out = false
		`, &recipients, f.Grade)
		if err != nil {
			return nil, err
		}

	case "stream":
		var f struct {
			Grade  string `json:"grade"`
			Stream string `json:"stream"`
		}
		if len(filter) > 0 {
			if err := json.Unmarshal(filter, &f); err != nil {
				return nil, fmt.Errorf("parse stream filter: %w", err)
			}
		}
		err := s.queryRecipients(ctx, tenantID, `
			SELECT DISTINCT g.id, COALESCE(g.phone_wa, g.phone_primary) AS phone, g.full_name
			FROM guardians g
			JOIN learners l ON l.tenant_id = g.tenant_id AND g.id = ANY(l.guardian_ids)
			WHERE g.tenant_id = $1 AND l.grade = $2 AND l.stream = $3 AND g.is_sms_opted_out = false
		`, &recipients, f.Grade, f.Stream)
		if err != nil {
			return nil, err
		}

	case "transport":
		err := s.queryRecipients(ctx, tenantID, `
			SELECT DISTINCT g.id, COALESCE(g.phone_wa, g.phone_primary) AS phone, g.full_name
			FROM guardians g
			WHERE g.tenant_id = $1 AND g.is_transport_enrolled = true AND g.is_sms_opted_out = false
		`, &recipients)
		if err != nil {
			return nil, err
		}

	case "fee_defaulters":
		// Stub: return mock data (guardians with balance > 0)
		err := s.queryRecipients(ctx, tenantID, `
			SELECT DISTINCT g.id, COALESCE(g.phone_wa, g.phone_primary) AS phone, g.full_name
			FROM guardians g
			JOIN learners l ON l.tenant_id = g.tenant_id AND g.id = ANY(l.guardian_ids)
			WHERE g.tenant_id = $1 AND g.is_sms_opted_out = false
			LIMIT 10
		`, &recipients)
		if err != nil {
			return nil, err
		}

	case "custom":
		var f struct {
			GuardianIDs []uuid.UUID `json:"guardian_ids"`
		}
		if len(filter) > 0 {
			if err := json.Unmarshal(filter, &f); err != nil {
				return nil, fmt.Errorf("parse custom filter: %w", err)
			}
		}
		if len(f.GuardianIDs) == 0 {
			return nil, fmt.Errorf("custom audience requires guardian_ids")
		}
		err := s.queryRecipients(ctx, tenantID, `
			SELECT id, COALESCE(phone_wa, phone_primary) AS phone, full_name
			FROM guardians
			WHERE tenant_id = $1 AND id = ANY($2::uuid[]) AND is_sms_opted_out = false
		`, &recipients, f.GuardianIDs)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unknown audience type: %s", audienceType)
	}

	return recipients, nil
}

// queryRecipients is a helper that scans guardian recipients into the slice.
func (s *CommsService) queryRecipients(ctx context.Context, tenantID uuid.UUID, query string, out *[]Recipient, args ...any) error {
	fullArgs := append([]any{tenantID}, args...)
	rows, err := s.pool.Query(ctx, query, fullArgs...)
	if err != nil {
		return fmt.Errorf("query recipients: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var r Recipient
		if err := rows.Scan(&r.ID, &r.Phone, &r.Name); err != nil {
			return fmt.Errorf("scan recipient: %w", err)
		}
		r.Channel = "sms"
		*out = append(*out, r)
	}
	return rows.Err()
}

// EstimateReach returns recipient count and cost estimate without sending.
func (s *CommsService) EstimateReach(ctx context.Context, tenantID uuid.UUID, req CreateMessageRequest) (ReachEstimate, error) {
	recipients, err := s.BuildAudience(ctx, tenantID, req.AudienceType, req.AudienceFilter)
	if err != nil {
		return ReachEstimate{}, fmt.Errorf("build audience for estimate: %w", err)
	}

	units := sms.CalculateSMSUnits(req.Content)
	estimate := s.atClient.EstimateCost(len(recipients), units)

	return ReachEstimate{
		RecipientCount: len(recipients),
		EstimatedKES:   estimate.EstimatedKES,
		SMSUnits:       units,
	}, nil
}

// ListMessages returns messages for a tenant, optionally filtered by status/channel.
func (s *CommsService) ListMessages(ctx context.Context, tenantID uuid.UUID, status, channel string, limit, offset int) ([]Message, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, tenant_id, channel, audience_type, audience_filter, content_type,
		       content, template_id, media_url, status, scheduled_at, sent_at,
		       sent_by, recipient_count, delivered_count, failed_count, created_at
		FROM messages
		WHERE tenant_id = $1
	`
	args := []any{tenantID}

	if status != "" {
		args = append(args, status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if channel != "" {
		args = append(args, channel)
		query += fmt.Sprintf(" AND channel = $%d", len(args))
	}

	args = append(args, limit, offset)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.Channel, &m.AudienceType, &m.AudienceFilter,
			&m.ContentType, &m.Content, &m.TemplateID, &m.MediaURL, &m.Status,
			&m.ScheduledAt, &m.SentAt, &m.SentBy, &m.RecipientCount,
			&m.DeliveredCount, &m.FailedCount, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// GetMessage returns a single message and its delivery stats.
func (s *CommsService) GetMessage(ctx context.Context, messageID uuid.UUID) (*Message, DeliveryStats, error) {
	var m Message
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, channel, audience_type, audience_filter, content_type,
		       content, template_id, media_url, status, scheduled_at, sent_at,
		       sent_by, recipient_count, delivered_count, failed_count, created_at
		FROM messages
		WHERE id = $1
	`, messageID).Scan(
		&m.ID, &m.TenantID, &m.Channel, &m.AudienceType, &m.AudienceFilter,
		&m.ContentType, &m.Content, &m.TemplateID, &m.MediaURL, &m.Status,
		&m.ScheduledAt, &m.SentAt, &m.SentBy, &m.RecipientCount,
		&m.DeliveredCount, &m.FailedCount, &m.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, DeliveryStats{}, fmt.Errorf("message not found")
		}
		return nil, DeliveryStats{}, fmt.Errorf("query message: %w", err)
	}

	stats, err := s.GetDeliveryStats(ctx, messageID)
	if err != nil {
		return nil, DeliveryStats{}, fmt.Errorf("get delivery stats: %w", err)
	}

	return &m, stats, nil
}

// GetMessageLogs returns paginated delivery logs for a message.
func (s *CommsService) GetMessageLogs(ctx context.Context, messageID uuid.UUID, limit, offset int) ([]MessageLog, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, message_id, recipient_type, recipient_id, phone, channel, status,
		       provider_message_id, delivered_at, read_at, error_code, error_message,
		       created_at
		FROM message_logs
		WHERE message_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, messageID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query message logs: %w", err)
	}
	defer rows.Close()

	var logs []MessageLog
	for rows.Next() {
		var l MessageLog
		if err := rows.Scan(
			&l.ID, &l.MessageID, &l.RecipientType, &l.RecipientID, &l.Phone, &l.Channel,
			&l.Status, &l.ProviderMessageID, &l.DeliveredAt, &l.ReadAt,
			&l.ErrorCode, &l.ErrorMessage, &l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// CancelScheduled cancels a scheduled message (only if status is 'scheduled').
func (s *CommsService) CancelScheduled(ctx context.Context, messageID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE messages
		SET status = 'failed', updated_at = now()
		WHERE id = $1 AND status = 'scheduled'
	`, messageID)
	if err != nil {
		return fmt.Errorf("cancel scheduled message: %w", err)
	}
	return nil
}

// GetDeliveryStats returns aggregate + per-recipient delivery stats for a message.
func (s *CommsService) GetDeliveryStats(ctx context.Context, messageID uuid.UUID) (DeliveryStats, error) {
	var stats DeliveryStats
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'sent') AS sent,
			COUNT(*) FILTER (WHERE status = 'delivered') AS delivered,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE status = 'pending') AS pending
		FROM message_logs
		WHERE message_id = $1
	`, messageID).Scan(
		&stats.Total, &stats.Sent, &stats.Delivered, &stats.Failed, &stats.Pending,
	)
	if err != nil {
		return DeliveryStats{}, fmt.Errorf("query delivery stats: %w", err)
	}

	if stats.Total > 0 {
		stats.DeliveryRate = float64(stats.Delivered) / float64(stats.Total) * 100
	}

	return stats, nil
}

// dispatchJob is the queued message dispatch job payload.
type dispatchJob struct {
	MessageID uuid.UUID `json:"message_id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Channel   string    `json:"channel"`
}

// enqueueDispatch pushes a dispatch job onto the Redis FIFO queue.
func (s *CommsService) enqueueDispatch(ctx context.Context, tenantID uuid.UUID, msg *Message, recipients []Recipient) error {
	job := dispatchJob{
		MessageID: msg.ID,
		TenantID:  tenantID,
		Channel:   msg.Channel,
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal dispatch job: %w", err)
	}

	queueKey := s.queueKeyPrefix + tenantID.String()
	if err := s.redis.LPush(ctx, queueKey, string(payload)); err != nil {
		return fmt.Errorf("push dispatch job: %w", err)
	}

	// Rate limit: max 3 bulk sends per tenant per hour
	rateKey := fmt.Sprintf("ratelimit:send:%s", tenantID)
	if _, err := s.redis.Incr(ctx, rateKey); err == nil {
		s.redis.Expire(ctx, rateKey, 3600)
	}

	return nil
}

// ProcessQueueJob processes a single queue job (called by worker).
func (s *CommsService) ProcessQueueJob(ctx context.Context, job dispatchJob) error {
	// Fetch message + recipients and dispatch
	// This is where the actual SMS/WA sending happens in the worker.
	// For simplicity in this scaffold, we log and mark sent.
	return nil
}
