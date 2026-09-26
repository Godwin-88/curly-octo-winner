package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Campaign struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	Name           string    `json:"name"`
	AudienceType   string    `json:"audience_type"`
	AudienceFilter string    `json:"audience_filter"`
	Content        string    `json:"content"`
	TemplateID     *string   `json:"template_id,omitempty"`
	Status         string    `json:"status"`
	RecipientCount int       `json:"recipient_count"`
	DeliveredCount int       `json:"delivered_count"`
	FailedCount    int       `json:"failed_count"`
	ScheduledAt    *string   `json:"scheduled_at,omitempty"`
	SentAt         *string   `json:"sent_at,omitempty"`
	CreatedBy      uuid.UUID `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CampaignLog struct {
	ID             uuid.UUID `json:"id"`
	CampaignID     uuid.UUID `json:"campaign_id"`
	RecipientPhone string    `json:"recipient_phone"`
	RecipientName  string    `json:"recipient_name"`
	Channel        string    `json:"channel"`
	Status         string    `json:"status"`
	ProviderMsgID  *string   `json:"provider_msg_id,omitempty"`
	ErrorCode      *string   `json:"error_code,omitempty"`
	ErrorMessage   *string   `json:"error_message,omitempty"`
	SentAt         *string   `json:"sent_at,omitempty"`
	DeliveredAt    *string   `json:"delivered_at,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type SendCampaignRequest struct {
	Name           string          `json:"name"`
	AudienceType   string          `json:"audience_type"`
	AudienceFilter json.RawMessage `json:"audience_filter"`
	Content        string          `json:"content"`
	TemplateID     *string         `json:"template_id,omitempty"`
	ScheduledAt    *string         `json:"scheduled_at,omitempty"`
}

type SMSService struct {
	pool     *pgxpool.Pool
	atClient *ATClient
}

func NewSMSService(pool *pgxpool.Pool, atClient *ATClient) *SMSService {
	return &SMSService{pool: pool, atClient: atClient}
}

