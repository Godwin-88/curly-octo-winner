package finance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shule360/api/pkg/mpesa"
)

// clientCache keeps one Daraja client per set of credentials, so its access
// token is reused across requests.
type clientCache struct {
	mu      sync.Mutex
	clients map[string]*mpesa.Client
}

func newClientCache() *clientCache {
	return &clientCache{clients: map[string]*mpesa.Client{}}
}

func (c *clientCache) get(key, secret, passkey, shortCode, baseURL string) *mpesa.Client {
	sum := sha256.Sum256([]byte(strings.Join([]string{key, secret, passkey, shortCode, baseURL}, "\x00")))
	id := hex.EncodeToString(sum[:])
	c.mu.Lock()
	defer c.mu.Unlock()
	if client, ok := c.clients[id]; ok {
		return client
	}
	client := mpesa.NewClient(key, secret, passkey, shortCode, baseURL)
	c.clients[id] = client
	return client
}

// --- Paybill payments (C2B) ---
//
// Most parents pay from the M-Pesa menu: Lipa na M-Pesa → Paybill → the
// school's number → an account number. Safaricom then tells this server.
// Every such payment is kept, whether or not it can be matched to a learner:
// it is money the school has received.

// C2BConfirmation is Safaricom's notice of a paybill payment.
type C2BConfirmation struct {
	TransID           string
	TransTime         string // YYYYMMDDHHmmss, East Africa Time
	AmountCents       int64
	BusinessShortCode string
	BillRefNumber     string
	MSISDN            string
	PayerName         string
}

// ParseAmount turns Safaricom's "1500.00" into cents without going through a
// float.
func ParseAmount(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	whole, frac, _ := strings.Cut(raw, ".")
	if whole == "" {
		return 0, fmt.Errorf("amount %q is not a number", raw)
	}
	frac = (frac + "00")[:2]
	var cents int64
	for _, part := range []string{whole, frac} {
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return 0, fmt.Errorf("amount %q is not a number", raw)
			}
			cents = cents*10 + int64(ch-'0')
			if cents > 1e15 {
				return 0, fmt.Errorf("amount %q is too large", raw)
			}
		}
	}
	return cents, nil
}

// ErrUnknownPaybill: no school, or more than one that cannot be told apart,
// collects into the shortcode a payment was made to.
var ErrUnknownPaybill = errors.New("no single school collects into this paybill")

// schoolForPaybill finds the school a paybill payment belongs to. When
// several schools share a paybill, the account number decides.
func schoolForPaybill(ctx context.Context, q querier, shortCode, billRef string) (uuid.UUID, error) {
	rows, err := q.Query(ctx, `
		SELECT id FROM tenants WHERE mpesa_shortcode = $1
		UNION
		SELECT tenant_id FROM tenant_integrations
		WHERE provider = 'mpesa' AND is_enabled AND config->>'shortcode' = $1`, shortCode)
	if err != nil {
		return uuid.Nil, err
	}
	var schools []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return uuid.Nil, err
		}
		schools = append(schools, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return uuid.Nil, err
	}
	if len(schools) == 1 {
		return schools[0], nil
	}

	var matched []uuid.UUID
	for _, school := range schools {
		learner, err := learnerForAccount(ctx, q, school, billRef)
		if err != nil {
			return uuid.Nil, err
		}
		if learner != nil {
			matched = append(matched, school)
		}
	}
	if len(matched) == 1 {
		return matched[0], nil
	}
	return uuid.Nil, ErrUnknownPaybill
}

