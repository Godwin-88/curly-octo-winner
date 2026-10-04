package finance

// These tests run against a real Postgres with the migrations applied, because
// what they check — what is committed before Safaricom is called, what a
// repeated callback changes, what another school can see — lives in the
// database.
//
// They are skipped unless TEST_DATABASE_URL is set. Each test creates its own
// school, so they do not interfere with each other or with seed data.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/pkg/mpesa"
)

// fakeDaraja stands in for Safaricom.
type fakeDaraja struct {
	mu      sync.Mutex
	pushes  []string // phone numbers asked
	pushErr error
	query   *mpesa.STKQueryResult
	// during runs while the push is in flight, to see what is committed.
	during func()
}

func (f *fakeDaraja) STKPush(_ context.Context, phone, amount, accountRef, callbackURL string) (*mpesa.STKPushResponse, error) {
	f.mu.Lock()
	f.pushes = append(f.pushes, phone+" "+amount+" "+accountRef)
	during, err := f.during, f.pushErr
	f.mu.Unlock()
	if during != nil {
		during()
	}
	if err != nil {
		return nil, err
	}
	return &mpesa.STKPushResponse{
		MerchantRequestID: "m-" + uuid.NewString(), CheckoutRequestID: "ws_CO_" + uuid.NewString(), ResponseCode: "0",
	}, nil
}

func (f *fakeDaraja) STKQuery(context.Context, string) (*mpesa.STKQueryResult, error) {
	if f.query == nil {
		return &mpesa.STKQueryResult{Final: false}, nil
	}
	return f.query, nil
}

type fixture struct {
	t        *testing.T
	pool     *pgxpool.Pool
	tenantID uuid.UUID
	staffID  uuid.UUID
	svc      *Service
	mpesa    *MpesaService
	daraja   *fakeDaraja
	n        int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	f := &fixture{t: t, pool: pool, daraja: &fakeDaraja{}}
	f.tenantID = f.newTenant()
	f.staffID = f.staff(f.tenantID)
	f.svc = NewService(pool)
	f.mpesa = NewMpesaService(pool, func(context.Context, uuid.UUID) (Gateway, string, error) {
		return f.daraja, "https://example.test/callback", nil
	})
	return f
}

func (f *fixture) newTenant() uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	if err := f.pool.QueryRow(context.Background(),
		`INSERT INTO tenants (name, slug) VALUES ('Test School', $1) RETURNING id`, "t-"+uuid.NewString()).Scan(&id); err != nil {
		f.t.Fatalf("insert tenant: %v", err)
	}
	f.t.Cleanup(func() {
		if _, err := f.pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, id); err != nil {
			f.t.Errorf("remove test school: %v", err)
		}
	})
	return id
}

func (f *fixture) staff(tenantID uuid.UUID) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO staff (tenant_id, full_name, email, role) VALUES ($1, 'Test Bursar', $2, 'bursar') RETURNING id`,
		tenantID, uuid.NewString()+"@example.test").Scan(&id); err != nil {
		f.t.Fatalf("insert staff: %v", err)
	}
	return id
}

// learner adds an active learner and returns the id and UPI.
func (f *fixture) learner(tenantID uuid.UUID, grade string) (uuid.UUID, string) {
	f.t.Helper()
	f.n++
	upi := strings.ToUpper("T" + strings.ReplaceAll(uuid.NewString(), "-", "")[:15])
	var id uuid.UUID
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO learners (tenant_id, upi, full_name, grade, stream) VALUES ($1, $2, $3, $4, 'North') RETURNING id`,
		tenantID, upi, fmt.Sprintf("Learner %d", f.n), grade).Scan(&id); err != nil {
		f.t.Fatalf("insert learner: %v", err)
	}
	return id, upi
}

func optional() *bool { v := true; return &v }

// structure creates Grade 4 Term 1 fees: tuition 15,000, activity 2,500 and an
// optional transport 6,000.
func (f *fixture) structure() *FeeStructure {
	f.t.Helper()
	fs, err := f.svc.CreateFeeStructure(context.Background(), f.tenantID, &f.staffID, CreateFeeStructureRequest{
		Name: "Grade 4 Term 1", Grade: "Grade 4", Term: 1, Year: 2026,
		Items: []FeeItemInput{
			{Name: "Tuition", AmountCents: 1500000, ItemType: "tuition"},
			{Name: "Activity", AmountCents: 250000, ItemType: "activity"},
			{Name: "Transport", AmountCents: 600000, ItemType: "transport", IsOptional: optional()},
		},
	})
	if err != nil {
		f.t.Fatalf("CreateFeeStructure: %v", err)
	}
	return fs
}

