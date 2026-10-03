package comms

// These tests run against a real Postgres with the migrations applied, because
// what they check — what is committed before the provider is called, what a
// restart finds, what another school can see — lives in the database.
//
// They are skipped unless TEST_DATABASE_URL is set:
//
//	TEST_DATABASE_URL=postgresql://postgres:postgres@localhost:5432/shule360_test go test ./internal/comms/
//
// Each test creates its own school, so they do not interfere with each other
// or with seed data.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/comms/sms"
)

// fakeSender stands in for Africa's Talking.
type fakeSender struct {
	mu     sync.Mutex
	calls  []sms.BulkSMSRequest
	refuse map[string]int // phone -> statusCode to refuse with
	omit   map[string]bool
	err    error
}

func (f *fakeSender) SendBulk(_ context.Context, req sms.BulkSMSRequest) ([]sms.SMSResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req)
	if f.err != nil {
		return nil, f.err
	}
	var results []sms.SMSResult
	for _, phone := range req.To {
		if f.omit[phone] {
			continue
		}
		if code, refused := f.refuse[phone]; refused {
			results = append(results, sms.SMSResult{Phone: phone, Status: "Failed", StatusCode: code, Detail: fmt.Sprintf("Refused%d", code)})
			continue
		}
		results = append(results, sms.SMSResult{
			Phone: phone, Status: "Success", StatusCode: 101, Detail: "Success",
			MessageID: "ATXid_" + uuid.NewString(), Cost: "KES 0.8000",
		})
	}
	return results, nil
}

func (f *fakeSender) sentTo() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var phones []string
	for _, call := range f.calls {
		phones = append(phones, call.To...)
	}
	return phones
}

type fixture struct {
	t        *testing.T
	pool     *pgxpool.Pool
	tenantID uuid.UUID
	staffID  uuid.UUID
	sender   *fakeSender
	disp     *Dispatcher
	svc      *CommsService
	nextNum  int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	f := &fixture{t: t, pool: pool, sender: &fakeSender{refuse: map[string]int{}, omit: map[string]bool{}}}
	f.tenantID = f.newTenant()
	if err := pool.QueryRow(ctx, `
		INSERT INTO staff (tenant_id, full_name, email, role) VALUES ($1, 'Test Principal', $2, 'principal') RETURNING id
	`, f.tenantID, uuid.NewString()+"@example.test").Scan(&f.staffID); err != nil {
		t.Fatalf("insert staff: %v", err)
	}

	f.disp = NewDispatcher(pool, func(context.Context, uuid.UUID) (Sender, error) { return f.sender, nil })
	f.svc = NewCommsService(pool, nil) // tests call Sweep themselves
	return f
}