// learnerForAccount reads the account number a parent typed: an invoice
// number or the learner's UPI, in any case and with or without spaces.
func learnerForAccount(ctx context.Context, q querier, tenantID uuid.UUID, billRef string) (*uuid.UUID, error) {
	ref := strings.ToUpper(strings.Join(strings.Fields(billRef), ""))
	if ref == "" {
		return nil, nil
	}
	rows, err := q.Query(ctx, `
		SELECT learner_id FROM invoices WHERE tenant_id = $1 AND upper(invoice_number) = $2
		UNION
		SELECT id FROM learners WHERE tenant_id = $1 AND upper(replace(upi, ' ', '')) = $2`, tenantID, ref)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var found []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		found = append(found, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(found) != 1 {
		return nil, nil
	}
	return &found[0], nil
}

// ReceiveC2B records a paybill payment and, when the account number names a
// learner, puts it against that learner's oldest unpaid invoices. A repeated
// confirmation changes nothing.
func (s *Service) ReceiveC2B(ctx context.Context, c C2BConfirmation) error {
	c.TransID = strings.ToUpper(strings.TrimSpace(c.TransID))
	if c.TransID == "" || c.AmountCents <= 0 {
		return invalid("A paybill confirmation needs a transaction id and an amount.")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	tenantID, err := schoolForPaybill(ctx, tx, strings.TrimSpace(c.BusinessShortCode), c.BillRefNumber)
	if err != nil {
		return err
	}

	// The same money entered by hand earlier must not be counted twice.
	var manual *string
	err = tx.QueryRow(ctx, `
		SELECT receipt_number FROM payments
		WHERE tenant_id = $1 AND channel = 'mpesa' AND status = 'completed'
		  AND (upper(reference) = $2 OR upper(mpesa_receipt) = $2) LIMIT 1`, tenantID, c.TransID).Scan(&manual)
	if err == nil {
		slog.Info("mpesa c2b: already recorded by hand", "trans_id", c.TransID)
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	transTime := time.Now()
	if t, err := time.ParseInLocation("20060102150405", strings.TrimSpace(c.TransTime), mpesa.EAT); err == nil {
		transTime = t
	}
	learnerID, err := learnerForAccount(ctx, tx, tenantID, c.BillRefNumber)
	if err != nil {
		return err
	}

	var inboxID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO mpesa_inbox (tenant_id, trans_id, trans_time, amount_cents, bill_ref, payer_name, payer_phone, short_code, learner_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (trans_id) DO NOTHING
		RETURNING id`,
		tenantID, c.TransID, transTime, c.AmountCents, nullable(c.BillRefNumber), nullable(c.PayerName),
		nullable(c.MSISDN), nullable(c.BusinessShortCode), learnerID,
	).Scan(&inboxID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // a repeat of a confirmation already recorded
	}
	if err != nil {
		return err
	}

	if learnerID != nil {
		rows, err := tx.Query(ctx, `
			SELECT id FROM invoices
			WHERE tenant_id = $1 AND learner_id = $2 AND `+strings.ReplaceAll(owing, "i.", "")+`
			  AND total_cents - discount_cents - paid_cents > 0
			ORDER BY year, term, created_at`, tenantID, *learnerID)
		if err != nil {
			return err
		}
		var invoices []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			invoices = append(invoices, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, invoiceID := range invoices {
			if _, err := allocate(ctx, tx, tenantID, nil, inboxID, invoiceID, 0); err != nil {
				var done *ConflictError
				if errors.As(err, &done) {
					break // nothing left to allocate
				}
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func nullable(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

// allocate puts part of a paybill payment against an invoice, as a completed
// payment with its own receipt number. amount 0 means as much as fits.
func allocate(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, actor *uuid.UUID, inboxID, invoiceID uuid.UUID, amount int64) (uuid.UUID, error) {
	var (
		received, allocated int64
		transID             string
		transTime           time.Time
		payerName, phone    *string
	)
	if err := tx.QueryRow(ctx, `
		SELECT amount_cents, allocated_cents, trans_id, trans_time, payer_name, payer_phone
		FROM mpesa_inbox WHERE tenant_id = $1 AND id = $2 FOR UPDATE`, tenantID, inboxID).
		Scan(&received, &allocated, &transID, &transTime, &payerName, &phone); err != nil {
		return uuid.Nil, notFound(err)
	}
	remaining := received - allocated
	if remaining <= 0 {
		return uuid.Nil, conflict("All of this payment has been allocated already.")
	}

	status, balance, err := lockInvoice(ctx, tx, tenantID, invoiceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return uuid.Nil, invalid("That invoice is not in this school.")
		}
		return uuid.Nil, err
	}
	if status == "void" {
		return uuid.Nil, conflict("That invoice is void.")
	}
	if balance <= 0 {
		return uuid.Nil, conflict("That invoice is already paid in full.")
	}
	if amount == 0 {
		amount = min(remaining, balance)
	}
	if amount < 0 {
		return uuid.Nil, invalid("Enter the amount to allocate.")
	}
	if amount > remaining {
		return uuid.Nil, invalid("Only %s of this payment is left to allocate.", KES(remaining))
	}
	if amount > balance {
		return uuid.Nil, invalid("Only %s is still owed on that invoice.", KES(balance))
	}

	number, err := receiptNumber(ctx, tx, tenantID, transTime)
	if err != nil {
		return uuid.Nil, err
	}
	var paymentID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO payments (tenant_id, invoice_id, amount_cents, channel, status, reference, mpesa_receipt,
			paid_by, phone, paid_at, received_by, receipt_number, inbox_id, notes)
		VALUES ($1, $2, $3, 'mpesa', 'completed', $4, $4, $5, $6, $7, $8, $9, $10, 'Paid to the paybill')
		RETURNING id`,
		tenantID, invoiceID, amount, transID, payerName, phone, transTime, actor, number, inboxID,
	).Scan(&paymentID); err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE mpesa_inbox SET
			allocated_cents = allocated_cents + $3,
			status = CASE WHEN allocated_cents + $3 >= amount_cents THEN 'allocated' ELSE 'part_allocated' END
		WHERE tenant_id = $1 AND id = $2`, tenantID, inboxID, amount); err != nil {
		return uuid.Nil, err
	}
	if err := refreshInvoiceFinance(ctx, tx, tenantID, invoiceID); err != nil {
		return uuid.Nil, err
	}
	return paymentID, nil
}

// Allocate puts a paybill payment, or part of it, against an invoice.
func (s *Service) Allocate(ctx context.Context, tenantID uuid.UUID, actor *uuid.UUID, inboxID uuid.UUID, req AllocateRequest) (*InboxPayment, error) {
	if req.InvoiceID == uuid.Nil {
		return nil, invalid("Choose the invoice to put this payment against.")
	}
	if req.AmountCents < 0 {
		return nil, invalid("Enter the amount to allocate.")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if _, err := allocate(ctx, tx, tenantID, actor, inboxID, req.InvoiceID, req.AmountCents); err != nil {
		return nil, err
	}
	// The payment now belongs to the learner it was allocated to.
	if _, err := tx.Exec(ctx, `
		UPDATE mpesa_inbox SET learner_id = COALESCE(learner_id, (SELECT learner_id FROM invoices WHERE id = $3))
		WHERE tenant_id = $1 AND id = $2`, tenantID, inboxID, req.InvoiceID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetInboxPayment(ctx, tenantID, inboxID)
}

const inboxColumns = `m.id, m.tenant_id, m.trans_id, m.trans_time, m.amount_cents, m.allocated_cents,
	m.bill_ref, m.payer_name, m.payer_phone, m.short_code, m.learner_id, l.full_name, m.status, m.created_at`

const inboxFrom = ` FROM mpesa_inbox m LEFT JOIN learners l ON l.id = m.learner_id `

func scanInbox(row pgx.Row) (*InboxPayment, error) {
	var p InboxPayment
	if err := row.Scan(&p.ID, &p.TenantID, &p.TransID, &p.TransTime, &p.AmountCents, &p.AllocatedCents,
		&p.BillRef, &p.PayerName, &p.PayerPhone, &p.ShortCode, &p.LearnerID, &p.LearnerName, &p.Status, &p.CreatedAt); err != nil {
		return nil, err
	}
	p.RemainingCents = p.AmountCents - p.AllocatedCents
	return &p, nil
}

// ListInbox returns paybill payments, newest first. status "open" means
// anything with money still to allocate.
func (s *Service) ListInbox(ctx context.Context, tenantID uuid.UUID, status, search string, limit, offset int) ([]InboxPayment, error) {
	query := `SELECT ` + inboxColumns + inboxFrom + ` WHERE m.tenant_id = $1`
	args := []any{tenantID}
	switch status {
	case "":
	case "open":
		query += ` AND m.status <> 'allocated'`
	default:
		args = append(args, status)
		query += fmt.Sprintf(` AND m.status = $%d`, len(args))
	}
	if search = strings.TrimSpace(search); search != "" {
		args = append(args, "%"+search+"%")
		n := len(args)
		query += fmt.Sprintf(` AND (m.trans_id ILIKE $%d OR m.bill_ref ILIKE $%d OR m.payer_name ILIKE $%d OR l.full_name ILIKE $%d)`, n, n, n, n)
	}
	limit, offset = pageBounds(limit, offset)
	args = append(args, limit, offset)
	query += fmt.Sprintf(` ORDER BY m.trans_time DESC, m.id LIMIT $%d OFFSET $%d`, len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []InboxPayment{}
	for rows.Next() {
		p, err := scanInbox(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// GetInboxPayment returns one paybill payment.
func (s *Service) GetInboxPayment(ctx context.Context, tenantID, id uuid.UUID) (*InboxPayment, error) {
	p, err := scanInbox(s.pool.QueryRow(ctx,
		`SELECT `+inboxColumns+inboxFrom+` WHERE m.tenant_id = $1 AND m.id = $2`, tenantID, id))
	return p, notFound(err)
}

// InboxAllocations returns the payments made out of one paybill payment.
func (s *Service) InboxAllocations(ctx context.Context, tenantID, inboxID uuid.UUID) ([]Payment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+paymentColumns+paymentFrom+` WHERE p.tenant_id = $1 AND p.inbox_id = $2 ORDER BY p.created_at`,
		tenantID, inboxID)
	if err != nil {
		return nil, err
	}
	return scanPayments(rows)
}