// invoice bills a new learner 17,500.
func (f *fixture) invoice() *Invoice {
	f.t.Helper()
	fs := f.structure()
	learnerID, _ := f.learner(f.tenantID, "Grade 4")
	inv, err := f.svc.CreateInvoice(context.Background(), f.tenantID, &f.staffID, CreateInvoiceRequest{
		LearnerID: learnerID, FeeStructureID: &fs.ID,
	})
	if err != nil {
		f.t.Fatalf("CreateInvoice: %v", err)
	}
	return inv
}

func (f *fixture) reload(id uuid.UUID) *Invoice {
	f.t.Helper()
	inv, err := f.svc.GetInvoice(context.Background(), f.tenantID, id)
	if err != nil {
		f.t.Fatalf("GetInvoice: %v", err)
	}
	return inv
}

func (f *fixture) cash(invoiceID uuid.UUID, cents int64) *Payment {
	f.t.Helper()
	pay, err := f.svc.CreatePayment(context.Background(), f.tenantID, &f.staffID, CreatePaymentRequest{
		InvoiceID: invoiceID, AmountCents: cents, Channel: "cash",
	})
	if err != nil {
		f.t.Fatalf("CreatePayment: %v", err)
	}
	return pay
}

func wantInvalid(t *testing.T, err error, contains string) {
	t.Helper()
	var v *ValidationError
	if !errors.As(err, &v) {
		t.Fatalf("want a validation error containing %q, got %v", contains, err)
	}
	if !strings.Contains(v.Message, contains) {
		t.Fatalf("validation message %q does not contain %q", v.Message, contains)
	}
}

func wantConflict(t *testing.T, err error, contains string) {
	t.Helper()
	var c *ConflictError
	if !errors.As(err, &c) {
		t.Fatalf("want a conflict containing %q, got %v", contains, err)
	}
	if !strings.Contains(c.Message, contains) {
		t.Fatalf("conflict message %q does not contain %q", c.Message, contains)
	}
}

// --- Billing ---

func TestInvoiceLeavesOutOptionalItemsUnlessAsked(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	if inv.TotalCents != 1750000 || len(inv.Items) != 2 {
		t.Fatalf("total = %d with %d items, want 1750000 with 2", inv.TotalCents, len(inv.Items))
	}
	if inv.Status != "unpaid" || inv.BalanceCents != 1750000 {
		t.Fatalf("status %s balance %d", inv.Status, inv.BalanceCents)
	}
	if !strings.HasPrefix(inv.InvoiceNumber, "INV-2026-") {
		t.Fatalf("invoice number %q", inv.InvoiceNumber)
	}
}

func TestLearnerIsBilledOncePerTerm(t *testing.T) {
	f := newFixture(t)
	fs := f.structure()
	learnerID, _ := f.learner(f.tenantID, "Grade 4")
	req := CreateInvoiceRequest{LearnerID: learnerID, FeeStructureID: &fs.ID}
	if _, err := f.svc.CreateInvoice(context.Background(), f.tenantID, &f.staffID, req); err != nil {
		t.Fatalf("first invoice: %v", err)
	}
	_, err := f.svc.CreateInvoice(context.Background(), f.tenantID, &f.staffID, req)
	wantConflict(t, err, "already has an invoice")
}

func TestVoidedInvoiceFreesTheTermForANewOne(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	if _, err := f.svc.VoidInvoice(context.Background(), f.tenantID, &f.staffID, inv.ID, "Wrong fee structure"); err != nil {
		t.Fatalf("void: %v", err)
	}
	if _, err := f.svc.CreateInvoice(context.Background(), f.tenantID, &f.staffID, CreateInvoiceRequest{
		LearnerID: inv.LearnerID, FeeStructureID: inv.FeeStructureID,
	}); err != nil {
		t.Fatalf("re-invoice after void: %v", err)
	}
}

func TestInvoiceRefusesAnotherGradesFeeStructure(t *testing.T) {
	f := newFixture(t)
	fs := f.structure()
	learnerID, _ := f.learner(f.tenantID, "Grade 5")
	_, err := f.svc.CreateInvoice(context.Background(), f.tenantID, &f.staffID, CreateInvoiceRequest{LearnerID: learnerID, FeeStructureID: &fs.ID})
	wantInvalid(t, err, "is in Grade 5")
}

func TestInvoiceRefusesAnotherSchoolsLearner(t *testing.T) {
	f := newFixture(t)
	fs := f.structure()
	other := f.newTenant()
	learnerID, _ := f.learner(other, "Grade 4")
	_, err := f.svc.CreateInvoice(context.Background(), f.tenantID, &f.staffID, CreateInvoiceRequest{LearnerID: learnerID, FeeStructureID: &fs.ID})
	wantInvalid(t, err, "not in this school")
}