func (f *fixture) newTenant() uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	slug := "t-" + uuid.NewString()
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO tenants (name, slug) VALUES ('Test School', $1) RETURNING id
	`, slug).Scan(&id); err != nil {
		f.t.Fatalf("insert tenant: %v", err)
	}
	f.t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, id)
	})
	return id
}

// phone returns a fresh valid number, unique within the test run.
func (f *fixture) phone() string {
	f.nextNum++
	return fmt.Sprintf("+2547%08d", (time.Now().UnixNano()/1000+int64(f.nextNum))%100000000)
}

// guardian adds a guardian with one learner in Grade 4 North.
func (f *fixture) guardian(name, phone string) uuid.UUID {
	f.t.Helper()
	ctx := context.Background()
	var id uuid.UUID
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO guardians (tenant_id, full_name, phone_primary) VALUES ($1, $2, $3) RETURNING id
	`, f.tenantID, name, phone).Scan(&id); err != nil {
		f.t.Fatalf("insert guardian: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO learners (tenant_id, upi, full_name, grade, stream, guardian_ids)
		VALUES ($1, $2, $3, 'Grade 4', 'North', ARRAY[$4]::uuid[])
	`, f.tenantID, "T"+strings.ReplaceAll(uuid.NewString(), "-", "")[:15], "Child of "+name, id); err != nil {
		f.t.Fatalf("insert learner: %v", err)
	}
	return id
}

func (f *fixture) contact(name, phone string) {
	f.t.Helper()
	if _, err := f.pool.Exec(context.Background(), `
		INSERT INTO contacts (tenant_id, full_name, phone) VALUES ($1, $2, $3)
	`, f.tenantID, name, phone); err != nil {
		f.t.Fatalf("insert contact: %v", err)
	}
}

func (f *fixture) send(content string) *Message {
	f.t.Helper()
	msg, err := f.svc.CreateAndSend(context.Background(), f.tenantID, Actor{StaffID: &f.staffID}, CreateMessageRequest{
		Channel: "sms", AudienceType: "all_parents", Content: content,
	})
	if err != nil {
		f.t.Fatalf("CreateAndSend: %v", err)
	}
	return msg
}

func (f *fixture) sweep() {
	f.t.Helper()
	if err := f.disp.Sweep(context.Background()); err != nil {
		f.t.Fatalf("Sweep: %v", err)
	}
}

func (f *fixture) message(id uuid.UUID) (*Message, DeliveryStats) {
	f.t.Helper()
	msg, stats, err := f.svc.GetMessage(context.Background(), f.tenantID, id)
	if err != nil {
		f.t.Fatalf("GetMessage: %v", err)
	}
	return msg, stats
}

// logs returns the delivery log keyed by phone.
func (f *fixture) logs(id uuid.UUID) map[string]MessageLog {
	f.t.Helper()
	logs, err := f.svc.GetMessageLogs(context.Background(), f.tenantID, id, "", 500, 0)
	if err != nil {
		f.t.Fatalf("GetMessageLogs: %v", err)
	}
	out := map[string]MessageLog{}
	for _, l := range logs {
		out[l.Phone] = l
	}
	return out
}

func code(l MessageLog) string {
	if l.ErrorCode == nil {
		return ""
	}
	return *l.ErrorCode
}

func wantValidation(t *testing.T, err error, contains string) {
	t.Helper()
	var v *ValidationError
	if !errors.As(err, &v) {
		t.Fatalf("want a validation error containing %q, got %v", contains, err)
	}
	if !strings.Contains(v.Message, contains) {
		t.Fatalf("validation error = %q, want it to contain %q", v.Message, contains)
	}
}

func TestSendRecordsEveryRecipientBeforeSending(t *testing.T) {
	f := newFixture(t)
	a, b := f.phone(), f.phone()
	f.guardian("Mary", a)
	f.guardian("John", b)
	f.guardian("Mary again", a)     // same number as Mary: one SMS, not two
	f.guardian("No phone", "12345") // cannot be texted
	optedOut := f.guardian("Opted out", f.phone())
	if _, err := f.pool.Exec(context.Background(), `UPDATE guardians SET is_sms_opted_out = true WHERE id = $1`, optedOut); err != nil {
		t.Fatal(err)
	}

	msg := f.send("School closes at noon on Friday.")

	// Recorded, nothing sent yet.
	if msg.Status != "sending" {
		t.Fatalf("status = %q, want sending", msg.Status)
	}
	if msg.SentBy == nil || *msg.SentBy != f.staffID {
		t.Fatalf("sent_by = %v, want the signed-in staff member", msg.SentBy)
	}
	if got := len(f.sender.calls); got != 0 {
		t.Fatalf("provider called %d times before dispatch", got)
	}
	logs := f.logs(msg.ID)
	if len(logs) != 3 {
		t.Fatalf("recorded %d recipients, want 3 (two numbers + one invalid)", len(logs))
	}
	if logs[a].Status != "pending" || logs[b].Status != "pending" {
		t.Fatalf("valid recipients should be pending: %+v", logs)
	}
	if l := logs["12345"]; l.Status != "failed" || code(l) != "INVALID_PHONE" {
		t.Fatalf("invalid number = %s/%s, want failed/INVALID_PHONE", l.Status, code(l))
	}

	f.sweep()

	if got := len(f.sender.calls); got != 1 {
		t.Fatalf("provider called %d times, want 1", got)
	}
	if sent := f.sender.sentTo(); len(sent) != 2 {
		t.Fatalf("sent to %v, want exactly the two valid numbers", sent)
	}
	msg, stats := f.message(msg.ID)
	if msg.Status != "sent" || msg.SentAt == nil {
		t.Fatalf("status = %q sent_at = %v, want sent", msg.Status, msg.SentAt)
	}
	// Accepted by the provider is "sent"; only a delivery report makes it "delivered".
	if stats.Sent != 2 || stats.Delivered != 0 || stats.Failed != 1 || stats.Pending != 0 {
		t.Fatalf("stats = %+v, want 2 sent, 0 delivered, 1 failed", stats)
	}
	if msg.DeliveredCount != 0 || msg.FailedCount != 1 {
		t.Fatalf("counts = %d delivered / %d failed, want 0 / 1", msg.DeliveredCount, msg.FailedCount)
	}
	l := f.logs(msg.ID)[a]
	if l.ProviderMessageID == nil || l.SentAt == nil || l.CostCents == nil || *l.CostCents != 80 {
		t.Fatalf("sent row is missing provider id, time or cost: %+v", l)
	}

	// A second sweep finds nothing to do.
	f.sweep()
	if got := len(f.sender.calls); got != 1 {
		t.Fatalf("a second sweep called the provider again (%d calls)", got)
	}
}

func TestDispatchSendsInChunks(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 5; i++ {
		f.guardian(fmt.Sprintf("Parent %d", i), f.phone())
	}
	f.disp.chunkSize = 2
	msg := f.send("Hello")
	f.sweep()

	if got := len(f.sender.calls); got != 3 {
		t.Fatalf("provider called %d times for 5 recipients in chunks of 2, want 3", got)
	}
	if _, stats := f.message(msg.ID); stats.Sent != 5 {
		t.Fatalf("sent = %d, want 5", stats.Sent)
	}
}

func TestPartialRefusalIsRecordedPerRecipient(t *testing.T) {
	f := newFixture(t)
	ok, bad, missing := f.phone(), f.phone(), f.phone()
	f.guardian("Fine", ok)
	f.guardian("Refused", bad)
	f.guardian("Missing", missing)
	f.sender.refuse[bad] = 403
	f.sender.omit[missing] = true

	msg := f.send("Hello")
	f.sweep()

	logs := f.logs(msg.ID)
	if logs[ok].Status != "sent" {
		t.Errorf("accepted number = %s, want sent", logs[ok].Status)
	}
	if logs[bad].Status != "failed" || code(logs[bad]) != "AT_403" {
		t.Errorf("refused number = %s/%s, want failed/AT_403", logs[bad].Status, code(logs[bad]))
	}
	if logs[missing].Status != "failed" || code(logs[missing]) != "NO_RESULT" {
		t.Errorf("number with no result = %s/%s, want failed/NO_RESULT", logs[missing].Status, code(logs[missing]))
	}
	if m, _ := f.message(msg.ID); m.Status != "sent" || m.FailedCount != 2 {
		t.Errorf("message = %s with %d failed, want sent with 2 failed", m.Status, m.FailedCount)
	}
}

func TestProviderTimeoutLeavesOutcomeUnknownAndContinues(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 4; i++ {
		f.guardian(fmt.Sprintf("Parent %d", i), f.phone())
	}
	f.disp.chunkSize = 2
	f.sender.err = errors.New("context deadline exceeded")

	msg := f.send("Hello")
	f.sweep()

	// A timeout says nothing about the next batch, so every chunk is tried.
	if got := len(f.sender.calls); got != 2 {
		t.Fatalf("provider called %d times, want 2", got)
	}
	for phone, l := range f.logs(msg.ID) {
		if l.Status != "failed" || code(l) != "OUTCOME_UNKNOWN" {
			t.Errorf("%s = %s/%s, want failed/OUTCOME_UNKNOWN", phone, l.Status, code(l))
		}
	}
	if m, _ := f.message(msg.ID); m.Status != "failed" {
		t.Errorf("message = %s, want failed", m.Status)
	}
}

func TestProviderRejectionStopsTheSend(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 5; i++ {
		f.guardian(fmt.Sprintf("Parent %d", i), f.phone())
	}
	f.disp.chunkSize = 2
	f.sender.err = &sms.RejectedError{StatusCode: 401, Body: "The supplied authentication is invalid"}

	msg := f.send("Hello")
	f.sweep()

	if got := len(f.sender.calls); got != 1 {
		t.Fatalf("provider called %d times after refusing the credentials, want 1", got)
	}
	for phone, l := range f.logs(msg.ID) {
		if l.Status != "failed" || code(l) != "PROVIDER_REJECTED" {
			t.Errorf("%s = %s/%s, want failed/PROVIDER_REJECTED", phone, l.Status, code(l))
		}
	}
	if m, _ := f.message(msg.ID); m.Status != "failed" || m.FailedCount != 5 {
		t.Errorf("message = %s with %d failed, want failed with 5", m.Status, m.FailedCount)
	}
}

func TestInsufficientBalanceStopsTheSend(t *testing.T) {
	f := newFixture(t)
	var phones []string
	for i := 0; i < 5; i++ {
		p := f.phone()
		phones = append(phones, p)
		f.guardian(fmt.Sprintf("Parent %d", i), p)
		f.sender.refuse[p] = 405
	}
	f.disp.chunkSize = 2

	msg := f.send("Hello")
	f.sweep()

	if got := len(f.sender.calls); got != 1 {
		t.Fatalf("provider called %d times with no credit, want 1", got)
	}
	for _, l := range f.logs(msg.ID) {
		if l.Status != "failed" || code(l) != "AT_405" {
			t.Errorf("%s = %s/%s, want failed/AT_405", l.Phone, l.Status, code(l))
		}
	}
}

func TestRestartNeverResendsAnAttemptedRecipient(t *testing.T) {
	f := newFixture(t)
	interrupted, fresh := f.phone(), f.phone()
	f.guardian("Interrupted", interrupted)
	f.guardian("Fresh", fresh)
	msg := f.send("Hello")

	// The process died after recording the attempt and before the result.
	if _, err := f.pool.Exec(context.Background(), `
		UPDATE message_logs SET attempted_at = now() WHERE message_id = $1 AND phone = $2
	`, msg.ID, interrupted); err != nil {
		t.Fatal(err)
	}

	f.sweep() // the sweep a restarted server runs first

	sent := f.sender.sentTo()
	if len(sent) != 1 || sent[0] != fresh {
		t.Fatalf("sent to %v, want only the recipient that was never attempted", sent)
	}
	logs := f.logs(msg.ID)
	if l := logs[interrupted]; l.Status != "failed" || code(l) != "OUTCOME_UNKNOWN" {
		t.Errorf("interrupted recipient = %s/%s, want failed/OUTCOME_UNKNOWN", l.Status, code(l))
	}
	if logs[fresh].Status != "sent" {
		t.Errorf("fresh recipient = %s, want sent", logs[fresh].Status)
	}
}

func TestMessageBeingSentElsewhereIsLeftAlone(t *testing.T) {
	f := newFixture(t)
	f.guardian("Mary", f.phone())
	msg := f.send("Hello")

	// Another instance holds the message.
	conn, err := f.pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if _, err := conn.Exec(context.Background(), `SELECT pg_advisory_lock($1)`, lockKey(msg.ID)); err != nil {
		t.Fatal(err)
	}

	f.sweep()
	if got := len(f.sender.calls); got != 0 {
		t.Fatalf("provider called %d times while another process held the message", got)
	}

	if _, err := conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, lockKey(msg.ID)); err != nil {
		t.Fatal(err)
	}
	f.sweep()
	if got := len(f.sender.calls); got != 1 {
		t.Fatalf("provider called %d times after the lock was released, want 1", got)
	}
}

func TestScheduledMessageWaitsThenSendsAndCanBeCancelled(t *testing.T) {
	f := newFixture(t)
	f.guardian("Mary", f.phone())
	ctx := context.Background()
	later := time.Now().Add(time.Hour)

	scheduled, err := f.svc.CreateAndSend(ctx, f.tenantID, Actor{StaffID: &f.staffID}, CreateMessageRequest{
		Channel: "sms", AudienceType: "all_parents", Content: "Later", ScheduledAt: &later,
	})
	if err != nil {
		t.Fatal(err)
	}
	if scheduled.Status != "scheduled" {
		t.Fatalf("status = %q, want scheduled", scheduled.Status)
	}
	f.sweep()
	if got := len(f.sender.calls); got != 0 {
		t.Fatalf("a message scheduled for later was sent (%d calls)", got)
	}

	// Its time arrives.
	if _, err := f.pool.Exec(ctx, `UPDATE messages SET scheduled_at = now() - interval '1 second' WHERE id = $1`, scheduled.ID); err != nil {
		t.Fatal(err)
	}
	f.sweep()
	if m, _ := f.message(scheduled.ID); m.Status != "sent" {
		t.Fatalf("due message = %s, want sent", m.Status)
	}

	// A sent message cannot be cancelled.
	wantValidation(t, f.svc.CancelScheduled(ctx, f.tenantID, scheduled.ID), "Only a scheduled message")

	// A scheduled one can, and is then never sent.
	second, err := f.svc.CreateAndSend(ctx, f.tenantID, Actor{StaffID: &f.staffID}, CreateMessageRequest{
		Channel: "sms", AudienceType: "all_parents", Content: "Never", ScheduledAt: &later,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.CancelScheduled(ctx, f.tenantID, second.ID); err != nil {
		t.Fatalf("CancelScheduled: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE messages SET scheduled_at = now() - interval '1 second' WHERE id = $1`, second.ID); err != nil {
		t.Fatal(err)
	}
	f.sweep()
	if got := len(f.sender.calls); got != 1 {
		t.Fatalf("a cancelled message was sent (%d calls, want 1 from the first message)", got)
	}
	m, _ := f.message(second.ID)
	if m.Status != "cancelled" {
		t.Fatalf("status = %s, want cancelled", m.Status)
	}
	for _, l := range f.logs(second.ID) {
		if code(l) != "CANCELLED" {
			t.Errorf("recipient of a cancelled message = %s/%s, want CANCELLED", l.Status, code(l))
		}
	}
}