func (s *SMSService) SendCampaign(ctx context.Context, tenantID uuid.UUID, req SendCampaignRequest) (*Campaign, error) {
	recipients, err := s.resolveAudience(ctx, tenantID, req.AudienceType, req.AudienceFilter)
	if err != nil {
		return nil, fmt.Errorf("resolve audience: %w", err)
	}

	status := "sending"
	if req.ScheduledAt != nil && *req.ScheduledAt != "" {
		scheduled, err := time.Parse(time.RFC3339, *req.ScheduledAt)
		if err == nil && scheduled.After(time.Now()) {
			status = "scheduled"
		}
	}

	var campaignID uuid.UUID
	err = s.pool.QueryRow(ctx, `
		INSERT INTO messages (tenant_id, channel, audience_type, audience_filter, content_type, content, template_id, status, scheduled_at, sent_by, recipient_count)
		VALUES ($1, 'sms', $2, $3::jsonb, 'text', $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, tenantID, req.AudienceType, req.AudienceFilter, req.Content, req.TemplateID, status, req.ScheduledAt, uuid.Nil, len(recipients)).Scan(&campaignID)
	if err != nil {
		return nil, fmt.Errorf("insert campaign: %w", err)
	}

	if status == "scheduled" {
		return s.getCampaign(ctx, tenantID, campaignID)
	}

	phones := make([]string, 0, len(recipients))
	names := make([]string, 0, len(recipients))
	ids := make([]uuid.UUID, 0, len(recipients))
	for _, r := range recipients {
		normalized, err := NormalizeKenyanPhone(r.Phone)
		if err != nil {
			continue
		}
		phones = append(phones, normalized)
		names = append(names, r.Name)
		ids = append(ids, r.ID)
	}

	if len(phones) == 0 {
		return s.getCampaign(ctx, tenantID, campaignID)
	}

	results, err := s.atClient.SendBulk(ctx, BulkSMSRequest{
		To:      phones,
		Message: req.Content,
	})
	if err != nil {
		return nil, fmt.Errorf("send bulk sms: %w", err)
	}

	sentAt := time.Now().Format(time.RFC3339)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for i, res := range results {
		logStatus := "failed"
		if res.Status == "Success" {
			logStatus = "delivered"
		}
		_, _ = tx.Exec(ctx, `
			INSERT INTO message_logs (message_id, recipient_type, recipient_id, phone, channel, status, provider_message_id, sent_at)
			VALUES ($1, 'guardian', $2, $3, 'sms', $4, $5, $6)
		`, campaignID, ids[i], phones[i], logStatus, &res.MessageID, &sentAt)
	}

	delivered := 0
	failed := 0
	for _, res := range results {
		if res.Status == "Success" {
			delivered++
		} else {
			failed++
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE messages
		SET status = 'sent', sent_at = $1, delivered_count = $2, failed_count = $3
		WHERE id = $4
	`, &sentAt, delivered, failed, campaignID)
	if err != nil {
		return nil, fmt.Errorf("update campaign: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return s.getCampaign(ctx, tenantID, campaignID)
}

func (s *SMSService) GetCampaign(ctx context.Context, tenantID, campaignID uuid.UUID) (*Campaign, error) {
	return s.getCampaign(ctx, tenantID, campaignID)
}

func (s *SMSService) getCampaign(ctx context.Context, tenantID, campaignID uuid.UUID) (*Campaign, error) {
	var c Campaign
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, COALESCE(name, ''), audience_type, audience_filter, content, template_id, status, recipient_count, delivered_count, failed_count, scheduled_at, sent_at, created_by, created_at, updated_at
		FROM messages
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, campaignID).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.AudienceType, &c.AudienceFilter,
		&c.Content, &c.TemplateID, &c.Status, &c.RecipientCount,
		&c.DeliveredCount, &c.FailedCount, &c.ScheduledAt, &c.SentAt,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *SMSService) ListCampaigns(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]Campaign, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, COALESCE(name, ''), audience_type, audience_filter, content, template_id, status, recipient_count, delivered_count, failed_count, scheduled_at, sent_at, created_by, created_at, updated_at
		FROM messages
		WHERE tenant_id = $1 AND channel = 'sms'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var campaigns []Campaign
	for rows.Next() {
		var c Campaign
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.Name, &c.AudienceType, &c.AudienceFilter,
			&c.Content, &c.TemplateID, &c.Status, &c.RecipientCount,
			&c.DeliveredCount, &c.FailedCount, &c.ScheduledAt, &c.SentAt,
			&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	return campaigns, rows.Err()
}

func (s *SMSService) GetCampaignLogs(ctx context.Context, tenantID, campaignID uuid.UUID, limit, offset int) ([]CampaignLog, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, message_id, recipient_type, recipient_id, phone, channel, status, provider_message_id, error_code, error_message, sent_at, delivered_at, created_at
		FROM message_logs
		WHERE message_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, campaignID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []CampaignLog
	for rows.Next() {
		var l CampaignLog
		if err := rows.Scan(
			&l.ID, &l.CampaignID, &l.RecipientPhone, &l.RecipientName,
			&l.Channel, &l.Status, &l.ProviderMsgID, &l.ErrorCode,
			&l.ErrorMessage, &l.SentAt, &l.DeliveredAt, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (s *SMSService) resolveAudience(ctx context.Context, tenantID uuid.UUID, audienceType string, filter json.RawMessage) ([]struct {
	ID    uuid.UUID
	Phone string
	Name  string
}, error) {
	var recipients []struct {
		ID    uuid.UUID
		Phone string
		Name  string
	}

	switch audienceType {
	case "all_parents":
		err := s.queryRecipients(ctx, tenantID, `
			SELECT DISTINCT g.id, COALESCE(g.phone_primary, '') AS phone, g.full_name
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
			SELECT DISTINCT g.id, COALESCE(g.phone_primary, '') AS phone, g.full_name
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
			SELECT DISTINCT g.id, COALESCE(g.phone_primary, '') AS phone, g.full_name
			FROM guardians g
			JOIN learners l ON l.tenant_id = g.tenant_id AND g.id = ANY(l.guardian_ids)
			WHERE g.tenant_id = $1 AND l.grade = $2 AND l.stream = $3 AND g.is_sms_opted_out = false
		`, &recipients, f.Grade, f.Stream)
		if err != nil {
			return nil, err
		}
	case "fee_defaulters":
		err := s.queryRecipients(ctx, tenantID, `
			SELECT DISTINCT g.id, COALESCE(g.phone_primary, '') AS phone, g.full_name
			FROM guardians g
			JOIN learners l ON l.tenant_id = g.tenant_id AND g.id = ANY(l.guardian_ids)
			JOIN invoices i ON i.tenant_id = g.tenant_id AND i.learner_id = l.id
			WHERE g.tenant_id = $1 AND i.status IN ('unpaid', 'overdue') AND g.is_sms_opted_out = false
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
			SELECT id, COALESCE(phone_primary, '') AS phone, full_name
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

func (s *SMSService) queryRecipients(ctx context.Context, tenantID uuid.UUID, query string, out *[]struct {
	ID    uuid.UUID
	Phone string
	Name  string
}, args ...any) error {
	fullArgs := append([]any{tenantID}, args...)
	rows, err := s.pool.Query(ctx, query, fullArgs...)
	if err != nil {
		return fmt.Errorf("query recipients: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var r struct {
			ID    uuid.UUID
			Phone string
			Name  string
		}
		if err := rows.Scan(&r.ID, &r.Phone, &r.Name); err != nil {
			return fmt.Errorf("scan recipient: %w", err)
		}
		*out = append(*out, r)
	}
	return rows.Err()
}

func InjectVariables(template string, data map[string]string) string {
	result := template
	for key, value := range data {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