func TestBulkInvoiceBillsEachLearnerOnce(t *testing.T) {
	f := newFixture(t)
	fs := f.structure()
	for i := 0; i < 3; i++ {
		f.learner(f.tenantID, "Grade 4")
	}
	f.learner(f.tenantID, "Grade 5")

	dry, err := f.svc.BulkInvoice(context.Background(), f.tenantID, &f.staffID, BulkInvoiceRequest{FeeStructureID: fs.ID, DryRun: true})
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if dry.Created != 3 || dry.TotalCents != 3*1750000 {
		t.Fatalf("dry run = %+v", dry)
	}
	if got, _ := f.svc.ListInvoices(context.Background(), f.tenantID, InvoiceFilter{}); len(got) != 0 {
		t.Fatalf("a dry run wrote %d invoices", len(got))
	}

	first, err := f.svc.BulkInvoice(context.Background(), f.tenantID, &f.staffID, BulkInvoiceRequest{FeeStructureID: fs.ID})
	if err != nil {
		t.Fatalf("bulk: %v", err)
	}
	if first.Created != 3 || first.AlreadyInvoiced != 0 {
		t.Fatalf("first run = %+v", first)
	}
	second, err := f.svc.BulkInvoice(context.Background(), f.tenantID, &f.staffID, BulkInvoiceRequest{FeeStructureID: fs.ID})
	if err != nil {
		t.Fatalf("second bulk: %v", err)
	}
	if second.Created != 0 || second.AlreadyInvoiced != 3 {
		t.Fatalf("second run = %+v", second)
	}

	invoices, _ := f.svc.ListInvoices(context.Background(), f.tenantID, InvoiceFilter{})
	numbers := map[string]bool{}
	for _, inv := range invoices {
		numbers[inv.InvoiceNumber] = true
	}
	if len(invoices) != 3 || len(numbers) != 3 {
		t.Fatalf("%d invoices with %d distinct numbers, want 3 and 3", len(invoices), len(numbers))
	}
}

func TestAddingAFeeItemUpdatesTheTotal(t *testing.T) {
	f := newFixture(t)
	fs := f.structure()
	got, err := f.svc.AddFeeItem(context.Background(), f.tenantID, fs.ID, FeeItemInput{Name: "Lunch", AmountCents: 500000})
	if err != nil {
		t.Fatalf("AddFeeItem: %v", err)
	}
	if got.TotalCents != fs.TotalCents+500000 {
		t.Fatalf("total = %d, want %d", got.TotalCents, fs.TotalCents+500000)
	}
	_, err = f.svc.AddFeeItem(context.Background(), f.tenantID, fs.ID, FeeItemInput{Name: "Lunch", AmountCents: 1})
	wantConflict(t, err, "already has an item called Lunch")
}

// --- Payments ---

func TestPaymentUpdatesTheInvoiceAndGetsAReceipt(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()

	first := f.cash(inv.ID, 1000000)
	if first.Status != "completed" || first.ReceiptNumber == nil {
		t.Fatalf("payment = %+v", first)
	}
	if first.ReceivedBy == nil || *first.ReceivedBy != f.staffID {
		t.Fatalf("received_by = %v, want the signed-in staff member", first.ReceivedBy)
	}
	if got := f.reload(inv.ID); got.Status != "partially_paid" || got.BalanceCents != 750000 {
		t.Fatalf("after part payment: status %s balance %d", got.Status, got.BalanceCents)
	}

	second := f.cash(inv.ID, 750000)
	if got := f.reload(inv.ID); got.Status != "paid" || got.BalanceCents != 0 {
		t.Fatalf("after full payment: status %s balance %d", got.Status, got.BalanceCents)
	}
	if *first.ReceiptNumber == *second.ReceiptNumber {
		t.Fatalf("two payments share receipt number %s", *first.ReceiptNumber)
	}
}

func TestPaymentBeyondTheBalanceIsRefused(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	_, err := f.svc.CreatePayment(context.Background(), f.tenantID, &f.staffID, CreatePaymentRequest{
		InvoiceID: inv.ID, AmountCents: 1750001, Channel: "cash",
	})
	wantInvalid(t, err, "Only KES 17,500.00 is still owed")

	f.cash(inv.ID, 1750000)
	_, err = f.svc.CreatePayment(context.Background(), f.tenantID, &f.staffID, CreatePaymentRequest{
		InvoiceID: inv.ID, AmountCents: 100, Channel: "cash",
	})
	wantConflict(t, err, "already paid in full")
}

func TestRefusedPaymentLeavesNoGapInReceiptNumbers(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	first := f.cash(inv.ID, 100000)
	_, _ = f.svc.CreatePayment(context.Background(), f.tenantID, &f.staffID, CreatePaymentRequest{
		InvoiceID: inv.ID, AmountCents: 99999999, Channel: "cash",
	})
	second := f.cash(inv.ID, 100000)
	var a, b int
	fmt.Sscanf(strings.TrimPrefix(*first.ReceiptNumber, "RCT-"), "%d-%d", new(int), &a)
	fmt.Sscanf(strings.TrimPrefix(*second.ReceiptNumber, "RCT-"), "%d-%d", new(int), &b)
	if b != a+1 {
		t.Fatalf("receipts %s then %s: the sequence has a gap", *first.ReceiptNumber, *second.ReceiptNumber)
	}
}

