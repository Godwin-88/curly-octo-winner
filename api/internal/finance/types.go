package finance

import (
	"time"

	"github.com/google/uuid"
)

// FeeStructure is a per-grade, per-term schedule of fee items.
type FeeStructure struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	Name       string     `json:"name"`
	Grade      string     `json:"grade"`
	Term       int        `json:"term"`
	Year       int        `json:"year"`
	TotalCents int64      `json:"total_cents"`
	Active     bool       `json:"active"`
	Notes      *string    `json:"notes,omitempty"`
	CreatedBy  *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	// Items included on detail/list responses
	Items []FeeStructureItem `json:"items,omitempty"`
}

// FeeStructureItem is a single line item within a fee structure.
type FeeStructureItem struct {
	ID             uuid.UUID `json:"id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	FeeStructureID uuid.UUID `json:"fee_structure_id"`
	Name           string    `json:"name"`
	AmountCents    int64     `json:"amount_cents"`
	ItemType       string    `json:"item_type"`
	IsOptional     bool      `json:"is_optional"`
	SortOrder      int       `json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
}

// Discount is a partial/full waiver or scholarship on an invoice.
type Discount struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	InvoiceID    uuid.UUID  `json:"invoice_id"`
	AmountCents  int64      `json:"amount_cents"`
	DiscountType string     `json:"discount_type"`
	Reason       *string    `json:"reason,omitempty"`
	ApprovedBy   *uuid.UUID `json:"approved_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Invoice is a learner's fee bill for a term/year.
type Invoice struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	LearnerID      uuid.UUID  `json:"learner_id"`
	FeeStructureID *uuid.UUID `json:"fee_structure_id,omitempty"`
	InvoiceNumber  string     `json:"invoice_number"`
	Term           int        `json:"term"`
	Year           int        `json:"year"`
	IssueDate      string     `json:"issue_date"`
	DueDate        *string    `json:"due_date,omitempty"`
	TotalCents     int64      `json:"total_cents"`
	DiscountCents  int64      `json:"discount_cents"`
	PaidCents      int64      `json:"paid_cents"`
	Status         string     `json:"status"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	// Joined fields
	LearnerName string `json:"learner_name,omitempty"`
	Grade       string `json:"grade,omitempty"`
	Stream      string `json:"stream,omitempty"`
	// Computed balance
	BalanceCents int64 `json:"balance_cents,omitempty"`
	// Items included on detail responses
	Items []InvoiceItem `json:"items,omitempty"`
}

// InvoiceItem is a snapshot line item on an invoice.
type InvoiceItem struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	InvoiceID   uuid.UUID `json:"invoice_id"`
	Name        string    `json:"name"`
	AmountCents int64     `json:"amount_cents"`
	ItemType    string    `json:"item_type"`
	IsOptional  bool      `json:"is_optional"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

// Payment is a single payment against an invoice.
type Payment struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	InvoiceID         uuid.UUID  `json:"invoice_id"`
	AmountCents       int64      `json:"amount_cents"`
	Channel           string     `json:"channel"`
	Status            string     `json:"status"`
	Reference         *string    `json:"reference,omitempty"`
	PaidBy            *string    `json:"paid_by,omitempty"`
	Phone             *string    `json:"phone,omitempty"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	ReceivedBy        *uuid.UUID `json:"received_by,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	CheckoutRequestID *string    `json:"checkout_request_id,omitempty"`
	MerchantRequestID *string    `json:"merchant_request_id,omitempty"`
	MpesaReceipt      *string    `json:"mpesa_receipt,omitempty"`
	MpesaResultCode   *string    `json:"mpesa_result_code,omitempty"`
	MpesaResultDesc   *string    `json:"mpesa_result_desc,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	// Joined fields
	InvoiceNumber string `json:"invoice_number,omitempty"`
	LearnerName   string `json:"learner_name,omitempty"`
	Grade         string `json:"grade,omitempty"`
}

type CreateFeeStructureRequest struct {
	Name      string         `json:"name"`
	Grade     string         `json:"grade"`
	Term      int            `json:"term"`
	Year      int            `json:"year"`
	Active    *bool          `json:"active,omitempty"`
	Notes     *string        `json:"notes,omitempty"`
	CreatedBy *uuid.UUID     `json:"created_by,omitempty"`
	Items     []FeeItemInput `json:"items,omitempty"`
}

type UpdateFeeStructureRequest struct {
	Name   *string `json:"name,omitempty"`
	Grade  *string `json:"grade,omitempty"`
	Term   *int    `json:"term,omitempty"`
	Year   *int    `json:"year,omitempty"`
	Active *bool   `json:"active,omitempty"`
	Notes  *string `json:"notes,omitempty"`
}

type FeeItemInput struct {
	Name        string `json:"name"`
	AmountCents int64  `json:"amount_cents"`
	ItemType    string `json:"item_type"`
	IsOptional  *bool  `json:"is_optional,omitempty"`
	SortOrder   *int   `json:"sort_order,omitempty"`
}

type CreateInvoiceRequest struct {
	LearnerID      uuid.UUID  `json:"learner_id"`
	FeeStructureID *uuid.UUID `json:"fee_structure_id,omitempty"`
	Term           int        `json:"term"`
	Year           int        `json:"year"`
	IssueDate      *string    `json:"issue_date,omitempty"`
	DueDate        *string    `json:"due_date,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	// Direct items only used when no fee_structure_id is provided
	Items []FeeItemInput `json:"items,omitempty"`
}

type UpdateInvoiceRequest struct {
	DueDate *string `json:"due_date,omitempty"`
	Status  *string `json:"status,omitempty"`
	Notes   *string `json:"notes,omitempty"`
}

type CreateDiscountRequest struct {
	AmountCents  int64      `json:"amount_cents"`
	DiscountType string     `json:"discount_type"`
	Reason       *string    `json:"reason,omitempty"`
	ApprovedBy   *uuid.UUID `json:"approved_by,omitempty"`
}

type CreatePaymentRequest struct {
	InvoiceID   uuid.UUID  `json:"invoice_id"`
	AmountCents int64      `json:"amount_cents"`
	Channel     string     `json:"channel"`
	Reference   *string    `json:"reference,omitempty"`
	PaidBy      *string    `json:"paid_by,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	ReceivedBy  *uuid.UUID `json:"received_by,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
}

type MpesaStkRequest struct {
	InvoiceID   uuid.UUID `json:"invoice_id"`
	Phone       string    `json:"phone"`
	AmountCents int64     `json:"amount_cents"`
	PaidBy      *string   `json:"paid_by,omitempty"`
}
