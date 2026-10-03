package comms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/comms/contacts"
	"github.com/shule360/api/internal/comms/sms"
	"github.com/shule360/api/pkg/pgxutil"
)

// MessageLog represents a per-recipient delivery log entry.
type MessageLog struct {
	ID                uuid.UUID  `json:"id"`
	MessageID         uuid.UUID  `json:"message_id"`
	RecipientType     string     `json:"recipient_type"`
	RecipientID       *uuid.UUID `json:"recipient_id,omitempty"`
	RecipientName     *string    `json:"recipient_name,omitempty"`
	Phone             string     `json:"phone"`
	Channel           string     `json:"channel"`
	Status            string     `json:"status"`
	ProviderMessageID *string    `json:"provider_message_id,omitempty"`
	SentAt            *time.Time `json:"sent_at,omitempty"`
	DeliveredAt       *time.Time `json:"delivered_at,omitempty"`
	ReadAt            *time.Time `json:"read_at,omitempty"`
	CostCents         *int       `json:"cost_cents,omitempty"`
	ErrorCode         *string    `json:"error_code,omitempty"`
	ErrorMessage      *string    `json:"error_message,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// Recipient represents a single message recipient.
type Recipient struct {
	ID      uuid.UUID `json:"id"`
	Type    string    `json:"type"` // guardian | contact
	Phone   string    `json:"phone"`
	Name    string    `json:"name"`
	Channel string    `json:"channel"` // sms
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
	SentBy         *uuid.UUID      `json:"sent_by,omitempty"`
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
	// IdempotencyKey is chosen by the client once per send attempt. A second
	// request with the same key returns the first message instead of sending
	// (and billing) again.
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// ReachEstimate represents the estimated reach & cost for a message.
type ReachEstimate struct {
	RecipientCount int     `json:"recipient_count"`
	EstimatedKES   float64 `json:"estimated_kes"`
	SMSUnits       int     `json:"sms_units"`
	// InvalidCount is how many people in the audience have a number that
	// cannot be texted; they are listed as failed on the message, not billed.
	InvalidCount int    `json:"invalid_count"`
	Encoding     string `json:"encoding"`
	Characters   int    `json:"characters"`
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

// Actor is who asked for a message to be sent: a member of the school's staff,
// or a platform/group user working inside the school. Exactly one is set for a
// request made through the API.
type Actor struct {
	StaffID    *uuid.UUID
	OperatorID *uuid.UUID
}

// ErrNotFound is returned when a message does not exist in the caller's school.
var ErrNotFound = errors.New("message not found")

// ValidationError is a refusal the caller can fix: it is shown to the user as
// written.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

// smsRateKES is the approximate price of one SMS unit in Kenya, used for the
// estimate shown before sending. What was actually charged is recorded per
// recipient from the provider's response.
const smsRateKES = 0.80

// CommsService handles the communications business logic.
type CommsService struct {
	pool       *pgxpool.Pool
	dispatcher *Dispatcher
}

// NewCommsService creates a new communications service. The dispatcher may be
// nil (tests of the read paths); a message is then recorded but waits for the
// next sweep of whichever process runs one.
func NewCommsService(pool *pgxpool.Pool, dispatcher *Dispatcher) *CommsService {
	return &CommsService{pool: pool, dispatcher: dispatcher}
}

const messageColumns = `id, tenant_id, channel, audience_type, audience_filter, content_type,
	content, template_id, media_url, status, scheduled_at, sent_at,
	sent_by, recipient_count, delivered_count, failed_count, created_at`

func scanMessage(row pgx.Row) (*Message, error) {
	var m Message
	err := row.Scan(
		&m.ID, &m.TenantID, &m.Channel, &m.AudienceType, &m.AudienceFilter,
		&m.ContentType, &m.Content, &m.TemplateID, &m.MediaURL, &m.Status,
		&m.ScheduledAt, &m.SentAt, &m.SentBy, &m.RecipientCount,
		&m.DeliveredCount, &m.FailedCount, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// CreateAndSend records a message and every one of its recipients, then hands
// it to the dispatcher (or leaves it for its scheduled time).
//
// Nothing is sent from here. The message and one `pending` log row per
// recipient are committed first, so there is always a record of who was meant
// to receive it before the provider is called.
func (s *CommsService) CreateAndSend(ctx context.Context, tenantID uuid.UUID, actor Actor, req CreateMessageRequest) (*Message, error) {
	if req.Channel == "" {
		req.Channel = "sms"
	}
	if req.Channel != "sms" {
		return nil, invalid("Only SMS can be sent at the moment. WhatsApp sending is not available yet.")
	}
	if req.ContentType == "" {
		req.ContentType = "text"
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return nil, invalid("Write the message before sending.")
	}
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if len(req.IdempotencyKey) > 100 {
		return nil, invalid("idempotency_key is longer than 100 characters.")
	}

	// A repeat of a request that already created a message returns that
	// message. Checked before the audience is resolved so a retry is cheap.
	if req.IdempotencyKey != "" {
		if existing, err := s.findByIdempotencyKey(ctx, tenantID, req.IdempotencyKey); err != nil {
			return nil, err
		} else if existing != nil {
			return existing, nil
		}
	}

	prepared, err := s.prepare(ctx, tenantID, req.AudienceType, req.AudienceFilter, req.Content)
	if err != nil {
		return nil, err
	}
	if len(prepared.valid) == 0 {
		if len(prepared.invalid) > 0 {
			return nil, invalid("No one in this audience has a phone number that can be texted (%d invalid).", len(prepared.invalid))
		}
		return nil, invalid("No one matches this audience, so there is nobody to send to.")
	}

	status := "sending"
	if req.ScheduledAt != nil && req.ScheduledAt.After(time.Now()) {
		status = "scheduled"
	} else {
		req.ScheduledAt = nil
	}
	if len(req.AudienceFilter) == 0 {
		req.AudienceFilter = json.RawMessage(`{}`)
	}

	msg, err := s.insertMessage(ctx, tenantID, actor, req, status, prepared)
	if err != nil {
		// Lost a race with an identical request: return the winner.
		var pgErr *pgconn.PgError
		if req.IdempotencyKey != "" && errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_messages_idempotency" {
			if existing, findErr := s.findByIdempotencyKey(ctx, tenantID, req.IdempotencyKey); findErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}

	if status == "sending" && s.dispatcher != nil {
		s.dispatcher.Notify()
	}
	return msg, nil
}

func (s *CommsService) findByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, key string) (*Message, error) {
	msg, err := scanMessage(s.pool.QueryRow(ctx, `SELECT `+messageColumns+`
		FROM messages WHERE tenant_id = $1 AND idempotency_key = $2`, tenantID, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("look up idempotency key: %w", err)
	}
	return msg, nil
}

// insertMessage writes the message and its recipient rows in one transaction.
func (s *CommsService) insertMessage(ctx context.Context, tenantID uuid.UUID, actor Actor, req CreateMessageRequest, status string, prepared *preparedAudience) (*Message, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var key any
	if req.IdempotencyKey != "" {
		key = req.IdempotencyKey
	}
	msg, err := scanMessage(tx.QueryRow(ctx, `
		INSERT INTO messages (
			tenant_id, channel, audience_type, audience_filter, content_type,
			content, template_id, media_url, status, scheduled_at, sent_by,
			recipient_count, failed_count, idempotency_key, sent_by_operator
		)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING `+messageColumns,
		tenantID, req.Channel, req.AudienceType, string(req.AudienceFilter), req.ContentType,
		req.Content, req.TemplateID, req.MediaURL, status, req.ScheduledAt, actor.StaffID,
		len(prepared.valid)+len(prepared.invalid), len(prepared.invalid), key, actor.OperatorID,
	))
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	batch := &pgx.Batch{}
	const insertLog = `
		INSERT INTO message_logs (
			tenant_id, message_id, recipient_type, recipient_id, recipient_name,
			phone, channel, status, rendered_content, error_code, error_message
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'sms', $7, $8, $9, $10)`
	for _, r := range prepared.valid {
		batch.Queue(insertLog, tenantID, msg.ID, r.Type, nullableID(r.ID), r.Name, r.Phone, "pending", r.rendered, nil, nil)
	}
	for _, r := range prepared.invalid {
		// Recorded, never attempted: the school sees exactly who was skipped
		// and why, instead of a recipient count that quietly shrinks.
		batch.Queue(insertLog, tenantID, msg.ID, r.Type, nullableID(r.ID), r.Name, truncatePhone(r.Phone), "failed", nil, "INVALID_PHONE", r.problem)
	}
	results := tx.SendBatch(ctx, batch)
	for range prepared.valid {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return nil, fmt.Errorf("insert recipient: %w", err)
		}
	}
	for range prepared.invalid {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return nil, fmt.Errorf("insert skipped recipient: %w", err)
		}
	}
	if err := results.Close(); err != nil {
		return nil, fmt.Errorf("insert recipients: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return msg, nil
}

func nullableID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

// truncatePhone keeps an unusable number inside the column (VARCHAR(15)) while
// leaving enough of it for the school to recognise whose it is.
func truncatePhone(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "(none)"
	}
	if len(raw) > 15 {
		return raw[:15]
	}
	return raw
}

// preparedRecipient is a recipient ready to be recorded.
type preparedRecipient struct {
	Recipient
	rendered *string // personalised text; nil when the message has no variables
	problem  string  // why the number cannot be texted
}

type preparedAudience struct {
	valid    []preparedRecipient
	invalid  []preparedRecipient
	segments sms.Segments // the longest text any recipient will receive
}

// prepare resolves an audience into the exact rows a send will record:
// numbers normalised to E.164, one row per number, the text personalised, and
// the length checked against what will really be billed.
func (s *CommsService) prepare(ctx context.Context, tenantID uuid.UUID, audienceType string, filter json.RawMessage, content string) (*preparedAudience, error) {
	recipients, err := s.BuildAudience(ctx, tenantID, audienceType, filter)
	if err != nil {
		return nil, err
	}

	out := &preparedAudience{}
	seen := make(map[string]struct{}, len(recipients))
	for _, r := range recipients {
		phone, err := contacts.NormalizePhone(r.Phone)
		if err != nil {
			key := "invalid:" + r.Type + ":" + r.ID.String()
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			out.invalid = append(out.invalid, preparedRecipient{Recipient: r, problem: err.Error()})
			continue
		}
		// One SMS per number: a parent of three learners, or a guardian who is
		// also in the contact book, is texted once.
		if _, dup := seen[phone]; dup {
			continue
		}
		seen[phone] = struct{}{}
		r.Phone = phone
		out.valid = append(out.valid, preparedRecipient{Recipient: r})
	}

	if err := s.personalise(ctx, tenantID, content, out); err != nil {
		return nil, err
	}
	if out.segments.Units > sms.MaxSMSUnits {
		return nil, invalid("This message is %d SMS units long (%s, %d characters). The limit is %d units; shorten it.",
			out.segments.Units, out.segments.Encoding, out.segments.Length, sms.MaxSMSUnits)
	}
	return out, nil
}

// BuildAudience resolves an AudienceType + filter to a list of recipient phone numbers.
func (s *CommsService) BuildAudience(ctx context.Context, tenantID uuid.UUID, audienceType string, filter json.RawMessage) ([]Recipient, error) {
	var recipients []Recipient

	// SMS goes to the guardian's primary number. (This used to prefer the
	// WhatsApp number, which a guardian may not carry as a SIM at all.)
	switch audienceType {
	case "all_parents":
		err := s.queryRecipients(ctx, tenantID, `
			SELECT DISTINCT g.id, g.phone_primary AS phone, g.full_name
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
			SELECT DISTINCT g.id, g.phone_primary AS phone, g.full_name
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
			SELECT DISTINCT g.id, g.phone_primary AS phone, g.full_name
			FROM guardians g
			JOIN learners l ON l.tenant_id = g.tenant_id AND g.id = ANY(l.guardian_ids)
			WHERE g.tenant_id = $1 AND l.grade = $2 AND l.stream = $3 AND g.is_sms_opted_out = false
		`, &recipients, f.Grade, f.Stream)
		if err != nil {
			return nil, err
		}

	case "transport":
		err := s.queryRecipients(ctx, tenantID, `
			SELECT DISTINCT g.id, g.phone_primary AS phone, g.full_name
			FROM guardians g
			WHERE g.tenant_id = $1 AND g.is_transport_enrolled = true AND g.is_sms_opted_out = false
		`, &recipients)
		if err != nil {
			return nil, err
		}

	case "fee_defaulters":
		// Guardians of learners with an outstanding balance on a live invoice
		// (not draft, not void, and not fully paid). Previously this was a
		// "stub" that returned the first 10 guardians of the school, which made
		// the reach estimate lie to the bursar.
		err := s.queryRecipients(ctx, tenantID, `
			SELECT DISTINCT g.id, g.phone_primary AS phone, g.full_name
			FROM guardians g
			JOIN learners l ON l.tenant_id = g.tenant_id AND g.id = ANY(l.guardian_ids)
			JOIN invoices i ON i.tenant_id = l.tenant_id AND i.learner_id = l.id
			WHERE g.tenant_id = $1
			  AND g.is_sms_opted_out = false
			  AND i.status NOT IN ('draft', 'void', 'paid')
			  AND (i.total_cents - i.discount_cents - i.paid_cents) > 0
		`, &recipients)
		if err != nil {
			return nil, err
		}

	case "contacts":
		// People from the school's contact book (Communications → Contacts).
		// Opted-out and archived contacts are never messaged, and the optional
		// tag filter is how a school targets a segment of that book.
		var f struct {
			Tag string `json:"tag"`
		}
		if len(filter) > 0 {
			if err := json.Unmarshal(filter, &f); err != nil {
				return nil, fmt.Errorf("parse contacts filter: %w", err)
			}
		}
		err := s.queryRecipients(ctx, tenantID, `
			SELECT id, phone, full_name
			FROM contacts
			WHERE tenant_id = $1
			  AND is_active = true
			  AND is_opted_out = false
			  AND phone <> ''
			  AND ($2 = '' OR tags @> ARRAY[$2]::text[])
		`, &recipients, strings.ToLower(strings.TrimSpace(f.Tag)))
		if err != nil {
			return nil, err
		}
		for i := range recipients {
			recipients[i].Type = "contact"
		}

	case "custom":
		var f struct {
			GuardianIDs []string `json:"guardian_ids"`
		}
		if len(filter) > 0 {
			if err := json.Unmarshal(filter, &f); err != nil {
				return nil, fmt.Errorf("parse custom filter: %w", err)
			}
		}
		if len(f.GuardianIDs) == 0 {
			return nil, invalid("Choose at least one guardian for a custom audience.")
		}
		// Validate each id so a bad value reports exactly which entry is wrong
		// instead of a raw pgx/Postgres error.
		ids := make([]uuid.UUID, 0, len(f.GuardianIDs))
		for _, raw := range f.GuardianIDs {
			parsed, err := uuid.Parse(strings.TrimSpace(raw))
			if err != nil {
				return nil, invalid("guardian_ids contains an invalid id: %q", raw)
			}
			ids = append(ids, parsed)
		}
		err := s.queryRecipients(ctx, tenantID, `
			SELECT id, phone_primary AS phone, full_name
			FROM guardians
			WHERE tenant_id = $1 AND id = ANY($2::uuid[]) AND is_sms_opted_out = false
		`, &recipients, pgxutil.UUIDArray(ids))
		if err != nil {
			return nil, err
		}

	default:
		return nil, invalid("Unknown audience %q.", audienceType)
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
		r.Type = "guardian"
		*out = append(*out, r)
	}
	return rows.Err()
}

// EstimateReach returns who a message would reach and what it would cost,
// without recording or sending anything. It runs the same preparation as a
// real send, so the numbers shown before confirming are the numbers billed.
func (s *CommsService) EstimateReach(ctx context.Context, tenantID uuid.UUID, req CreateMessageRequest) (ReachEstimate, error) {
	prepared, err := s.prepare(ctx, tenantID, req.AudienceType, req.AudienceFilter, strings.TrimSpace(req.Content))
	if err != nil {
		return ReachEstimate{}, err
	}
	seg := prepared.segments
	return ReachEstimate{
		RecipientCount: len(prepared.valid),
		EstimatedKES:   float64(len(prepared.valid)*seg.Units) * smsRateKES,
		SMSUnits:       seg.Units,
		InvalidCount:   len(prepared.invalid),
		Encoding:       seg.Encoding,
		Characters:     seg.Length,
	}, nil
}

// ListMessages returns messages for a tenant, optionally filtered by status/channel.
func (s *CommsService) ListMessages(ctx context.Context, tenantID uuid.UUID, status, channel string, limit, offset int) ([]Message, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `SELECT ` + messageColumns + ` FROM messages WHERE tenant_id = $1`
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

	messages := []Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, *m)
	}
	return messages, rows.Err()
}

// GetMessage returns a single message of this school and its delivery stats.
// A message of another school is reported exactly like one that does not exist.
func (s *CommsService) GetMessage(ctx context.Context, tenantID, messageID uuid.UUID) (*Message, DeliveryStats, error) {
	m, err := scanMessage(s.pool.QueryRow(ctx, `SELECT `+messageColumns+`
		FROM messages WHERE tenant_id = $1 AND id = $2`, tenantID, messageID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, DeliveryStats{}, ErrNotFound
		}
		return nil, DeliveryStats{}, fmt.Errorf("query message: %w", err)
	}

	stats, err := s.GetDeliveryStats(ctx, tenantID, messageID)
	if err != nil {
		return nil, DeliveryStats{}, fmt.Errorf("get delivery stats: %w", err)
	}
	return m, stats, nil
}

// GetMessageLogs returns paginated delivery logs for a message of this school.
func (s *CommsService) GetMessageLogs(ctx context.Context, tenantID, messageID uuid.UUID, status string, limit, offset int) ([]MessageLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM messages WHERE tenant_id = $1 AND id = $2)`,
		tenantID, messageID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("query message: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, message_id, recipient_type, recipient_id, recipient_name, phone, channel, status,
		       provider_message_id, sent_at, delivered_at, read_at, cost_cents,
		       error_code, error_message, created_at
		FROM message_logs
		WHERE tenant_id = $1 AND message_id = $2 AND ($3 = '' OR status = $3)
		ORDER BY (status = 'failed') DESC, recipient_name NULLS LAST, phone
		LIMIT $4 OFFSET $5
	`, tenantID, messageID, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query message logs: %w", err)
	}
	defer rows.Close()

	logs := []MessageLog{}
	for rows.Next() {
		var l MessageLog
		if err := rows.Scan(
			&l.ID, &l.MessageID, &l.RecipientType, &l.RecipientID, &l.RecipientName, &l.Phone, &l.Channel,
			&l.Status, &l.ProviderMessageID, &l.SentAt, &l.DeliveredAt, &l.ReadAt, &l.CostCents,
			&l.ErrorCode, &l.ErrorMessage, &l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// CancelScheduled cancels a message that has not started sending. Its
// recipients are marked as not sent.
func (s *CommsService) CancelScheduled(ctx context.Context, tenantID, messageID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	tag, err := tx.Exec(ctx, `
		UPDATE messages
		SET status = 'cancelled', updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND status = 'scheduled'
	`, tenantID, messageID)
	if err != nil {
		return fmt.Errorf("cancel scheduled message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		var status string
		err := tx.QueryRow(ctx, `SELECT status FROM messages WHERE tenant_id = $1 AND id = $2`, tenantID, messageID).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("query message: %w", err)
		}
		return invalid("Only a scheduled message can be cancelled; this one is %s.", status)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE message_logs
		SET status = 'failed', error_code = 'CANCELLED', error_message = 'The message was cancelled before it was sent.', updated_at = now()
		WHERE message_id = $1 AND status = 'pending'
	`, messageID); err != nil {
		return fmt.Errorf("cancel recipients: %w", err)
	}
	if _, err := tx.Exec(ctx, sms.RefreshCountsSQL, messageID); err != nil {
		return fmt.Errorf("refresh counts: %w", err)
	}
	return tx.Commit(ctx)
}