func TestBankPaymentNeedsAReferenceAndItIsUsedOnce(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	_, err := f.svc.CreatePayment(context.Background(), f.tenantID, &f.staffID, CreatePaymentRequest{
		InvoiceID: inv.ID, AmountCents: 100000, Channel: "bank",
	})
	wantInvalid(t, err, "bank reference")

	ref := "ft26001abc"
	req := CreatePaymentRequest{InvoiceID: inv.ID, AmountCents: 100000, Channel: "bank", Reference: &ref}
	if _, err := f.svc.CreatePayment(context.Background(), f.tenantID, &f.staffID, req); err != nil {
		t.Fatalf("bank payment: %v", err)
	}
	_, err = f.svc.CreatePayment(context.Background(), f.tenantID, &f.staffID, req)
	wantConflict(t, err, "FT26001ABC is already recorded")
}

func TestSamePaymentSubmittedTwiceIsRecordedOnce(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	req := CreatePaymentRequest{InvoiceID: inv.ID, AmountCents: 500000, Channel: "cash", IdempotencyKey: uuid.NewString()}
	first, err := f.svc.CreatePayment(context.Background(), f.tenantID, &f.staffID, req)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := f.svc.CreatePayment(context.Background(), f.tenantID, &f.staffID, req)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("the double submit created a second payment")
	}
	if got := f.reload(inv.ID); got.PaidCents != 500000 {
		t.Fatalf("paid = %d, want 500000", got.PaidCents)
	}
}

func TestReversalNeedsAReasonAndReopensTheInvoice(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	pay := f.cash(inv.ID, 1750000)

	_, err := f.svc.ReversePayment(context.Background(), f.tenantID, &f.staffID, pay.ID, "  ")
	wantInvalid(t, err, "Say why")

	reversed, err := f.svc.ReversePayment(context.Background(), f.tenantID, &f.staffID, pay.ID, "Cheque bounced")
	if err != nil {
		t.Fatalf("reverse: %v", err)
	}
	if reversed.Status != "reversed" || reversed.ReversalReason == nil || reversed.ReversedAt == nil {
		t.Fatalf("reversed payment = %+v", reversed)
	}
	if got := f.reload(inv.ID); got.Status != "unpaid" || got.BalanceCents != 1750000 {
		t.Fatalf("after reversal: status %s balance %d", got.Status, got.BalanceCents)
	}
	_, err = f.svc.ReversePayment(context.Background(), f.tenantID, &f.staffID, pay.ID, "again")
	wantConflict(t, err, "already been reversed")
}

func TestInvoiceWithMoneyCannotBeVoidedOrDeleted(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	f.cash(inv.ID, 100000)

	_, err := f.svc.VoidInvoice(context.Background(), f.tenantID, &f.staffID, inv.ID, "mistake")
	wantConflict(t, err, "Reverse them before voiding")

	// The database refuses too, whatever the code does.
	if _, err := f.pool.Exec(context.Background(), `DELETE FROM invoices WHERE id = $1`, inv.ID); err == nil {
		t.Fatalf("an invoice with a payment was deleted")
	}
	_, err = f.svc.CreatePayment(context.Background(), f.tenantID, &f.staffID, CreatePaymentRequest{InvoiceID: inv.ID, AmountCents: 1, Channel: "cash"})
	if err != nil {
		t.Fatalf("the invoice should still take payments: %v", err)
	}
}

func TestDiscountCannotExceedWhatIsOwed(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	reason := "Sibling"
	_, err := f.svc.CreateDiscount(context.Background(), f.tenantID, &f.staffID, inv.ID, CreateDiscountRequest{AmountCents: 1750001, Reason: &reason})
	wantInvalid(t, err, "cannot be more than that")
	_, err = f.svc.CreateDiscount(context.Background(), f.tenantID, &f.staffID, inv.ID, CreateDiscountRequest{AmountCents: 1000})
	wantInvalid(t, err, "Say why")

	d, err := f.svc.CreateDiscount(context.Background(), f.tenantID, &f.staffID, inv.ID, CreateDiscountRequest{AmountCents: 250000, DiscountType: "sibling", Reason: &reason})
	if err != nil {
		t.Fatalf("discount: %v", err)
	}
	if d.ApprovedBy == nil || *d.ApprovedBy != f.staffID {
		t.Fatalf("approved_by = %v, want the signed-in staff member", d.ApprovedBy)
	}
	if got := f.reload(inv.ID); got.BalanceCents != 1500000 {
		t.Fatalf("balance = %d, want 1500000", got.BalanceCents)
	}
}