func TestIdempotencyKeyPreventsADoubleSend(t *testing.T) {
	f := newFixture(t)
	f.guardian("Mary", f.phone())
	ctx := context.Background()
	req := CreateMessageRequest{Channel: "sms", AudienceType: "all_parents", Content: "Once", IdempotencyKey: uuid.NewString()}

	first, err := f.svc.CreateAndSend(ctx, f.tenantID, Actor{StaffID: &f.staffID}, req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.svc.CreateAndSend(ctx, f.tenantID, Actor{StaffID: &f.staffID}, req)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("the same key created two messages: %s and %s", first.ID, second.ID)
	}
	f.sweep()
	if sent := f.sender.sentTo(); len(sent) != 1 {
		t.Fatalf("sent %d SMS for a double-clicked send, want 1", len(sent))
	}

	// The key belongs to the school: another school may use the same value.
	other := f.newTenant()
	if _, err := f.pool.Exec(ctx, `INSERT INTO guardians (tenant_id, full_name, phone_primary) VALUES ($1, 'Other', $2)`, other, f.phone()); err != nil {
		t.Fatal(err)
	}
	var gid uuid.UUID
	if err := f.pool.QueryRow(ctx, `SELECT id FROM guardians WHERE tenant_id = $1`, other).Scan(&gid); err != nil {
		t.Fatal(err)
	}
	filter, _ := json.Marshal(map[string]any{"guardian_ids": []string{gid.String()}})
	req.AudienceType, req.AudienceFilter = "custom", filter
	third, err := f.svc.CreateAndSend(ctx, other, Actor{}, req)
	if err != nil {
		t.Fatalf("same key in another school: %v", err)
	}
	if third.ID == first.ID || third.TenantID != other {
		t.Fatalf("another school's send returned this school's message")
	}
}