// ResendFailed creates a new message addressed to the recipients a previous
// message failed to reach. Recipients whose outcome is unknown (the provider
// may have accepted them) are only included when includeUncertain is set, so a
// parent is not texted twice by accident.
func (s *CommsService) ResendFailed(ctx context.Context, tenantID uuid.UUID, actor Actor, sourceID uuid.UUID, includeUncertain bool, idempotencyKey string) (*Message, error) {
	source, _, err := s.GetMessage(ctx, tenantID, sourceID)
	if err != nil {
		return nil, err
	}
	if source.Status != "sent" && source.Status != "failed" {
		return nil, invalid("This message is %s; there is nothing to resend yet.", source.Status)
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey != "" {
		if existing, err := s.findByIdempotencyKey(ctx, tenantID, idempotencyKey); err != nil {
			return nil, err
		} else if existing != nil {
			return existing, nil
		}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT recipient_type, recipient_id, COALESCE(recipient_name, ''), phone, rendered_content
		FROM message_logs
		WHERE tenant_id = $1 AND message_id = $2 AND status = 'failed'
		  AND error_code IS DISTINCT FROM 'INVALID_PHONE'
		  AND error_code IS DISTINCT FROM 'CANCELLED'
		  AND ($3 OR error_code IS DISTINCT FROM 'OUTCOME_UNKNOWN')
	`, tenantID, sourceID, includeUncertain)
	if err != nil {
		return nil, fmt.Errorf("query failed recipients: %w", err)
	}
	defer rows.Close()

	prepared := &preparedAudience{}
	for rows.Next() {
		var r preparedRecipient
		var id *uuid.UUID
		if err := rows.Scan(&r.Type, &id, &r.Name, &r.Phone, &r.rendered); err != nil {
			return nil, fmt.Errorf("scan failed recipient: %w", err)
		}
		if id != nil {
			r.ID = *id
		}
		prepared.valid = append(prepared.valid, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(prepared.valid) == 0 {
		return nil, invalid("There are no failed recipients that can be resent to.")
	}

	filter, _ := json.Marshal(map[string]any{"source_message_id": sourceID})
	req := CreateMessageRequest{
		Channel:        "sms",
		AudienceType:   "resend",
		AudienceFilter: filter,
		ContentType:    source.ContentType,
		Content:        source.Content,
		IdempotencyKey: idempotencyKey,
	}
	msg, err := s.insertMessage(ctx, tenantID, actor, req, "sending", prepared)
	if err != nil {
		return nil, err
	}
	if s.dispatcher != nil {
		s.dispatcher.Notify()
	}
	return msg, nil
}

// GetDeliveryStats returns aggregate delivery stats for a message.
func (s *CommsService) GetDeliveryStats(ctx context.Context, tenantID, messageID uuid.UUID) (DeliveryStats, error) {
	var stats DeliveryStats
	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'sent') AS sent,
			COUNT(*) FILTER (WHERE status IN ('delivered', 'read')) AS delivered,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE status = 'pending') AS pending
		FROM message_logs
		WHERE tenant_id = $1 AND message_id = $2
	`, tenantID, messageID).Scan(
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

// AudienceOptions is what the composer offers to choose from.
type AudienceOptions struct {
	Classes   []AudienceClass `json:"classes"`
	Tags      []string        `json:"tags"`
	Variables []string        `json:"variables"`
	MaxUnits  int             `json:"max_units"`
}

// AudienceClass is a grade and stream that has active learners.
type AudienceClass struct {
	Grade  string `json:"grade"`
	Stream string `json:"stream"`
}

// GetAudienceOptions returns the school's real classes and contact tags, so
// the composer never offers an audience that does not exist.
func (s *CommsService) GetAudienceOptions(ctx context.Context, tenantID uuid.UUID) (*AudienceOptions, error) {
	out := &AudienceOptions{Classes: []AudienceClass{}, Tags: []string{}, Variables: SupportedVariables(), MaxUnits: sms.MaxSMSUnits}

	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT grade, COALESCE(stream, '')
		FROM learners
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY 1, 2
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query classes: %w", err)
	}
	for rows.Next() {
		var c AudienceClass
		if err := rows.Scan(&c.Grade, &c.Stream); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan class: %w", err)
		}
		out.Classes = append(out.Classes, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tagRows, err := s.pool.Query(ctx, `
		SELECT DISTINCT unnest(tags) AS tag
		FROM contacts
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY 1
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query tags: %w", err)
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var tag string
		if err := tagRows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		out.Tags = append(out.Tags, tag)
	}
	return out, tagRows.Err()
}
