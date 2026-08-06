package intelligence

import (
	"time"

	"github.com/google/uuid"
)

// --- Financial Analytics ---

type FeeCollectionSummary struct {
	TenantID            uuid.UUID `json:"tenant_id"`
	Term                int       `json:"term"`
	Year                int       `json:"year"`
	InvoiceCount        int64     `json:"invoice_count"`
	TotalBilledCents    int64     `json:"total_billed_cents"`
	TotalDiscountCents  int64     `json:"total_discount_cents"`
	TotalCollectedCents int64     `json:"total_collected_cents"`
	OutstandingCents    int64     `json:"outstanding_cents"`
	CollectionRate      float64   `json:"collection_rate"`
}

type PaymentChannelBreakdown struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	Term         int       `json:"term"`
	Year         int       `json:"year"`
	Channel      string    `json:"channel"`
	PaymentCount int64     `json:"payment_count"`
	TotalCents   int64     `json:"total_cents"`
}

type FeeDefaulter struct {
	TenantID      uuid.UUID  `json:"tenant_id"`
	LearnerID     uuid.UUID  `json:"learner_id"`
	LearnerName   string     `json:"learner_name"`
	Grade         string     `json:"grade"`
	Stream        string     `json:"stream"`
	Term          int        `json:"term"`
	Year          int        `json:"year"`
	InvoiceNumber string     `json:"invoice_number"`
	TotalCents    int64      `json:"total_cents"`
	DiscountCents int64      `json:"discount_cents"`
	PaidCents     int64      `json:"paid_cents"`
	BalanceCents  int64      `json:"balance_cents"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	Status        string     `json:"status"`
}

type MonthlyCollectionTrend struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	Month        time.Time `json:"month"`
	PaymentCount int64     `json:"payment_count"`
	TotalCents   int64     `json:"total_cents"`
}

// --- Communication Analytics ---

type CampaignDeliverySummary struct {
	MessageID      uuid.UUID  `json:"message_id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	Channel        string     `json:"channel"`
	AudienceType   string     `json:"audience_type"`
	Status         string     `json:"status"`
	SentAt         *time.Time `json:"sent_at,omitempty"`
	RecipientCount int        `json:"recipient_count"`
	DeliveredCount int        `json:"delivered_count"`
	FailedCount    int        `json:"failed_count"`
	DeliveryRate   float64    `json:"delivery_rate"`
}

type ChannelReach struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	Channel         string    `json:"channel"`
	CampaignCount   int64     `json:"campaign_count"`
	TotalRecipients int64     `json:"total_recipients"`
	TotalDelivered  int64     `json:"total_delivered"`
	TotalFailed     int64     `json:"total_failed"`
}

type FailedNumber struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	MessageID    uuid.UUID `json:"message_id"`
	Channel      string    `json:"channel"`
	Phone        string    `json:"phone"`
	ErrorCode    *string   `json:"error_code,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// --- AI Knowledge Base ---

type FAQEntry struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	Category  string    `json:"category"`
	Keywords  []string  `json:"keywords"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateFAQEntryRequest struct {
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Category string   `json:"category"`
	Keywords []string `json:"keywords,omitempty"`
	IsActive *bool    `json:"is_active,omitempty"`
}

type UpdateFAQEntryRequest struct {
	Question *string  `json:"question,omitempty"`
	Answer   *string  `json:"answer,omitempty"`
	Category *string  `json:"category,omitempty"`
	Keywords []string `json:"keywords,omitempty"`
	IsActive *bool    `json:"is_active,omitempty"`
}

type MessageTemplateEmbedding struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	TemplateID *string   `json:"template_id,omitempty"`
	Content    string    `json:"content"`
	Purpose    *string   `json:"purpose,omitempty"`
	Tone       string    `json:"tone"`
	Language   string    `json:"language"`
	VectorID   *string   `json:"vector_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateTemplateEmbeddingRequest struct {
	TemplateID *string `json:"template_id,omitempty"`
	Content    string  `json:"content"`
	Purpose    *string `json:"purpose,omitempty"`
	Tone       string  `json:"tone"`
	Language   string  `json:"language"`
}

type TemplateSuggestion struct {
	Content  string  `json:"content"`
	Purpose  *string `json:"purpose,omitempty"`
	Tone     string  `json:"tone"`
	Language string  `json:"language"`
	Score    float64 `json:"score"`
}

type AutoResponse struct {
	Answer   string  `json:"answer"`
	Category string  `json:"category"`
	Score    float64 `json:"score"`
	Matched  bool    `json:"matched"`
}

type PortfolioSummary struct {
	LearnerID   uuid.UUID `json:"learner_id"`
	LearnerName string    `json:"learner_name"`
	Term        int       `json:"term"`
	Year        int       `json:"year"`
	Summary     string    `json:"summary"`
	NoteCount   int64     `json:"note_count"`
}
