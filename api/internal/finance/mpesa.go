package finance

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/comms/sms"
	"github.com/shule360/api/pkg/httputil"
	"github.com/shule360/api/pkg/mpesa"
)

// Gateway is the part of Daraja a fee payment needs.
type Gateway interface {
	STKPush(ctx context.Context, phone, amount, accountRef, callbackURL string) (*mpesa.STKPushResponse, error)
	STKQuery(ctx context.Context, checkoutRequestID string) (*mpesa.STKQueryResult, error)
}

// GatewayFactory returns the Daraja account a school collects into, and the
// address Safaricom should report the outcome to.
type GatewayFactory func(ctx context.Context, tenantID uuid.UUID) (gateway Gateway, callbackURL string, err error)

// NotConfiguredError: the school has no usable M-Pesa account.
type NotConfiguredError struct{ Reason string }

func (e *NotConfiguredError) Error() string { return e.Reason }

// SecretResolver is settings.Service.ResolveSecrets.
type SecretResolver interface {
	ResolveSecrets(ctx context.Context, tenantID uuid.UUID, provider string) (config map[string]any, secrets map[string]string, usePlatformDefault bool, err error)
}

// PlatformMpesa is the Daraja account from the API environment, used by
// schools that have not entered their own.
type PlatformMpesa struct {
	ConsumerKey    string
	ConsumerSecret string
	Passkey        string
	ShortCode      string
	BaseURL        string
	CallbackURL    string
}

// Safaricom's two hosts. A school may choose between them; it may not point
// the server anywhere else.
const (
	darajaSandbox = "https://sandbox.safaricom.co.ke"
	darajaLive    = "https://api.safaricom.co.ke"
)

// NewGatewayFactory builds the per-school Daraja lookup.
//
// Credentials: the school's own (Settings → Integrations → M-Pesa) when it has
// saved a consumer key, secret, passkey and shortcode; the platform's
// otherwise. Fees paid through the platform's account land in the platform's
// paybill, so a school collecting its own money needs its own.
//
// The callback address is always the platform's: results must come back to
// this API, whatever a school typed in the field.
func NewGatewayFactory(resolver SecretResolver, platform PlatformMpesa) GatewayFactory {
	clients := newClientCache()
	return func(ctx context.Context, tenantID uuid.UUID) (Gateway, string, error) {
		key, secret, passkey, shortCode := platform.ConsumerKey, platform.ConsumerSecret, platform.Passkey, platform.ShortCode
		baseURL := platform.BaseURL

		if resolver != nil {
			config, secrets, usePlatform, err := resolver.ResolveSecrets(ctx, tenantID, "mpesa")
			if err != nil {
				return nil, "", err
			}
			if !usePlatform {
				ownKey, ownSecret := strings.TrimSpace(secrets["consumer_key"]), strings.TrimSpace(secrets["consumer_secret"])
				ownPasskey := strings.TrimSpace(secrets["passkey"])
				ownShort := configValue(config, secrets, "shortcode")
				if ownKey != "" && ownSecret != "" && ownPasskey != "" && ownShort != "" {
					key, secret, passkey, shortCode = ownKey, ownSecret, ownPasskey, ownShort
					// The platform's base URL stays in force when it is set to
					// something that is not Safaricom (the local stand-in).
					if platform.BaseURL == "" || platform.BaseURL == darajaSandbox || platform.BaseURL == darajaLive {
						switch strings.TrimRight(configValue(config, secrets, "base_url"), "/") {
						case darajaLive:
							baseURL = darajaLive
						default:
							baseURL = darajaSandbox
						}
					}
				}
			}
		}

		if !usable(key) || !usable(secret) || !usable(passkey) || !usable(shortCode) {
			return nil, "", &NotConfiguredError{Reason: "M-Pesa is not set up for this school: add the Daraja consumer key, secret, passkey and paybill under Settings → Integrations."}
		}
		if strings.TrimSpace(platform.CallbackURL) == "" {
			return nil, "", &NotConfiguredError{Reason: "M-Pesa cannot report results to this server yet: MPESA_CALLBACK_URL is not set."}
		}
		return clients.get(key, secret, passkey, shortCode, baseURL), platform.CallbackURL, nil
	}
}