func TestAnotherSchoolCannotSeeOrTouchAnInvoice(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	pay := f.cash(inv.ID, 100000)
	other := f.newTenant()
	otherStaff := f.staff(other)
	ctx := context.Background()

	if _, err := f.svc.GetInvoice(ctx, other, inv.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetInvoice from another school: %v", err)
	}
	if _, err := f.svc.GetPayment(ctx, other, pay.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetPayment from another school: %v", err)
	}
	if _, err := f.svc.ReversePayment(ctx, other, &otherStaff, pay.ID, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ReversePayment from another school: %v", err)
	}
	if _, err := f.svc.VoidInvoice(ctx, other, &otherStaff, inv.ID, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("VoidInvoice from another school: %v", err)
	}
	_, err := f.svc.CreatePayment(ctx, other, &otherStaff, CreatePaymentRequest{InvoiceID: inv.ID, AmountCents: 100, Channel: "cash"})
	wantInvalid(t, err, "not in this school")
	if _, err := f.svc.Statement(ctx, other, inv.LearnerID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Statement from another school: %v", err)
	}
}

func TestSummaryArrearsAndStatementAgree(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	f.cash(inv.ID, 500000)
	ctx := context.Background()

	sum, err := f.svc.Summary(ctx, f.tenantID, 1, 2026)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if sum.BilledCents != 1750000 || sum.CollectedCents != 500000 || sum.OutstandingCents != 1250000 || sum.ByChannel["cash"] != 500000 {
		t.Fatalf("summary = %+v", sum)
	}
	arrears, err := f.svc.Arrears(ctx, f.tenantID, "", "", 0, 0)
	if err != nil || len(arrears) != 1 || arrears[0].BalanceCents != 1250000 {
		t.Fatalf("arrears = %+v, %v", arrears, err)
	}
	st, err := f.svc.Statement(ctx, f.tenantID, inv.LearnerID)
	if err != nil || st.BalanceCents != 1250000 || len(st.Payments) != 1 || len(st.Invoices) != 1 {
		t.Fatalf("statement = %+v, %v", st, err)
	}
}

// --- M-Pesa ---

func (f *fixture) stk(invoiceID uuid.UUID, cents int64) (*Payment, error) {
	return f.mpesa.InitiateSTKPush(context.Background(), f.tenantID, &f.staffID, MpesaStkRequest{
		InvoiceID: invoiceID, Phone: "0712345678", AmountCents: cents,
	})
}

func checkoutOf(t *testing.T, p *Payment) string {
	t.Helper()
	if p.CheckoutRequestID == nil || *p.CheckoutRequestID == "" {
		t.Fatalf("payment has no checkout request id: %+v", p)
	}
	return *p.CheckoutRequestID
}

func TestSTKRequestIsCommittedBeforeSafaricomIsCalled(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	var seen int
	f.daraja.during = func() {
		// A separate connection: it sees only what is committed.
		_ = f.pool.QueryRow(context.Background(),
			`SELECT COUNT(*) FROM payments WHERE invoice_id = $1 AND status = 'pending' AND attempted_at IS NOT NULL`, inv.ID).Scan(&seen)
	}
	pay, err := f.stk(inv.ID, 500000)
	if err != nil {
		t.Fatalf("stk: %v", err)
	}
	if seen != 1 {
		t.Fatalf("while Safaricom was being called %d pending payments were committed, want 1", seen)
	}
	if pay.Status != "pending" || len(f.daraja.pushes) != 1 || !strings.HasPrefix(f.daraja.pushes[0], "254712345678 5000 ") {
		t.Fatalf("payment %s, pushes %v", pay.Status, f.daraja.pushes)
	}
	if got := f.reload(inv.ID); got.PaidCents != 0 {
		t.Fatalf("a request that has not been paid counted as %d paid", got.PaidCents)
	}
}

func TestSTKCallbackCompletesOnceHoweverOftenItArrives(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	pay, err := f.stk(inv.ID, 500000)
	if err != nil {
		t.Fatalf("stk: %v", err)
	}
	result := STKResult{CheckoutRequestID: checkoutOf(t, pay), ResultCode: "0", Receipt: "SJ12ABCDEF"}

	for i, wantChanged := range []bool{true, false, false} {
		changed, err := f.mpesa.ConfirmSTKPush(context.Background(), result)
		if err != nil || changed != wantChanged {
			t.Fatalf("callback %d: changed=%v err=%v, want changed=%v", i+1, changed, err, wantChanged)
		}
	}
	got, _ := f.svc.GetPayment(context.Background(), f.tenantID, pay.ID)
	if got.Status != "completed" || got.ReceiptNumber == nil || got.MpesaReceipt == nil || *got.MpesaReceipt != "SJ12ABCDEF" {
		t.Fatalf("payment after callback = %+v", got)
	}
	if after := f.reload(inv.ID); after.PaidCents != 500000 || after.Status != "partially_paid" {
		t.Fatalf("invoice after three callbacks: paid %d status %s", after.PaidCents, after.Status)
	}
}

