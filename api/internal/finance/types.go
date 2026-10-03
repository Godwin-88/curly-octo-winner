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
	LearnerUPI  string `json:"learner_upi,omitempty"`
	// Set once the invoice is voided.
	VoidedAt   *time.Time `json:"voided_at,omitempty"`
	VoidReason *string    `json:"void_reason,omitempty"`
	// What is still owed. Never negative: money paid beyond the invoice is
	// CreditCents.
	BalanceCents int64 `json:"balance_cents"`
	CreditCents  int64 `json:"credit_cents"`
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
	// The school's own receipt number, given when the money is confirmed.
	ReceiptNumber  *string    `json:"receipt_number,omitempty"`
	FailureCode    *string    `json:"failure_code,omitempty"`
	ReversedAt     *time.Time `json:"reversed_at,omitempty"`
	ReversalReason *string    `json:"reversal_reason,omitempty"`
	// Joined fields
	InvoiceNumber string    `json:"invoice_number,omitempty"`
	LearnerID     uuid.UUID `json:"learner_id"`
	LearnerName   string    `json:"learner_name,omitempty"`
	Grade         string    `json:"grade,omitempty"`
}

type CreateFeeStructureRequest struct {
	Name   string         `json:"name"`
	Grade  string         `json:"grade"`
	Term   int            `json:"term"`
	Year   int            `json:"year"`
	Active *bool          `json:"active,omitempty"`
	Notes  *string        `json:"notes,omitempty"`
	Items  []FeeItemInput `json:"items,omitempty"`
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
	// Optional items of the fee structure (transport, lunch) are billed only
	// when this is set.
	IncludeOptional bool `json:"include_optional,omitempty"`
	// Direct items only used when no fee_structure_id is provided
	Items []FeeItemInput `json:"items,omitempty"`
}

// UpdateInvoiceRequest changes what may change on an issued invoice. Its
// status is never set by hand: it follows from the payments.
type UpdateInvoiceRequest struct {
	DueDate *string `json:"due_date,omitempty"`
	Notes   *string `json:"notes,omitempty"`
}

// BulkInvoiceRequest bills every active learner of a fee structure's grade.
type BulkInvoiceRequest struct {
	FeeStructureID  uuid.UUID `json:"fee_structure_id"`
	Stream          string    `json:"stream,omitempty"`
	DueDate         *string   `json:"due_date,omitempty"`
	IncludeOptional bool      `json:"include_optional,omitempty"`
	// DryRun answers what would happen and writes nothing.
	DryRun bool `json:"dry_run,omitempty"`
}

type BulkInvoiceResult struct {
	Learners        int   `json:"learners"`
	Created         int   `json:"created"`
	AlreadyInvoiced int   `json:"already_invoiced"`
	EachCents       int64 `json:"each_cents"`
	TotalCents      int64 `json:"total_cents"`
}

type CreateDiscountRequest struct {
	AmountCents  int64   `json:"amount_cents"`
	DiscountType string  `json:"discount_type"`
	Reason       *string `json:"reason,omitempty"`
}

type CreatePaymentRequest struct {
	InvoiceID      uuid.UUID  `json:"invoice_id"`
	AmountCents    int64      `json:"amount_cents"`
	Channel        string     `json:"channel"`
	Reference      *string    `json:"reference,omitempty"`
	PaidBy         *string    `json:"paid_by,omitempty"`
	Phone          *string    `json:"phone,omitempty"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	IdempotencyKey string     `json:"idempotency_key,omitempty"`
}

type MpesaStkRequest struct {
	InvoiceID      uuid.UUID `json:"invoice_id"`
	Phone          string    `json:"phone"`
	AmountCents    int64     `json:"amount_cents"`
	PaidBy         *string   `json:"paid_by,omitempty"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
}

// InboxPayment is money received on the school's paybill that did not come
// from an STK request made here.
type InboxPayment struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	TransID        string     `json:"trans_id"`
	TransTime      time.Time  `json:"trans_time"`
	AmountCents    int64      `json:"amount_cents"`
	AllocatedCents int64      `json:"allocated_cents"`
	RemainingCents int64      `json:"remaining_cents"`
	BillRef        *string    `json:"bill_ref,omitempty"`
	PayerName      *string    `json:"payer_name,omitempty"`
	PayerPhone     *string    `json:"payer_phone,omitempty"`
	ShortCode      *string    `json:"short_code,omitempty"`
	LearnerID      *uuid.UUID `json:"learner_id,omitempty"`
	LearnerName    *string    `json:"learner_name,omitempty"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
}

type AllocateRequest struct {
	InvoiceID   uuid.UUID `json:"invoice_id"`
	AmountCents int64     `json:"amount_cents"`
}

// ArrearsRow is one learner who still owes fees.
type ArrearsRow struct {
	LearnerID    uuid.UUID `json:"learner_id"`
	LearnerName  string    `json:"learner_name"`
	Grade        string    `json:"grade"`
	Stream       string    `json:"stream"`
	Invoices     int       `json:"invoices"`
	BalanceCents int64     `json:"balance_cents"`
	OldestDue    *string   `json:"oldest_due,omitempty"`
}

// Statement is everything billed to and received for one learner.
type Statement struct {
	LearnerID    uuid.UUID `json:"learner_id"`
	LearnerName  string    `json:"learner_name"`
	Grade        string    `json:"grade"`
	Stream       string    `json:"stream"`
	BilledCents  int64     `json:"billed_cents"`
	PaidCents    int64     `json:"paid_cents"`
	BalanceCents int64     `json:"balance_cents"`
	Invoices     []Invoice `json:"invoices"`
	Payments     []Payment `json:"payments"`
}

// Summary is the position for a term, or for everything when term and year
// are zero.
type Summary struct {
	Invoices         int              `json:"invoices"`
	BilledCents      int64            `json:"billed_cents"`
	DiscountCents    int64            `json:"discount_cents"`
	CollectedCents   int64            `json:"collected_cents"`
	OutstandingCents int64            `json:"outstanding_cents"`
	OverdueCents     int64            `json:"overdue_cents"`
	LearnersOwing    int              `json:"learners_owing"`
	ByChannel        map[string]int64 `json:"by_channel"`
	UnmatchedCount   int              `json:"unmatched_count"`
	UnmatchedCents   int64            `json:"unmatched_cents"`
	PendingMpesa     int              `json:"pending_mpesa"`
}