func TestAnotherSchoolCannotSeeOrActOnAMessage(t *testing.T) {
	f := newFixture(t)
	f.guardian("Mary", f.phone())
	ctx := context.Background()
	later := time.Now().Add(time.Hour)
	msg, err := f.svc.CreateAndSend(ctx, f.tenantID, Actor{StaffID: &f.staffID}, CreateMessageRequest{
		Channel: "sms", AudienceType: "all_parents", Content: "Private", ScheduledAt: &later,
	})
	if err != nil {
		t.Fatal(err)
	}
	other := f.newTenant()

	if _, _, err := f.svc.GetMessage(ctx, other, msg.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetMessage from another school = %v, want ErrNotFound", err)
	}
	if _, err := f.svc.GetMessageLogs(ctx, other, msg.ID, "", 100, 0); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetMessageLogs from another school = %v, want ErrNotFound", err)
	}
	if err := f.svc.CancelScheduled(ctx, other, msg.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("CancelScheduled from another school = %v, want ErrNotFound", err)
	}
	if _, err := f.svc.ResendFailed(ctx, other, Actor{}, msg.ID, true, ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("ResendFailed from another school = %v, want ErrNotFound", err)
	}
	list, err := f.svc.ListMessages(ctx, other, "", "", 50, 0)
	if err != nil || len(list) != 0 {
		t.Errorf("ListMessages from another school = %d messages, err %v; want none", len(list), err)
	}
	if m, _ := f.message(msg.ID); m.Status != "scheduled" {
		t.Errorf("the message was changed from another school: status %s", m.Status)
	}
}