func TestCancelledSTKPaysNothing(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	pay, _ := f.stk(inv.ID, 500000)
	if _, err := f.mpesa.ConfirmSTKPush(context.Background(), STKResult{
		CheckoutRequestID: checkoutOf(t, pay), ResultCode: "1032", ResultDesc: "Request cancelled by user",
	}); err != nil {
		t.Fatalf("callback: %v", err)
	}
	got, _ := f.svc.GetPayment(context.Background(), f.tenantID, pay.ID)
	if got.Status != "failed" || got.FailureCode == nil || *got.FailureCode != "CANCELLED" || got.ReceiptNumber != nil {
		t.Fatalf("payment = %+v", got)
	}
	if after := f.reload(inv.ID); after.PaidCents != 0 {
		t.Fatalf("a cancelled request counted as %d paid", after.PaidCents)
	}
	// A late "paid" for a request already closed must not revive it.
	if changed, _ := f.mpesa.ConfirmSTKPush(context.Background(), STKResult{CheckoutRequestID: checkoutOf(t, pay), ResultCode: "0", Receipt: "X"}); changed {
		t.Fatalf("a closed request was reopened by a later callback")
	}
}

func TestSTKRefusedBySafaricomIsMarkedFailed(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	f.daraja.pushErr = &mpesa.RejectedError{StatusCode: 400, Body: "Invalid PhoneNumber"}
	pay, err := f.stk(inv.ID, 500000)
	if err != nil {
		t.Fatalf("stk: %v", err)
	}
	if pay.Status != "failed" || pay.FailureCode == nil || *pay.FailureCode != "REJECTED" {
		t.Fatalf("payment = %+v", pay)
	}
}

func TestSTKWithNoAnswerIsNeverAssumedEitherWay(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	f.daraja.pushErr = errors.New("context deadline exceeded")
	pay, err := f.stk(inv.ID, 500000)
	if err != nil {
		t.Fatalf("stk: %v", err)
	}
	if pay.Status != "failed" || pay.FailureCode == nil || *pay.FailureCode != "OUTCOME_UNKNOWN" {
		t.Fatalf("payment = %+v", pay)
	}
	if after := f.reload(inv.ID); after.PaidCents != 0 {
		t.Fatalf("an unanswered request counted as paid")
	}
}

func TestSTKRefusals(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()

	_, err := f.stk(inv.ID, 500050)
	wantInvalid(t, err, "whole shillings")
	_, err = f.stk(inv.ID, 1750100)
	wantInvalid(t, err, "Only KES 17,500.00 is still owed")
	_, err = f.mpesa.InitiateSTKPush(context.Background(), f.tenantID, &f.staffID, MpesaStkRequest{InvoiceID: inv.ID, Phone: "12345", AmountCents: 100000})
	wantInvalid(t, err, "Safaricom number")
	if len(f.daraja.pushes) != 0 {
		t.Fatalf("Safaricom was called for a refused request: %v", f.daraja.pushes)
	}

	if _, err := f.stk(inv.ID, 500000); err != nil {
		t.Fatalf("stk: %v", err)
	}
	_, err = f.stk(inv.ID, 500000)
	wantConflict(t, err, "still waiting for the parent")
	if len(f.daraja.pushes) != 1 {
		t.Fatalf("a second prompt was sent while the first was waiting")
	}
}

func TestSweepSettlesARequestWhoseCallbackNeverCame(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice()
	pay, _ := f.stk(inv.ID, 500000)

	// Safaricom has no result yet: nothing changes.
	if err := f.mpesa.sweep(context.Background(), 0); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if got, _ := f.svc.GetPayment(context.Background(), f.tenantID, pay.ID); got.Status != "pending" {
		t.Fatalf("status = %s while Safaricom is still processing", got.Status)
	}

	f.daraja.query = &mpesa.STKQueryResult{Final: true, ResultCode: "0", ResultDesc: "ok"}
	if err := f.mpesa.sweep(context.Background(), 0); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	got, _ := f.svc.GetPayment(context.Background(), f.tenantID, pay.ID)
	if got.Status != "completed" || got.ReceiptNumber == nil {
		t.Fatalf("payment after sweep = %+v", got)
	}
	// The callback turns up late: it adds the M-Pesa code and nothing else.
	if _, err := f.mpesa.ConfirmSTKPush(context.Background(), STKResult{CheckoutRequestID: checkoutOf(t, pay), ResultCode: "0", Receipt: "SLATE12345"}); err != nil {
		t.Fatalf("late callback: %v", err)
	}
	got, _ = f.svc.GetPayment(context.Background(), f.tenantID, pay.ID)
	if got.MpesaReceipt == nil || *got.MpesaReceipt != "SLATE12345" {
		t.Fatalf("late callback did not add the M-Pesa code: %+v", got)
	}
	if after := f.reload(inv.ID); after.PaidCents != 500000 {
		t.Fatalf("paid = %d after sweep and late callback, want 500000", after.PaidCents)
	}
}

