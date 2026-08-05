package finance

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/pkg/mpesa"
)

// MpesaService handles M-Pesa STK push payments via the Daraja API.
type MpesaService struct {
	pool     *pgxpool.Pool
	client   *mpesa.Client
	callback string
}

// NewMpesaService creates an M-Pesa service for STK push payments.
func NewMpesaService(pool *pgxpool.Pool, client *mpesa.Client, callbackURL string) *MpesaService {
	return &MpesaService{pool: pool, client: client, callback: callbackURL}
}

// InitiateSTKPush initiates an M-Pesa STK push for an invoice payment.
// It creates a pending payment record and returns it with the checkout request ID.
func (s *MpesaService) InitiateSTKPush(ctx context.Context, tenantID uuid.UUID, req MpesaStkRequest) (*Payment, error) {
	if req.InvoiceID == uuid.Nil || req.Phone == "" || req.AmountCents <= 0 {
		return nil, errors.New("invoice_id, phone, and a positive amount_cents are required")
	}

	// Fetch invoice info for account reference
	var invoiceNumber string
	var learnerName string
	err := s.pool.QueryRow(ctx, `
		SELECT i.invoice_number, l.full_name
		FROM invoices i
		JOIN learners l ON l.id = i.learner_id
		WHERE i.tenant_id = $1 AND i.id = $2`, tenantID, req.InvoiceID).
		Scan(&invoiceNumber, &learnerName)
	if err != nil {
		return nil, err
	}

	// Normalize phone to 2547XXXXXXXX
	phone := req.Phone
	if len(phone) == 9 && phone[0] == '7' {
		phone = "254" + phone
	} else if len(phone) == 10 && phone[0] == '0' {
		phone = "254" + phone[1:]
	} else if len(phone) == 13 && phone[:4] == "+254" {
		phone = phone[1:]
	}
	if len(phone) != 12 || phone[:3] != "254" {
		return nil, errors.New("phone must be a valid Kenyan phone number")
	}

	amountKES := (req.AmountCents + 99) / 100 // round up to whole shillings
	if amountKES < 1 {
		amountKES = 1
	}
	accountRef := invoiceNumber
	if len(accountRef) > 12 {
		accountRef = accountRef[:12]
	}

	stk, err := s.client.STKPush(ctx, phone, fmt.Sprintf("%d", amountKES), accountRef, s.callback)
	if err != nil {
		return nil, err
	}

	// Create a pending payment record linked to the checkout request
	status := "pending"
	var paidBy *string
	if req.PaidBy != nil {
		paidBy = req.PaidBy
	}
	phonePtr := &phone

	query := fmt.Sprintf(`INSERT INTO payments (tenant_id, invoice_id, amount_cents, channel, status, phone, paid_by,
		checkout_request_id, merchant_request_id)
		VALUES ($1, $2, $3, 'mpesa', $4, $5, $6, $7, $8)
		RETURNING %s`, paymentColumns+`
		FROM invoices i JOIN learners l ON l.id = i.learner_id WHERE i.id = payments.invoice_id`)
	pay, err := scanPayment(s.pool.QueryRow(ctx, query,
		tenantID, req.InvoiceID, req.AmountCents, status, phonePtr, paidBy,
		stk.CheckoutRequestID, stk.MerchantRequestID,
	))
	if err != nil {
		return nil, err
	}
	return pay, nil
}

// ConfirmSTKPush confirms an M-Pesa STK callback.
// It marks the payment completed (or failed), records M-Pesa details, and refreshes the invoice.
func (s *MpesaService) ConfirmSTKPush(ctx context.Context, tenantID uuid.UUID, checkoutRequestID, resultCode, resultDesc, receipt string) error {
	if checkoutRequestID == "" {
		return errors.New("checkout_request_id is required")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var invoiceID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT invoice_id FROM payments
		WHERE tenant_id = $1 AND checkout_request_id = $2 AND channel = 'mpesa'
		ORDER BY created_at DESC LIMIT 1`, tenantID, checkoutRequestID).
		Scan(&invoiceID)
	if err != nil {
		return err
	}

	newStatus := "completed"
	if resultCode != "0" {
		newStatus = "failed"
	}
	_, err = tx.Exec(ctx, `
		UPDATE payments SET
			status = $3,
			mpesa_receipt = COALESCE($4, mpesa_receipt),
			mpesa_result_code = $5,
			mpesa_result_desc = $6,
			paid_at = CASE WHEN $3 = 'completed' THEN COALESCE(paid_at, now()) ELSE paid_at END
		WHERE tenant_id = $1 AND id = (SELECT id FROM payments WHERE tenant_id = $1 AND checkout_request_id = $2 AND channel = 'mpesa' ORDER BY created_at DESC LIMIT 1)`,
		tenantID, checkoutRequestID, newStatus, receipt, resultCode, resultDesc)
	if err != nil {
		return err
	}

	if err := refreshInvoiceFinance(ctx, tx, tenantID, invoiceID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ListInvoicePayments returns all payments for an invoice.
func (s *Service) ListInvoicePayments(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]Payment, error) {
	query := fmt.Sprintf(`SELECT %s FROM payments p
		JOIN invoices i ON i.id = p.invoice_id
		JOIN learners l ON l.id = i.learner_id
		WHERE p.tenant_id = $1 AND p.invoice_id = $2
		ORDER BY p.created_at DESC`, paymentColumns)
	rows, err := s.pool.Query(ctx, query, tenantID, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []Payment
	for rows.Next() {
		pay, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		payments = append(payments, *pay)
	}
	return payments, rows.Err()
}