func TestPersonalisationFillsEachRecipientOrRefuses(t *testing.T) {
	f := newFixture(t)
	a, b := f.phone(), f.phone()
	f.guardian("Mary Wanjiku", a)
	f.guardian("John Otieno", b)
	ctx := context.Background()

	msg := f.send("Dear {{parent_name}}, {{learner_name}} ({{class}}) reports at 8am. {{school_name}}")
	f.sweep()

	if got := len(f.sender.calls); got != 2 {
		t.Fatalf("provider called %d times for two different texts, want 2", got)
	}
	texts := map[string]string{}
	for _, call := range f.sender.calls {
		for _, phone := range call.To {
			texts[phone] = call.Message
		}
	}
	want := "Dear Mary Wanjiku, Child of Mary Wanjiku (Grade 4 North) reports at 8am. Test School"
	if texts[a] != want {
		t.Errorf("Mary received %q, want %q", texts[a], want)
	}
	if !strings.HasPrefix(texts[b], "Dear John Otieno, ") {
		t.Errorf("John received %q", texts[b])
	}
	if m, _ := f.message(msg.ID); m.Status != "sent" {
		t.Errorf("status = %s, want sent", m.Status)
	}

	// An unknown variable is refused and nothing is recorded.
	before := countMessages(t, f)
	_, err := f.svc.CreateAndSend(ctx, f.tenantID, Actor{StaffID: &f.staffID}, CreateMessageRequest{
		Channel: "sms", AudienceType: "all_parents", Content: "Bus {{bus_number}}",
	})
	wantValidation(t, err, "{{bus_number}} is not something that can be filled in")

	// A contact has no learner: the send is refused, not sent with a gap.
	f.contact("Sponsor", f.phone())
	_, err = f.svc.CreateAndSend(ctx, f.tenantID, Actor{StaffID: &f.staffID}, CreateMessageRequest{
		Channel: "sms", AudienceType: "contacts", Content: "About {{learner_name}}",
	})
	wantValidation(t, err, "{{learner_name}} cannot be filled in for 1 of 1 recipients (for example Sponsor)")

	if after := countMessages(t, f); after != before {
		t.Errorf("a refused send recorded a message (%d -> %d)", before, after)
	}
	if got := len(f.sender.calls); got != 2 {
		t.Errorf("a refused send reached the provider (%d calls)", got)
	}
}