// --- Paybill ---

func (f *fixture) paybill() string {
	f.t.Helper()
	code := fmt.Sprintf("9%05d", time.Now().UnixNano()%100000)
	if _, err := f.pool.Exec(context.Background(), `UPDATE tenants SET mpesa_shortcode = $2 WHERE id = $1`, f.tenantID, code); err != nil {
		f.t.Fatalf("set paybill: %v", err)
	}
	return code
}

func TestPaybillPaymentIsAllocatedToTheLearnerNamedInTheAccount(t *testing.T) {
	f := newFixture(t)
	code := f.paybill()
	fs := f.structure()
	learnerID, upi := f.learner(f.tenantID, "Grade 4")
	inv, err := f.svc.CreateInvoice(context.Background(), f.tenantID, &f.staffID, CreateInvoiceRequest{LearnerID: learnerID, FeeStructureID: &fs.ID})
	if err != nil {
		t.Fatalf("invoice: %v", err)
	}

	c := C2BConfirmation{
		TransID: "T" + strings.ToUpper(uuid.NewString()[:9]), TransTime: "20261003101500", AmountCents: 2000000,
		BusinessShortCode: code, BillRefNumber: " " + strings.ToLower(upi) + " ", PayerName: "Grace Muthoni",
	}
	for i := 0; i < 2; i++ { // Safaricom repeats confirmations
		if err := f.svc.ReceiveC2B(context.Background(), c); err != nil {
			t.Fatalf("ReceiveC2B %d: %v", i+1, err)
		}
	}

	if got := f.reload(inv.ID); got.Status != "paid" || got.PaidCents != 1750000 {
		t.Fatalf("invoice: status %s paid %d", got.Status, got.PaidCents)
	}
	inbox, err := f.svc.ListInbox(context.Background(), f.tenantID, "", "", 0, 0)
	if err != nil || len(inbox) != 1 {
		t.Fatalf("inbox = %+v, %v", inbox, err)
	}
	// 20,000 came in; 17,500 was owed. The other 2,500 is still the school's
	// money and stays visible until someone allocates it.
	if inbox[0].Status != "part_allocated" || inbox[0].RemainingCents != 250000 {
		t.Fatalf("inbox row = %+v", inbox[0])
	}
}

func TestUnmatchedPaybillPaymentIsKeptAndAllocatedByHand(t *testing.T) {
	f := newFixture(t)
	code := f.paybill()
	inv := f.invoice()
	transID := "U" + strings.ToUpper(uuid.NewString()[:9])
	if err := f.svc.ReceiveC2B(context.Background(), C2BConfirmation{
		TransID: transID, AmountCents: 300000, BusinessShortCode: code, BillRefNumber: "school fees",
	}); err != nil {
		t.Fatalf("ReceiveC2B: %v", err)
	}
	inbox, _ := f.svc.ListInbox(context.Background(), f.tenantID, "open", "", 0, 0)
	if len(inbox) != 1 || inbox[0].Status != "unmatched" {
		t.Fatalf("inbox = %+v", inbox)
	}

	// Entering the same M-Pesa code by hand would count the money twice.
	_, err := f.svc.CreatePayment(context.Background(), f.tenantID, &f.staffID, CreatePaymentRequest{
		InvoiceID: inv.ID, AmountCents: 300000, Channel: "mpesa", Reference: &transID,
	})
	wantConflict(t, err, "already arrived from Safaricom")

	_, err = f.svc.Allocate(context.Background(), f.tenantID, &f.staffID, inbox[0].ID, AllocateRequest{InvoiceID: inv.ID, AmountCents: 300001})
	wantInvalid(t, err, "is left to allocate")

	got, err := f.svc.Allocate(context.Background(), f.tenantID, &f.staffID, inbox[0].ID, AllocateRequest{InvoiceID: inv.ID})
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if got.Status != "allocated" || got.RemainingCents != 0 || got.LearnerID == nil {
		t.Fatalf("inbox after allocation = %+v", got)
	}
	if after := f.reload(inv.ID); after.PaidCents != 300000 {
		t.Fatalf("paid = %d, want 300000", after.PaidCents)
	}
	_, err = f.svc.Allocate(context.Background(), f.tenantID, &f.staffID, inbox[0].ID, AllocateRequest{InvoiceID: inv.ID})
	wantConflict(t, err, "allocated already")

	// Reversing the allocation returns the money to the inbox, not to nowhere.
	allocations, _ := f.svc.InboxAllocations(context.Background(), f.tenantID, inbox[0].ID)
	if _, err := f.svc.ReversePayment(context.Background(), f.tenantID, &f.staffID, allocations[0].ID, "Wrong learner"); err != nil {
		t.Fatalf("reverse: %v", err)
	}
	back, _ := f.svc.GetInboxPayment(context.Background(), f.tenantID, inbox[0].ID)
	if back.Status != "unmatched" || back.RemainingCents != 300000 {
		t.Fatalf("inbox after reversal = %+v", back)
	}
}

