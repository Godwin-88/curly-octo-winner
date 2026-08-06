package intelligence

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles digital intelligence operations: financial analytics,
// communication analytics, and AI knowledge base.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates an intelligence service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// --- Financial Analytics ---

func (s *Service) FeeCollectionSummary(ctx context.Context, tenantID uuid.UUID, term, year int) ([]FeeCollectionSummary, error) {
	query := `SELECT tenant_id, term, year, invoice_count, total_billed_cents, total_discount_cents,
		total_collected_cents, outstanding_cents, collection_rate
		FROM fee_collection_summary WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2
	if term > 0 {
		query += fmt.Sprintf(` AND term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	query += ` ORDER BY year DESC, term DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FeeCollectionSummary
	for rows.Next() {
		var f FeeCollectionSummary
		if err := rows.Scan(&f.TenantID, &f.Term, &f.Year, &f.InvoiceCount, &f.TotalBilledCents,
			&f.TotalDiscountCents, &f.TotalCollectedCents, &f.OutstandingCents, &f.CollectionRate); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Service) PaymentChannelBreakdown(ctx context.Context, tenantID uuid.UUID, term, year int) ([]PaymentChannelBreakdown, error) {
	query := `SELECT tenant_id, term, year, channel, payment_count, total_cents
		FROM payment_channel_breakdown WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2
	if term > 0 {
		query += fmt.Sprintf(` AND term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	query += ` ORDER BY total_cents DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PaymentChannelBreakdown
	for rows.Next() {
		var p PaymentChannelBreakdown
		if err := rows.Scan(&p.TenantID, &p.Term, &p.Year, &p.Channel, &p.PaymentCount, &p.TotalCents); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Service) FeeDefaulters(ctx context.Context, tenantID uuid.UUID, term, year int) ([]FeeDefaulter, error) {
	query := `SELECT fd.tenant_id, fd.learner_id, l.full_name, l.grade, l.stream, fd.term, fd.year,
		fd.invoice_number, fd.total_cents, fd.discount_cents, fd.paid_cents, fd.balance_cents, fd.due_date, fd.status
		FROM fee_defaulters fd
		JOIN learners l ON l.id = fd.learner_id AND l.tenant_id = fd.tenant_id
		WHERE fd.tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2
	if term > 0 {
		query += fmt.Sprintf(` AND fd.term = $%d`, argIdx)
		args = append(args, term)
		argIdx++
	}
	if year > 0 {
		query += fmt.Sprintf(` AND fd.year = $%d`, argIdx)
		args = append(args, year)
		argIdx++
	}
	query += ` ORDER BY fd.balance_cents DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FeeDefaulter
	for rows.Next() {
		var f FeeDefaulter
		if err := rows.Scan(&f.TenantID, &f.LearnerID, &f.LearnerName, &f.Grade, &f.Stream, &f.Term, &f.Year,
			&f.InvoiceNumber, &f.TotalCents, &f.DiscountCents, &f.PaidCents, &f.BalanceCents, &f.DueDate, &f.Status); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Service) MonthlyCollectionTrend(ctx context.Context, tenantID uuid.UUID) ([]MonthlyCollectionTrend, error) {
	query := `SELECT tenant_id, month, payment_count, total_cents
		FROM monthly_collection_trend WHERE tenant_id = $1 ORDER BY month ASC`

	rows, err := s.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MonthlyCollectionTrend
	for rows.Next() {
		var m MonthlyCollectionTrend
		if err := rows.Scan(&m.TenantID, &m.Month, &m.PaymentCount, &m.TotalCents); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// --- Communication Analytics ---

func (s *Service) CampaignDeliverySummary(ctx context.Context, tenantID uuid.UUID) ([]CampaignDeliverySummary, error) {
	query := `SELECT message_id, tenant_id, channel, audience_type, status, sent_at,
		recipient_count, delivered_count, failed_count, delivery_rate
		FROM campaign_delivery_summary WHERE tenant_id = $1 ORDER BY sent_at DESC NULLS LAST`

	rows, err := s.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CampaignDeliverySummary
	for rows.Next() {
		var c CampaignDeliverySummary
		if err := rows.Scan(&c.MessageID, &c.TenantID, &c.Channel, &c.AudienceType, &c.Status, &c.SentAt,
			&c.RecipientCount, &c.DeliveredCount, &c.FailedCount, &c.DeliveryRate); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) ChannelReach(ctx context.Context, tenantID uuid.UUID) ([]ChannelReach, error) {
	query := `SELECT tenant_id, channel, campaign_count, total_recipients, total_delivered, total_failed
		FROM channel_reach WHERE tenant_id = $1 ORDER BY total_recipients DESC`

	rows, err := s.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ChannelReach
	for rows.Next() {
		var c ChannelReach
		if err := rows.Scan(&c.TenantID, &c.Channel, &c.CampaignCount, &c.TotalRecipients, &c.TotalDelivered, &c.TotalFailed); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) FailedNumbers(ctx context.Context, tenantID uuid.UUID) ([]FailedNumber, error) {
	query := `SELECT tenant_id, message_id, channel, phone, error_code, error_message, created_at
		FROM failed_number_analysis WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 100`

	rows, err := s.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FailedNumber
	for rows.Next() {
		var f FailedNumber
		if err := rows.Scan(&f.TenantID, &f.MessageID, &f.Channel, &f.Phone, &f.ErrorCode, &f.ErrorMessage, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// --- FAQ Knowledge Base ---

const faqColumns = `id, tenant_id, question, answer, category, keywords, is_active, created_at, updated_at`

func scanFAQ(row pgx.Row) (*FAQEntry, error) {
	var f FAQEntry
	err := row.Scan(&f.ID, &f.TenantID, &f.Question, &f.Answer, &f.Category, &f.Keywords, &f.IsActive, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if f.Keywords == nil {
		f.Keywords = []string{}
	}
	return &f, nil
}

func (s *Service) ListFAQEntries(ctx context.Context, tenantID uuid.UUID, category string) ([]FAQEntry, error) {
	query := fmt.Sprintf(`SELECT %s FROM faq_entries WHERE tenant_id = $1`, faqColumns)
	args := []any{tenantID}
	argIdx := 2
	if category != "" {
		query += fmt.Sprintf(` AND category = $%d`, argIdx)
		args = append(args, category)
		argIdx++
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FAQEntry
	for rows.Next() {
		f, err := scanFAQ(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *f)
	}
	return out, rows.Err()
}

func (s *Service) GetFAQEntry(ctx context.Context, tenantID, id uuid.UUID) (*FAQEntry, error) {
	query := fmt.Sprintf(`SELECT %s FROM faq_entries WHERE tenant_id = $1 AND id = $2`, faqColumns)
	return scanFAQ(s.pool.QueryRow(ctx, query, tenantID, id))
}

func (s *Service) CreateFAQEntry(ctx context.Context, tenantID uuid.UUID, req CreateFAQEntryRequest) (*FAQEntry, error) {
	category := req.Category
	if category == "" {
		category = "general"
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	keywords := req.Keywords
	if keywords == nil {
		keywords = []string{}
	}
	query := fmt.Sprintf(`INSERT INTO faq_entries (tenant_id, question, answer, category, keywords, is_active)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING %s`, faqColumns)
	return scanFAQ(s.pool.QueryRow(ctx, query, tenantID, req.Question, req.Answer, category, keywords, isActive))
}

func (s *Service) UpdateFAQEntry(ctx context.Context, tenantID, id uuid.UUID, req UpdateFAQEntryRequest) (*FAQEntry, error) {
	if _, err := s.GetFAQEntry(ctx, tenantID, id); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE faq_entries SET
		question = COALESCE($3, question),
		answer = COALESCE($4, answer),
		category = COALESCE($5, category),
		keywords = COALESCE($6, keywords),
		is_active = COALESCE($7, is_active)
		WHERE tenant_id = $1 AND id = $2
		RETURNING %s`, faqColumns)
	return scanFAQ(s.pool.QueryRow(ctx, query, tenantID, id, req.Question, req.Answer, req.Category, req.Keywords, req.IsActive))
}

func (s *Service) DeleteFAQEntry(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM faq_entries WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Message Template Embeddings ---

const templateColumns = `id, tenant_id, template_id, content, purpose, tone, language, vector_id, created_at`

func scanTemplate(row pgx.Row) (*MessageTemplateEmbedding, error) {
	var t MessageTemplateEmbedding
	err := row.Scan(&t.ID, &t.TenantID, &t.TemplateID, &t.Content, &t.Purpose, &t.Tone, &t.Language, &t.VectorID, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Service) ListTemplateEmbeddings(ctx context.Context, tenantID uuid.UUID) ([]MessageTemplateEmbedding, error) {
	query := fmt.Sprintf(`SELECT %s FROM message_template_embeddings WHERE tenant_id = $1 ORDER BY created_at DESC`, templateColumns)
	rows, err := s.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []MessageTemplateEmbedding
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (s *Service) CreateTemplateEmbedding(ctx context.Context, tenantID uuid.UUID, req CreateTemplateEmbeddingRequest) (*MessageTemplateEmbedding, error) {
	tone := req.Tone
	if tone == "" {
		tone = "formal"
	}
	language := req.Language
	if language == "" {
		language = "en"
	}
	query := fmt.Sprintf(`INSERT INTO message_template_embeddings (tenant_id, template_id, content, purpose, tone, language)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING %s`, templateColumns)
	return scanTemplate(s.pool.QueryRow(ctx, query, tenantID, req.TemplateID, req.Content, req.Purpose, tone, language))
}

func (s *Service) DeleteTemplateEmbedding(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM message_template_embeddings WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Portfolio Summary (rule-based fallback) ---

func (s *Service) PortfolioSummary(ctx context.Context, tenantID, learnerID uuid.UUID, term, year int) (*PortfolioSummary, error) {
	var name string
	err := s.pool.QueryRow(ctx, `SELECT full_name FROM learners WHERE tenant_id = $1 AND id = $2`, tenantID, learnerID).Scan(&name)
	if err != nil {
		return nil, err
	}

	var noteCount int64
	var avgLevel float64
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(AVG(rubric_level), 0)
		FROM assessments
		WHERE tenant_id = $1 AND learner_id = $2 AND term = $3 AND year = $4`,
		tenantID, learnerID, term, year).Scan(&noteCount, &avgLevel)
	if err != nil {
		return nil, err
	}

	summary := "No assessment notes available for this learner/term."
	if noteCount > 0 {
		level := "Meeting Expectation"
		switch {
		case avgLevel < 2:
			level = "Below Expectation"
		case avgLevel < 3:
			level = "Approaching Expectation"
		case avgLevel > 3:
			level = "Exceeding Expectation"
		}
		summary = fmt.Sprintf("%s has %d recorded observations with an average rubric level of %.2f (%s).", name, noteCount, avgLevel, level)
	}

	return &PortfolioSummary{
		LearnerID:   learnerID,
		LearnerName: name,
		Term:        term,
		Year:        year,
		Summary:     summary,
		NoteCount:   noteCount,
	}, nil
}

// round2 rounds a float to 2 decimal places.
func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