func usable(v string) bool {
	v = strings.TrimSpace(v)
	return v != "" && !strings.HasPrefix(v, "your-") && !strings.HasPrefix(v, "your_") && !strings.HasPrefix(v, "REPLACE_")
}

func configValue(config map[string]any, secrets map[string]string, key string) string {
	if v, ok := config[key].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(secrets[key])
}

// MpesaService requests fee payments by STK push and settles them.
//
// A request is recorded and committed before Safaricom is called, so a
// payment the parent approves can always be matched. It is completed only
// when Safaricom says so, by its callback or — when the callback never
// arrives — by asking. A request whose outcome cannot be established is
// marked failed with OUTCOME_UNKNOWN and is never retried by itself: the
// bursar is told to check before asking the parent again.
type MpesaService struct {
	pool     *pgxpool.Pool
	gateways GatewayFactory
	finance  *Service
}

// NewMpesaService creates the M-Pesa service.
func NewMpesaService(pool *pgxpool.Pool, gateways GatewayFactory) *MpesaService {
	return &MpesaService{pool: pool, gateways: gateways, finance: NewService(pool)}
}

// InitiateSTKPush asks a parent's phone to pay towards an invoice.
func (s *MpesaService) InitiateSTKPush(ctx context.Context, tenantID uuid.UUID, actor *uuid.UUID, req MpesaStkRequest) (*Payment, error) {
	if req.InvoiceID == uuid.Nil {
		return nil, invalid("Choose the invoice this payment is for.")
	}
	if req.AmountCents <= 0 {
		return nil, invalid("Enter the amount to request.")
	}
	if req.AmountCents%100 != 0 {
		return nil, invalid("M-Pesa takes whole shillings only. Enter an amount without cents.")
	}
	e164, err := sms.NormalizeKenyanPhone(req.Phone)
	if err != nil {
		return nil, invalid("Enter the parent's Safaricom number, for example 0712 345 678.")
	}
	phone := strings.TrimPrefix(e164, "+")

	if existing, err := s.finance.paymentByKey(ctx, tenantID, req.IdempotencyKey); err != nil || existing != nil {
		return existing, err
	}

	gateway, callbackURL, err := s.gateways(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 1. Record the request, and commit.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	status, balance, err := lockInvoice(ctx, tx, tenantID, req.InvoiceID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, invalid("That invoice is not in this school.")
		}
		return nil, err
	}
	if status == "void" {
		return nil, conflict("This invoice is void; a payment cannot be requested for it.")
	}
	if balance <= 0 {
		return nil, conflict("This invoice is already paid in full.")
	}
	if req.AmountCents > balance {
		return nil, invalid("Only %s is still owed on this invoice. Request at most that.", KES(balance))
	}
	var waiting int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM payments
		WHERE tenant_id = $1 AND invoice_id = $2 AND channel = 'mpesa' AND status = 'pending'`,
		tenantID, req.InvoiceID).Scan(&waiting); err != nil {
		return nil, err
	}
	if waiting > 0 {
		return nil, conflict("An M-Pesa request for this invoice is still waiting for the parent. Wait for it to finish before sending another.")
	}

	var invoiceNumber string
	if err := tx.QueryRow(ctx,
		`SELECT invoice_number FROM invoices WHERE tenant_id = $1 AND id = $2`, tenantID, req.InvoiceID).Scan(&invoiceNumber); err != nil {
		return nil, err
	}

	var key *string
	if req.IdempotencyKey != "" {
		key = &req.IdempotencyKey
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO payments (tenant_id, invoice_id, amount_cents, channel, status, phone, paid_by, received_by,
			attempted_at, idempotency_key)
		VALUES ($1, $2, $3, 'mpesa', 'pending', $4, $5, $6, now(), $7)
		RETURNING id`,
		tenantID, req.InvoiceID, req.AmountCents, "+"+phone, trimmed(req.PaidBy), actor, key,
	).Scan(&id)
	if err != nil {
		if httputil.IsUniqueViolation(err) {
			if existing, lookupErr := s.finance.paymentByKey(ctx, tenantID, req.IdempotencyKey); lookupErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// 2. Ask Safaricom, outside any transaction.
	accountRef := invoiceNumber
	if len(accountRef) > 12 {
		// Daraja allows 12 characters: keep the running number, the part that
		// identifies the invoice.
		accountRef = accountRef[len(accountRef)-12:]
	}
	stk, err := gateway.STKPush(ctx, phone, fmt.Sprintf("%d", req.AmountCents/100), accountRef, callbackURL)

	// 3. Record what happened.
	record := context.WithoutCancel(ctx)
	var rejected *mpesa.RejectedError
	switch {
	case err == nil:
		if _, err := s.pool.Exec(record, `
			UPDATE payments SET checkout_request_id = $3, merchant_request_id = $4
			WHERE tenant_id = $1 AND id = $2`, tenantID, id, stk.CheckoutRequestID, stk.MerchantRequestID); err != nil {
			slog.Error("mpesa: could not store checkout id; the callback will not match",
				"payment_id", id, "checkout_request_id", stk.CheckoutRequestID, "error", err)
			return nil, err
		}
	case errors.As(err, &rejected):
		// Safaricom answered and refused: the parent was never asked.
		if _, dbErr := s.pool.Exec(record, `
			UPDATE payments SET status = 'failed', failure_code = 'REJECTED', mpesa_result_desc = $3
			WHERE tenant_id = $1 AND id = $2 AND status = 'pending'`, tenantID, id, rejected.Body); dbErr != nil {
			slog.Error("mpesa: could not mark a refused request failed", "payment_id", id, "error", dbErr)
		}
	default:
		// No answer. The prompt may be on the phone, and without a checkout id
		// the result cannot be matched: say so rather than guess.
		slog.Error("mpesa: stk push outcome unknown", "payment_id", id, "error", err)
		if _, dbErr := s.pool.Exec(record, `
			UPDATE payments SET status = 'failed', failure_code = 'OUTCOME_UNKNOWN',
				mpesa_result_desc = 'Safaricom did not answer. The parent may still have been asked to pay.'
			WHERE tenant_id = $1 AND id = $2 AND status = 'pending'`, tenantID, id); dbErr != nil {
			slog.Error("mpesa: could not mark an unanswered request", "payment_id", id, "error", dbErr)
		}
	}
	return s.finance.GetPayment(record, tenantID, id)
}

// STKResult is the outcome Safaricom reports for an STK request.
type STKResult struct {
	CheckoutRequestID string
	ResultCode        string
	ResultDesc        string
	Receipt           string
}

// ConfirmSTKPush records the outcome of an STK request. It returns false when
// nothing changed: the request is unknown here, or was settled already
// (Safaricom repeats callbacks).
func (s *MpesaService) ConfirmSTKPush(ctx context.Context, result STKResult) (bool, error) {
	if result.CheckoutRequestID == "" {
		return false, invalid("CheckoutRequestID is required.")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	var (
		id, tenantID, invoiceID uuid.UUID
		status                  string
		receipt                 *string
	)
	err = tx.QueryRow(ctx, `
		SELECT id, tenant_id, invoice_id, status, mpesa_receipt FROM payments
		WHERE checkout_request_id = $1 AND channel = 'mpesa'
		FOR UPDATE`, result.CheckoutRequestID).Scan(&id, &tenantID, &invoiceID, &status, &receipt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	paid := result.ResultCode == "0"
	if status != "pending" {
		// Settled already. The one thing a late callback may still add is the
		// M-Pesa confirmation code, when the status query got there first.
		if status == "completed" && paid && receipt == nil && result.Receipt != "" {
			if _, err := tx.Exec(ctx, `UPDATE payments SET mpesa_receipt = $2, reference = $2 WHERE id = $1`, id, result.Receipt); err != nil {
				return false, err
			}
			return true, tx.Commit(ctx)
		}
		return false, nil
	}

	if !paid {
		if _, err := tx.Exec(ctx, `
			UPDATE payments SET status = 'failed', failure_code = $2, mpesa_result_code = $3, mpesa_result_desc = $4
			WHERE id = $1`, id, failureCode(result.ResultCode), result.ResultCode, result.ResultDesc); err != nil {
			return false, err
		}
		return true, tx.Commit(ctx)
	}

	if _, _, err := lockInvoice(ctx, tx, tenantID, invoiceID); err != nil {
		return false, err
	}
	number, err := receiptNumber(ctx, tx, tenantID, time.Now())
	if err != nil {
		return false, err
	}
	var code *string
	if result.Receipt != "" {
		code = &result.Receipt
	}
	if _, err := tx.Exec(ctx, `
		UPDATE payments SET status = 'completed', paid_at = now(), receipt_number = $2,
			mpesa_receipt = $3, reference = COALESCE($3, reference),
			mpesa_result_code = $4, mpesa_result_desc = $5, failure_code = NULL
		WHERE id = $1`, id, number, code, result.ResultCode, result.ResultDesc); err != nil {
		return false, err
	}
	if err := refreshInvoiceFinance(ctx, tx, tenantID, invoiceID); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// failureCode names the Daraja result codes a bursar will meet.
func failureCode(resultCode string) string {
	switch resultCode {
	case "1":
		return "INSUFFICIENT_FUNDS"
	case "1032":
		return "CANCELLED"
	case "1037":
		return "NO_RESPONSE"
	case "2001":
		return "WRONG_PIN"
	case "1019":
		return "EXPIRED"
	case "1001":
		return "BUSY"
	default:
		return "DECLINED"
	}
}

// Run settles STK requests whose callback never arrived, until ctx ends.
func (s *MpesaService) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Sweep(ctx); err != nil && ctx.Err() == nil {
				slog.Error("mpesa: sweep failed", "error", err)
			}
		}
	}
}

// An STK prompt stays on the phone for about a minute; after this long a
// request with no callback is asked about.
const stkGrace = 90 * time.Second

// After this long without an answer the request is closed as unknown.
const stkGiveUp = 24 * time.Hour

// Sweep asks Safaricom about every pending request old enough to have been
// answered, and records what it says.
func (s *MpesaService) Sweep(ctx context.Context) error {
	return s.sweep(ctx, stkGrace)
}

func (s *MpesaService) sweep(ctx context.Context, grace time.Duration) error {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, checkout_request_id, created_at FROM payments
		WHERE channel = 'mpesa' AND status = 'pending' AND created_at < now() - make_interval(secs => $1)
		ORDER BY created_at
		LIMIT 100`, grace.Seconds())
	if err != nil {
		return err
	}
	type pendingRow struct {
		id, tenantID uuid.UUID
		checkoutID   *string
		createdAt    time.Time
	}
	var pending []pendingRow
	for rows.Next() {
		var p pendingRow
		if err := rows.Scan(&p.id, &p.tenantID, &p.checkoutID, &p.createdAt); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, p := range pending {
		expired := time.Since(p.createdAt) > stkGiveUp
		if p.checkoutID == nil || *p.checkoutID == "" {
			// The server stopped between recording the request and hearing
			// from Safaricom. Whether the parent was asked is not known.
			s.closeUnknown(ctx, p.id)
			continue
		}
		gateway, _, err := s.gateways(ctx, p.tenantID)
		if err != nil {
			if expired {
				s.closeUnknown(ctx, p.id)
			}
			continue
		}
		result, err := gateway.STKQuery(ctx, *p.checkoutID)
		if err != nil || !result.Final {
			if err != nil {
				slog.Warn("mpesa: status query failed", "payment_id", p.id, "error", err)
			}
			if expired {
				s.closeUnknown(ctx, p.id)
			}
			continue
		}
		if _, err := s.ConfirmSTKPush(ctx, STKResult{
			CheckoutRequestID: *p.checkoutID, ResultCode: result.ResultCode, ResultDesc: result.ResultDesc,
		}); err != nil {
			slog.Error("mpesa: could not record a queried result", "payment_id", p.id, "error", err)
		}
	}
	return nil
}

func (s *MpesaService) closeUnknown(ctx context.Context, id uuid.UUID) {
	if _, err := s.pool.Exec(ctx, `
		UPDATE payments SET status = 'failed', failure_code = 'OUTCOME_UNKNOWN',
			mpesa_result_desc = 'No result was received from Safaricom. Check the M-Pesa statement before asking the parent to pay again.'
		WHERE id = $1 AND status = 'pending'`, id); err != nil {
		slog.Error("mpesa: could not close an unanswered request", "payment_id", id, "error", err)
	}
}