func countMessages(t *testing.T, f *fixture) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM messages WHERE tenant_id = $1`, f.tenantID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRefusals(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	send := func(req CreateMessageRequest) error {
		_, err := f.svc.CreateAndSend(ctx, f.tenantID, Actor{StaffID: &f.staffID}, req)
		return err
	}

	// Nobody in the audience.
	wantValidation(t, send(CreateMessageRequest{Channel: "sms", AudienceType: "all_parents", Content: "Hi"}), "nobody to send to")

	f.guardian("Mary", f.phone())
	wantValidation(t, send(CreateMessageRequest{Channel: "sms", AudienceType: "all_parents", Content: "   "}), "Write the message")
	wantValidation(t, send(CreateMessageRequest{Channel: "whatsapp", AudienceType: "all_parents", Content: "Hi"}), "Only SMS")
	wantValidation(t, send(CreateMessageRequest{Channel: "sms", AudienceType: "everyone", Content: "Hi"}), "Unknown audience")
	wantValidation(t, send(CreateMessageRequest{Channel: "sms", AudienceType: "all_parents", Content: strings.Repeat("a", 460)}), "4 SMS units")
	// 136 Unicode characters is three units by length but over the limit in billing terms.
	wantValidation(t, send(CreateMessageRequest{Channel: "sms", AudienceType: "all_parents", Content: strings.Repeat("’", 202)}), "4 SMS units")

	if n := countMessages(t, f); n != 0 {
		t.Errorf("refused sends recorded %d messages", n)
	}
}

func TestEstimateMatchesWhatASendWouldBill(t *testing.T) {
	f := newFixture(t)
	shared := f.phone()
	f.guardian("Mary", shared)
	f.guardian("Mary's partner", shared)
	f.guardian("John", f.phone())
	f.guardian("Bad number", "0200000")

	est, err := f.svc.EstimateReach(context.Background(), f.tenantID, CreateMessageRequest{
		AudienceType: "all_parents", Content: strings.Repeat("a", 161),
	})
	if err != nil {
		t.Fatal(err)
	}
	if est.RecipientCount != 2 || est.InvalidCount != 1 || est.SMSUnits != 2 {
		t.Fatalf("estimate = %+v, want 2 recipients, 1 invalid, 2 units", est)
	}
	if est.EstimatedKES != 3.2 {
		t.Errorf("estimated KES = %v, want 3.2 (2 recipients x 2 units x 0.80)", est.EstimatedKES)
	}
	if n := countMessages(t, f); n != 0 {
		t.Errorf("an estimate recorded %d messages", n)
	}
}

func TestNotConfiguredFailsTheMessageWithAReason(t *testing.T) {
	f := newFixture(t)
	f.guardian("Mary", f.phone())
	f.disp = NewDispatcher(f.pool, func(context.Context, uuid.UUID) (Sender, error) {
		return nil, &NotConfiguredError{Reason: "SMS is not set up for this school"}
	})
	msg := f.send("Hello")
	f.sweep()

	m, _ := f.message(msg.ID)
	if m.Status != "failed" {
		t.Fatalf("status = %s, want failed", m.Status)
	}
	for _, l := range f.logs(msg.ID) {
		if code(l) != "NOT_CONFIGURED" || l.ErrorMessage == nil || !strings.Contains(*l.ErrorMessage, "not set up") {
			t.Errorf("recipient = %s/%s, want NOT_CONFIGURED with the reason", l.Status, code(l))
		}
	}
}

func TestResendFailedSkipsUncertainRecipientsUnlessAsked(t *testing.T) {
	f := newFixture(t)
	ok, refused, uncertain := f.phone(), f.phone(), f.phone()
	f.guardian("Fine", ok)
	f.guardian("Refused", refused)
	f.guardian("Uncertain", uncertain)
	f.guardian("Bad number", "12345")
	f.sender.refuse[refused] = 407
	ctx := context.Background()

	msg := f.send("Hello")
	if _, err := f.pool.Exec(ctx, `UPDATE message_logs SET attempted_at = now() WHERE message_id = $1 AND phone = $2`, msg.ID, uncertain); err != nil {
		t.Fatal(err)
	}
	f.sweep()
	delete(f.sender.refuse, refused)

	resend, err := f.svc.ResendFailed(ctx, f.tenantID, Actor{StaffID: &f.staffID}, msg.ID, false, "")
	if err != nil {
		t.Fatalf("ResendFailed: %v", err)
	}
	logs := f.logs(resend.ID)
	if len(logs) != 1 || logs[refused].Status != "pending" {
		t.Fatalf("resend addressed %v, want only the definitely-failed recipient", logs)
	}
	if resend.AudienceType != "resend" || resend.Content != "Hello" {
		t.Errorf("resend = %s %q", resend.AudienceType, resend.Content)
	}

	withUncertain, err := f.svc.ResendFailed(ctx, f.tenantID, Actor{StaffID: &f.staffID}, msg.ID, true, "")
	if err != nil {
		t.Fatalf("ResendFailed (uncertain included): %v", err)
	}
	if got := len(f.logs(withUncertain.ID)); got != 2 {
		t.Fatalf("resend with uncertain recipients addressed %d, want 2", got)
	}

	// Nothing failed: nothing to resend.
	f.sweep()
	_, err = f.svc.ResendFailed(ctx, f.tenantID, Actor{StaffID: &f.staffID}, resend.ID, true, "")
	wantValidation(t, err, "no failed recipients")
}

func TestDeliveryReports(t *testing.T) {
	f := newFixture(t)
	a, b := f.phone(), f.phone()
	f.guardian("Mary", a)
	f.guardian("John", b)
	msg := f.send("Hello")
	f.sweep()
	logs := f.logs(msg.ID)
	idA, idB := *logs[a].ProviderMessageID, *logs[b].ProviderMessageID

	const token = "s3cret-token"
	dlr := sms.NewDLRHandler(f.pool, token)
	router := chi.NewRouter()
	dlr.Mount(router)
	post := func(path string, form url.Values) int {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}
	ok := "/webhooks/sms/dlr/" + token

	// A wrong token changes nothing and does not reveal the route.
	if got := post("/webhooks/sms/dlr/wrong", url.Values{"id": {idA}, "status": {"Success"}}); got != http.StatusNotFound {
		t.Fatalf("wrong token = %d, want 404", got)
	}
	if f.logs(msg.ID)[a].Status != "sent" {
		t.Fatal("a report with a wrong token was applied")
	}

	// A non-final report changes nothing.
	if got := post(ok, url.Values{"id": {idA}, "status": {"Buffered"}}); got != http.StatusOK {
		t.Fatalf("buffered report = %d, want 200", got)
	}
	if f.logs(msg.ID)[a].Status != "sent" {
		t.Fatal("a non-final report changed the status")
	}

	if got := post(ok, url.Values{"id": {idA}, "status": {"Success"}}); got != http.StatusOK {
		t.Fatalf("delivered report = %d, want 200", got)
	}
	if got := post(ok, url.Values{"id": {idB}, "status": {"Failed"}, "failureReason": {"UserInBlackList"}}); got != http.StatusOK {
		t.Fatalf("failed report = %d, want 200", got)
	}
	logs = f.logs(msg.ID)
	if logs[a].Status != "delivered" || logs[a].DeliveredAt == nil {
		t.Errorf("recipient A = %s, want delivered with a time", logs[a].Status)
	}
	if logs[b].Status != "failed" || code(logs[b]) != "DELIVERY_FAILED" || logs[b].ErrorMessage == nil || *logs[b].ErrorMessage != "UserInBlackList" {
		t.Errorf("recipient B = %s/%s, want failed/DELIVERY_FAILED with the reason", logs[b].Status, code(logs[b]))
	}
	m, stats := f.message(msg.ID)
	if m.DeliveredCount != 1 || m.FailedCount != 1 || stats.DeliveryRate != 50 {
		t.Errorf("counts = %d delivered, %d failed, rate %v; want 1, 1, 50", m.DeliveredCount, m.FailedCount, stats.DeliveryRate)
	}

	// Delivered is final: a late "Failed" does not undo it.
	post(ok, url.Values{"id": {idA}, "status": {"Failed"}})
	if f.logs(msg.ID)[a].Status != "delivered" {
		t.Error("a late failure report undid a delivery")
	}
	// A later success for a message reported failed does correct it.
	post(ok, url.Values{"id": {idB}, "status": {"Success"}})
	if l := f.logs(msg.ID)[b]; l.Status != "delivered" || l.ErrorCode != nil {
		t.Errorf("recipient B after a success report = %s/%s, want delivered with no error", l.Status, code(l))
	}

	// Unknown id: accepted (so it is not retried forever), nothing changes.
	if got := post(ok, url.Values{"id": {"ATXid_unknown"}, "status": {"Success"}}); got != http.StatusOK {
		t.Errorf("unknown id = %d, want 200", got)
	}
	if got := post(ok, url.Values{"status": {"Success"}}); got != http.StatusBadRequest {
		t.Errorf("missing id = %d, want 400", got)
	}

	// With no token configured the endpoint refuses everything.
	open := chi.NewRouter()
	sms.NewDLRHandler(f.pool, "").Mount(open)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/sms/dlr/anything", strings.NewReader("id="+idA+"&status=Failed"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	open.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("no token configured = %d, want 404", rec.Code)
	}
}

// --- sender lookup -----------------------------------------------------------

type fakeResolver struct {
	config      map[string]any
	secrets     map[string]string
	usePlatform bool
}

func (r fakeResolver) ResolveSecrets(context.Context, uuid.UUID, string) (map[string]any, map[string]string, bool, error) {
	return r.config, r.secrets, r.usePlatform, nil
}

func TestSenderFactory(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	platform := PlatformSMS{APIKey: "platform-key", Username: "platformuser", SenderID: "PLATFORM"}

	// Nothing configured anywhere, or only the .env.example placeholder.
	for _, p := range []PlatformSMS{{}, {APIKey: "your_at_api_key_here", Username: "sandbox"}} {
		_, err := NewSenderFactory(f.pool, fakeResolver{usePlatform: true}, p)(ctx, f.tenantID)
		var notConfigured *NotConfiguredError
		if !errors.As(err, &notConfigured) {
			t.Errorf("platform %+v: err = %v, want NotConfiguredError", p, err)
		}
	}

	// Inheriting the platform account.
	sender, err := NewSenderFactory(f.pool, fakeResolver{usePlatform: true}, platform)(ctx, f.tenantID)
	if err != nil {
		t.Fatal(err)
	}
	if got := sender.(*sms.ATClient).SenderID(); got != "PLATFORM" {
		t.Errorf("sender id = %q, want the platform's", got)
	}

	// The school's own sender id on the tenant row wins over the platform's.
	if _, err := f.pool.Exec(ctx, `UPDATE tenants SET at_sender_id = 'MYSCHOOL' WHERE id = $1`, f.tenantID); err != nil {
		t.Fatal(err)
	}
	sender, _ = NewSenderFactory(f.pool, fakeResolver{usePlatform: true}, platform)(ctx, f.tenantID)
	if got := sender.(*sms.ATClient).SenderID(); got != "MYSCHOOL" {
		t.Errorf("sender id = %q, want the school's", got)
	}

	// The school's own integration settings win over both.
	own := fakeResolver{
		config:  map[string]any{"username": "schooluser", "sender_id": "OWNID"},
		secrets: map[string]string{"api_key": "school-key"},
	}
	sender, err = NewSenderFactory(f.pool, own, PlatformSMS{})(ctx, f.tenantID)
	if err != nil {
		t.Fatalf("school with its own credentials and no platform account: %v", err)
	}
	if got := sender.(*sms.ATClient).SenderID(); got != "OWNID" {
		t.Errorf("sender id = %q, want the integration's", got)
	}
}

// {{fee_balance}} states what a parent owes from confirmed payments, and is
// refused for anyone who owes nothing rather than texting them "KES 0".
func TestFeeBalanceIsFilledInOnlyForParentsWhoOwe(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	owing := f.guardian("Mary Wanjiku", f.phone())
	f.guardian("John Otieno", f.phone())

	// Mary's child is billed 17,500 and 5,000 of it is paid.
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO invoices (tenant_id, learner_id, invoice_number, term, year, total_cents, paid_cents, status)
		SELECT tenant_id, id, 'INV-TEST-1', 1, 2026, 1750000, 500000, 'partially_paid'
		FROM learners WHERE tenant_id = $1 AND $2 = ANY(guardian_ids)`, f.tenantID, owing); err != nil {
		t.Fatalf("insert invoice: %v", err)
	}

	_, err := f.svc.CreateAndSend(ctx, f.tenantID, Actor{StaffID: &f.staffID}, CreateMessageRequest{
		Channel: "sms", AudienceType: "all_parents", Content: "Fees due: {{fee_balance}}",
	})
	wantValidation(t, err, "{{fee_balance}} cannot be filled in for 1 of 2 recipients")

	msg, err := f.svc.CreateAndSend(ctx, f.tenantID, Actor{StaffID: &f.staffID}, CreateMessageRequest{
		Channel: "sms", AudienceType: "fee_defaulters", Content: "Dear {{parent_name}}, fees due: {{fee_balance}}.",
	})
	if err != nil {
		t.Fatalf("CreateAndSend: %v", err)
	}
	f.sweep()
	f.sender.mu.Lock()
	defer f.sender.mu.Unlock()
	if len(f.sender.calls) != 1 || f.sender.calls[0].Message != "Dear Mary Wanjiku, fees due: KES 12,500." {
		t.Fatalf("sent %+v for message %s", f.sender.calls, msg.ID)
	}
}