func TestPaybillPaymentToAnUnknownPaybillIsReported(t *testing.T) {
	f := newFixture(t)
	err := f.svc.ReceiveC2B(context.Background(), C2BConfirmation{
		TransID: "Z" + strings.ToUpper(uuid.NewString()[:9]), AmountCents: 100000, BusinessShortCode: "000000", BillRefNumber: "x",
	})
	if !errors.Is(err, ErrUnknownPaybill) {
		t.Fatalf("err = %v, want ErrUnknownPaybill", err)
	}
}

func TestParseAmount(t *testing.T) {
	for raw, want := range map[string]int64{"1500.00": 150000, "1500": 150000, "10.5": 1050, "0.01": 1, " 20 ": 2000} {
		got, err := ParseAmount(raw)
		if err != nil || got != want {
			t.Errorf("ParseAmount(%q) = %d, %v; want %d", raw, got, err, want)
		}
	}
	for _, raw := range []string{"", "abc", "1,500", "-5", "1e3"} {
		if _, err := ParseAmount(raw); err == nil {
			t.Errorf("ParseAmount(%q) was accepted", raw)
		}
	}
}

func TestKES(t *testing.T) {
	for cents, want := range map[int64]string{0: "KES 0.00", 1750000: "KES 17,500.00", 99: "KES 0.99", 123456789: "KES 1,234,567.89"} {
		if got := KES(cents); got != want {
			t.Errorf("KES(%d) = %q, want %q", cents, got, want)
		}
	}
}

// What a school charges for is its own list: it adds to it, renames and
// retires entries, and a fee structure can charge anything on it.
func TestFeeItemsAreTheSchoolsOwn(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	uniform, err := f.svc.CreateFeeCategory(ctx, f.tenantID, FeeCategoryInput{Name: "  School   uniform "})
	if err != nil || uniform.Name != "School uniform" || !uniform.IsActive {
		t.Fatalf("create = %+v, %v", uniform, err)
	}
	_, err = f.svc.CreateFeeCategory(ctx, f.tenantID, FeeCategoryInput{Name: "school UNIFORM"})
	wantConflict(t, err, "already has a fee item called")
	_, err = f.svc.CreateFeeCategory(ctx, f.tenantID, FeeCategoryInput{Name: " "})
	wantInvalid(t, err, "Give the fee item a name")

	// A fee structure charges it, with a kind nobody built in.
	fs, err := f.svc.CreateFeeStructure(ctx, f.tenantID, &f.staffID, CreateFeeStructureRequest{
		Name: "PP1 Term 1", Grade: "PP1", Term: 1, Year: 2026,
		Items: []FeeItemInput{{Name: "School uniform", AmountCents: 450000, ItemType: "uniform"}},
	})
	if err != nil || fs.TotalCents != 450000 {
		t.Fatalf("fee structure with a school-defined item: %+v, %v", fs, err)
	}
	if got, _ := f.svc.GetFeeCategory(ctx, f.tenantID, uniform.ID); got.InUse != 1 {
		t.Errorf("in_use = %d, want 1", got.InUse)
	}

	// Retired items leave the list but stay on what was already made.
	off := false
	if _, err := f.svc.UpdateFeeCategory(ctx, f.tenantID, uniform.ID, FeeCategoryInput{IsActive: &off}); err != nil {
		t.Fatal(err)
	}
	if active, _ := f.svc.ListFeeCategories(ctx, f.tenantID, ""); len(active) != 0 {
		t.Errorf("a retired item is still offered: %+v", active)
	}
	if all, _ := f.svc.ListFeeCategories(ctx, f.tenantID, "all"); len(all) != 1 {
		t.Errorf("a retired item disappeared: %+v", all)
	}
	if again, _ := f.svc.GetFeeStructure(ctx, f.tenantID, fs.ID); len(again.Items) != 1 {
		t.Errorf("retiring an item changed an existing fee structure")
	}

	other := f.newTenant()
	if _, err := f.svc.GetFeeCategory(ctx, other, uniform.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("another school read the fee item: %v", err)
	}
	if _, err := f.svc.UpdateFeeCategory(ctx, other, uniform.ID, FeeCategoryInput{Name: "x"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("another school changed the fee item: %v", err)
	}
}
