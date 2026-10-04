package academic

// These tests run against a real Postgres with the migrations applied: what
// they check — whose learner it is, what a published card refuses, what is
// recorded before a parent is texted — lives in the database.
//
// They are skipped unless TEST_DATABASE_URL is set. Each test creates its own
// school.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/academic/assessment"
	"github.com/shule360/api/internal/academic/attendance"
	"github.com/shule360/api/internal/academic/curriculum"
	"github.com/shule360/api/internal/comms"
	"github.com/shule360/api/internal/reports"
	"github.com/shule360/api/pkg/apperr"
)

type fixture struct {
	t         *testing.T
	pool      *pgxpool.Pool
	tenantID  uuid.UUID
	teacherID uuid.UUID
	subStrand uuid.UUID
	n         int
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
	f := &fixture{t: t, pool: pool}
	f.tenantID = f.school()
	f.teacherID = f.staff(f.tenantID)
	f.subStrand = f.subStrandIn(f.tenantID, "Mathematics", "Numbers", "Whole numbers")
	return f
}

func (f *fixture) exec(sql string, args ...any) {
	f.t.Helper()
	if _, err := f.pool.Exec(context.Background(), sql, args...); err != nil {
		f.t.Fatalf("exec: %v", err)
	}
}

func (f *fixture) id(sql string, args ...any) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&id); err != nil {
		f.t.Fatalf("insert: %v", err)
	}
	return id
}

func (f *fixture) school() uuid.UUID {
	id := f.id(`INSERT INTO tenants (name, slug) VALUES ('Test School', $1) RETURNING id`, "t-"+uuid.NewString())
	f.t.Cleanup(func() {
		if _, err := f.pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, id); err != nil {
			f.t.Errorf("remove test school: %v", err)
		}
	})
	return id
}

func (f *fixture) staff(tenantID uuid.UUID) uuid.UUID {
	return f.id(`INSERT INTO staff (tenant_id, full_name, email, role) VALUES ($1, 'Test Teacher', $2, 'teacher') RETURNING id`,
		tenantID, uuid.NewString()+"@example.test")
}

// learner adds a learner with one parent who can be texted.
func (f *fixture) learner(tenantID uuid.UUID) uuid.UUID {
	f.n++
	guardian := f.id(`INSERT INTO guardians (tenant_id, full_name, phone_primary) VALUES ($1, $2, $3) RETURNING id`,
		tenantID, fmt.Sprintf("Parent %d", f.n), fmt.Sprintf("+2547%08d", time.Now().UnixNano()%100000000))
	upi := strings.ToUpper("T" + strings.ReplaceAll(uuid.NewString(), "-", "")[:15])
	// No stream: a learner without one used to break every joined list.
	return f.id(`INSERT INTO learners (tenant_id, upi, full_name, grade, guardian_ids) VALUES ($1, $2, $3, 'Grade 4', ARRAY[$4::uuid]) RETURNING id`,
		tenantID, upi, fmt.Sprintf("Learner %d", f.n), guardian)
}

func (f *fixture) subStrandIn(tenantID uuid.UUID, area, strand, sub string) uuid.UUID {
	areaID := f.id(`INSERT INTO learning_areas (tenant_id, name, grade_level) VALUES ($1, $2, 'Grade 4') RETURNING id`, tenantID, area)
	strandID := f.id(`INSERT INTO strands (tenant_id, learning_area_id, name) VALUES ($1, $2, $3) RETURNING id`, tenantID, areaID, strand)
	return f.id(`INSERT INTO sub_strands (tenant_id, strand_id, name) VALUES ($1, $2, $3) RETURNING id`, tenantID, strandID, sub)
}

func (f *fixture) observe(learner uuid.UUID, level int) *assessment.Assessment {
	f.t.Helper()
	a, err := assessment.NewService(f.pool).Create(context.Background(), f.tenantID, assessment.CreateAssessmentRequest{
		LearnerID: learner, SubStrandID: f.subStrand, RubricLevel: level, TeacherID: f.teacherID, Term: 3, Year: 2026,
	})
	if err != nil {
		f.t.Fatalf("observe: %v", err)
	}
	return a
}

func refused(t *testing.T, err error, want string) {
	t.Helper()
	var v *apperr.ValidationError
	var c *apperr.ConflictError
	if !errors.As(err, &v) && !errors.As(err, &c) {
		t.Fatalf("want a refusal containing %q, got %v", want, err)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("refusal = %q, want it to contain %q", err.Error(), want)
	}
}

// --- Observations ---

// The request that the web sends must reach the fields it names. Without JSON
// tags learner_id was dropped and no observation could ever be recorded.
func TestObservationRequestReadsWhatTheWebSends(t *testing.T) {
	var req assessment.CreateAssessmentRequest
	body := `{"learner_id":"11111111-1111-1111-1111-111111111111","sub_strand_id":"22222222-2222-2222-2222-222222222222",
		"rubric_level":3,"note":"Counts to 100","term":3,"year":2026,"teacher_id":"33333333-3333-3333-3333-333333333333"}`
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	if req.LearnerID == uuid.Nil || req.SubStrandID == uuid.Nil || req.RubricLevel != 3 {
		t.Fatalf("request did not bind: %+v", req)
	}
	if req.TeacherID != uuid.Nil {
		t.Fatal("the teacher was read from the request; it must come from the session")
	}
}

func TestObservationIsRecordedAndListedWithItsLearner(t *testing.T) {
	f := newFixture(t)
	learner := f.learner(f.tenantID)
	f.observe(learner, 3)

	rows, err := assessment.NewService(f.pool).ListSummariesByTermYear(context.Background(), f.tenantID, 3, 2026)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 || rows[0].LearnerID != learner || rows[0].LearningArea != "Mathematics" || rows[0].TeacherID != f.teacherID {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestObservationRefusals(t *testing.T) {
	f := newFixture(t)
	svc := assessment.NewService(f.pool)
	learner := f.learner(f.tenantID)
	other := f.school()
	otherLearner := f.learner(other)
	otherSub := f.subStrandIn(other, "English", "Reading", "Fluency")

	ok := assessment.CreateAssessmentRequest{LearnerID: learner, SubStrandID: f.subStrand, RubricLevel: 2, TeacherID: f.teacherID, Term: 3, Year: 2026}
	cases := []struct {
		name   string
		change func(*assessment.CreateAssessmentRequest)
		want   string
	}{
		{"no learner", func(r *assessment.CreateAssessmentRequest) { r.LearnerID = uuid.Nil }, "Choose the learner"},
		{"level 5", func(r *assessment.CreateAssessmentRequest) { r.RubricLevel = 5 }, "rubric level"},
		{"term 4", func(r *assessment.CreateAssessmentRequest) { r.Term = 4 }, "Term must be"},
		{"nobody signed in", func(r *assessment.CreateAssessmentRequest) { r.TeacherID = uuid.Nil }, "Sign in"},
		{"another school's learner", func(r *assessment.CreateAssessmentRequest) { r.LearnerID = otherLearner }, "not in your school"},
		{"another school's sub-strand", func(r *assessment.CreateAssessmentRequest) { r.SubStrandID = otherSub }, "not in your school's curriculum"},
	}
	for _, c := range cases {
		req := ok
		c.change(&req)
		_, err := svc.Create(context.Background(), f.tenantID, req)
		if err == nil {
			t.Fatalf("%s: was accepted", c.name)
		}
		refused(t, err, c.want)
	}
}

func TestObservationIsRemovedOnlyByItsTeacherOrThePrincipal(t *testing.T) {
	f := newFixture(t)
	svc := assessment.NewService(f.pool)
	a := f.observe(f.learner(f.tenantID), 3)
	colleague := f.staff(f.tenantID)

	refused(t, svc.Delete(context.Background(), f.tenantID, a.ID, colleague, false), "Only the teacher")
	if err := svc.Delete(context.Background(), f.school(), a.ID, f.teacherID, true); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("another school deleting it: %v", err)
	}
	if err := svc.Delete(context.Background(), f.tenantID, a.ID, colleague, true); err != nil {
		t.Fatalf("principal: %v", err)
	}
}

// --- Attendance ---

func today() attendance.Date { return attendance.Date{Time: attendance.Today()} }

func TestRegisterRefusesADayThatHasNotHappened(t *testing.T) {
	f := newFixture(t)
	svc := attendance.NewService(f.pool)
	learner := f.learner(f.tenantID)
	tomorrow := attendance.Date{Time: attendance.Today().AddDate(0, 0, 1)}

	_, err := svc.MarkBulk(context.Background(), f.tenantID, attendance.BulkMarkRequest{
		Date: tomorrow, Marks: []attendance.CreateAttendanceRequest{{LearnerID: learner, Status: attendance.AttendancePresent}},
	})
	refused(t, err, "has not happened yet")
	_, err = svc.MarkAttendance(context.Background(), f.tenantID, attendance.CreateAttendanceRequest{
		LearnerID: learner, Date: tomorrow, Status: attendance.AttendancePresent,
	})
	refused(t, err, "has not happened yet")
}

func TestRegisterRefusesAnotherSchoolsLearnerAndSavesNothing(t *testing.T) {
	f := newFixture(t)
	svc := attendance.NewService(f.pool)
	mine := f.learner(f.tenantID)
	theirs := f.learner(f.school())

	_, err := svc.MarkBulk(context.Background(), f.tenantID, attendance.BulkMarkRequest{
		Date: today(), Marks: []attendance.CreateAttendanceRequest{
			{LearnerID: mine, Status: attendance.AttendancePresent},
			{LearnerID: theirs, Status: attendance.AttendanceAbsent},
		},
	})
	refused(t, err, "not in your school")
	rows, err := svc.ListByLearner(context.Background(), f.tenantID, mine)
	if err != nil || len(rows) != 0 {
		t.Fatalf("part of a refused register was saved: %v %v", rows, err)
	}
}

// A mark with no reason has a NULL reason; listing a learner's attendance
// failed on the first one.
func TestLearnerAttendanceListsMarksWithoutAReason(t *testing.T) {
	f := newFixture(t)
	svc := attendance.NewService(f.pool)
	learner := f.learner(f.tenantID)
	f.exec(`INSERT INTO attendance (tenant_id, learner_id, date, status) VALUES ($1, $2, CURRENT_DATE - 1, 'present')`, f.tenantID, learner)

	rows, err := svc.ListByLearner(context.Background(), f.tenantID, learner)
	if err != nil || len(rows) != 1 {
		t.Fatalf("ListByLearner: %v %v", rows, err)
	}
	if _, err := svc.ListSummariesByDate(context.Background(), f.tenantID, rows[0].Date); err != nil {
		t.Fatalf("ListSummariesByDate: %v", err)
	}
}

func (f *fixture) markToday(marks map[uuid.UUID]attendance.AttendanceStatus) {
	f.t.Helper()
	req := attendance.BulkMarkRequest{Date: today()}
	for id, status := range marks {
		req.Marks = append(req.Marks, attendance.CreateAttendanceRequest{LearnerID: id, Status: status, MarkedBy: f.teacherID})
	}
	if _, err := attendance.NewService(f.pool).MarkBulk(context.Background(), f.tenantID, req); err != nil {
		f.t.Fatalf("MarkBulk: %v", err)
	}
}

func TestAbsenceAlertIsARecordedMessageSentOncePerLearner(t *testing.T) {
	f := newFixture(t)
	absent, present := f.learner(f.tenantID), f.learner(f.tenantID)
	f.markToday(map[uuid.UUID]attendance.AttendanceStatus{absent: attendance.AttendanceAbsent, present: attendance.AttendancePresent})

	// No dispatcher: the message is recorded and waits, which is all this
	// checks. Nothing is sent to a provider from a test.
	alerts := attendance.NewAbsenceAlertService(f.pool, comms.NewCommsService(f.pool, nil))
	result, err := alerts.AlertAbsences(context.Background(), f.tenantID, f.teacherID, attendance.Today())
	if err != nil {
		t.Fatalf("AlertAbsences: %v", err)
	}
	if result.Sent != 1 || len(result.Skipped) != 0 {
		t.Fatalf("result = %+v", result)
	}

	var content string
	var recipients int
	var notified bool
	if err := f.pool.QueryRow(context.Background(), `
		SELECT m.content, (SELECT COUNT(*) FROM message_logs ml WHERE ml.message_id = m.id), a.sms_notified
		FROM attendance a JOIN messages m ON m.id = a.alert_message_id
		WHERE a.tenant_id = $1 AND a.learner_id = $2`, f.tenantID, absent).Scan(&content, &recipients, &notified); err != nil {
		t.Fatalf("the alert is not linked to a message: %v", err)
	}
	if !notified || recipients != 1 || !strings.Contains(content, "was marked absent today") || strings.Contains(content, "{{") {
		t.Fatalf("content=%q recipients=%d notified=%v", content, recipients, notified)
	}

	// Saving the register again changes nothing about who was told, and a
	// second pass texts nobody.
	f.markToday(map[uuid.UUID]attendance.AttendanceStatus{absent: attendance.AttendanceAbsent, present: attendance.AttendancePresent})
	again, err := alerts.AlertAbsences(context.Background(), f.tenantID, f.teacherID, attendance.Today())
	if err != nil || again.Sent != 0 {
		t.Fatalf("second pass: %+v %v", again, err)
	}
	var messages int
	_ = f.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM messages WHERE tenant_id = $1`, f.tenantID).Scan(&messages)
	if messages != 1 {
		t.Fatalf("%d messages, want 1", messages)
	}
}

func TestAbsenceAlertSaysWhoWasNotTextedAndWhy(t *testing.T) {
	f := newFixture(t)
	optedOut := f.learner(f.tenantID)
	f.exec(`UPDATE guardians SET is_sms_opted_out = true WHERE tenant_id = $1`, f.tenantID)
	noParent := f.id(`INSERT INTO learners (tenant_id, upi, full_name, grade) VALUES ($1, $2, 'No Parent', 'Grade 4') RETURNING id`,
		f.tenantID, "NP"+strings.ToUpper(uuid.NewString()[:8]))
	f.markToday(map[uuid.UUID]attendance.AttendanceStatus{optedOut: attendance.AttendanceAbsent, noParent: attendance.AttendanceAbsent})

	alerts := attendance.NewAbsenceAlertService(f.pool, comms.NewCommsService(f.pool, nil))
	result, err := alerts.AlertAbsences(context.Background(), f.tenantID, f.teacherID, attendance.Today())
	if err != nil {
		t.Fatalf("AlertAbsences: %v", err)
	}
	if result.Sent != 0 || len(result.Skipped) != 2 {
		t.Fatalf("result = %+v", result)
	}
	var notified int
	_ = f.pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM attendance WHERE tenant_id = $1 AND sms_notified`, f.tenantID).Scan(&notified)
	if notified != 0 {
		t.Fatalf("%d marks claim a parent was texted", notified)
	}
}

func TestAbsenceAlertRefusals(t *testing.T) {
	f := newFixture(t)
	alerts := attendance.NewAbsenceAlertService(f.pool, comms.NewCommsService(f.pool, nil))

	_, err := alerts.AlertAbsences(context.Background(), f.tenantID, f.teacherID, attendance.Today().AddDate(0, 0, -1))
	refused(t, err, "only texted about today's register")

	f.exec(`UPDATE tenants SET modules = ARRAY['academic'] WHERE id = $1`, f.tenantID)
	_, err = alerts.AlertAbsences(context.Background(), f.tenantID, f.teacherID, attendance.Today())
	refused(t, err, "does not have Communications")
}

// --- Report cards ---

func TestReportCardIsDraftedPublishedAndThenLeftAlone(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	svc := reports.NewService(f.pool)
	learner := f.learner(f.tenantID)
	f.observe(learner, 2)
	f.observe(learner, 4) // the later observation of the same sub-strand is the one that counts
	f.exec(`INSERT INTO attendance (tenant_id, learner_id, date, status) VALUES
		($1, $2, '2026-09-07', 'present'), ($1, $2, '2026-09-08', 'late'), ($1, $2, '2026-09-09', 'absent'), ($1, $2, '2026-09-10', 'present')`,
		f.tenantID, learner)

	card, err := svc.GenerateReportCard(ctx, f.tenantID, learner, 3, 2026, f.teacherID, reports.CardInput{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if card.Status != reports.StatusDraft || len(card.Items) != 1 || *card.Items[0].RubricLevel != 4 || *card.OverallRating != 4 {
		t.Fatalf("card = %+v", card)
	}
	if card.GeneratedBy == nil || *card.GeneratedBy != f.teacherID {
		t.Fatalf("generated_by = %v", card.GeneratedBy)
	}
	if rate := card.AttendanceSummary["attendance_rate"]; rate != 75.0 {
		t.Fatalf("attendance rate = %v, want 75 (late counts as attended)", rate)
	}

	// The update that used to be a SQL error.
	comments := map[string]string{"Class teacher": "A steady term."}
	if card, err = svc.UpdateReportCard(ctx, f.tenantID, card.ID, reports.CardInput{TeacherComments: comments}); err != nil {
		t.Fatalf("update: %v", err)
	}
	// Rebuilding a draft keeps what the teacher wrote.
	if card, err = svc.GenerateReportCard(ctx, f.tenantID, learner, 3, 2026, f.teacherID, reports.CardInput{}); err != nil {
		t.Fatalf("regenerate: %v", err)
	}
	if card.TeacherComments["Class teacher"] != "A steady term." {
		t.Fatalf("comments lost on regenerate: %v", card.TeacherComments)
	}

	if card, err = svc.PublishReportCard(ctx, f.tenantID, card.ID, f.teacherID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if card.Status != reports.StatusFinal || card.PublishedAt == nil || *card.PublishedBy != f.teacherID {
		t.Fatalf("published card = %+v", card)
	}

	_, err = svc.UpdateReportCard(ctx, f.tenantID, card.ID, reports.CardInput{TeacherComments: map[string]string{"Class teacher": "Changed"}})
	refused(t, err, "published")
	_, err = svc.GenerateReportCard(ctx, f.tenantID, learner, 3, 2026, f.teacherID, reports.CardInput{})
	refused(t, err, "is published")
	_, err = svc.PublishReportCard(ctx, f.tenantID, card.ID, f.teacherID)
	refused(t, err, "already published")
	refused(t, svc.DeleteReportCard(ctx, f.tenantID, card.ID), "cannot be deleted")

	if card, err = svc.ReopenReportCard(ctx, f.tenantID, card.ID); err != nil || card.Status != reports.StatusDraft || card.PublishedAt != nil {
		t.Fatalf("reopen: %+v %v", card, err)
	}
	_, err = svc.ReopenReportCard(ctx, f.tenantID, card.ID)
	refused(t, err, "not published")
	if err := svc.DeleteReportCard(ctx, f.tenantID, card.ID); err != nil {
		t.Fatalf("delete draft: %v", err)
	}
}

func TestReportCardRefusals(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	svc := reports.NewService(f.pool)
	learner := f.learner(f.tenantID)

	_, err := svc.GenerateReportCard(ctx, f.tenantID, learner, 3, 2026, f.teacherID, reports.CardInput{})
	refused(t, err, "No observations are recorded")
	_, err = svc.GenerateReportCard(ctx, f.tenantID, learner, 4, 2026, f.teacherID, reports.CardInput{})
	refused(t, err, "Term must be")
	_, err = svc.GenerateReportCard(ctx, f.tenantID, f.learner(f.school()), 3, 2026, f.teacherID, reports.CardInput{})
	refused(t, err, "not in your school")

	f.observe(learner, 3)
	five := 5
	_, err = svc.GenerateReportCard(ctx, f.tenantID, learner, 3, 2026, f.teacherID, reports.CardInput{OverallRating: &five})
	refused(t, err, "overall rating")

	card, err := svc.GenerateReportCard(ctx, f.tenantID, learner, 3, 2026, f.teacherID, reports.CardInput{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// Another school cannot read, publish, print or delete it.
	other := f.school()
	if _, err := svc.GetReportCard(ctx, other, card.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("get from another school: %v", err)
	}
	if _, err := svc.PublishReportCard(ctx, other, card.ID, f.teacherID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("publish from another school: %v", err)
	}
	if _, _, err := svc.ReportCardPDF(ctx, other, card.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("pdf from another school: %v", err)
	}
	if err := svc.DeleteReportCard(ctx, other, card.ID); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("delete from another school: %v", err)
	}
}

func TestReportCardPDFIsADocument(t *testing.T) {
	f := newFixture(t)
	svc := reports.NewService(f.pool)
	learner := f.learner(f.tenantID)
	f.exec(`UPDATE learners SET full_name = 'Wanjikũ O''Neil / Achieng' WHERE id = $1`, learner)
	f.observe(learner, 3)
	card, err := svc.GenerateReportCard(context.Background(), f.tenantID, learner, 3, 2026, f.teacherID, reports.CardInput{
		TeacherComments: map[string]string{"Class teacher": strings.Repeat("Works well with others. ", 30)},
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	pdf, name, err := svc.ReportCardPDF(context.Background(), f.tenantID, card.ID)
	if err != nil {
		t.Fatalf("pdf: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) || len(pdf) < 1500 {
		t.Fatalf("not a PDF: %d bytes, starts %q", len(pdf), pdf[:8])
	}
	if name != "report-card-wanjik-o-neil-achieng-term-3-2026.pdf" {
		t.Fatalf("file name = %q", name)
	}
}

func TestClassReportCardsSkipPublishedAndNameTheUnobserved(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	svc := reports.NewService(f.pool)
	observed, published, unobserved := f.learner(f.tenantID), f.learner(f.tenantID), f.learner(f.tenantID)
	f.observe(observed, 3)
	f.observe(published, 2)
	card, err := svc.GenerateReportCard(ctx, f.tenantID, published, 3, 2026, f.teacherID, reports.CardInput{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PublishReportCard(ctx, f.tenantID, card.ID, f.teacherID); err != nil {
		t.Fatal(err)
	}
	_ = unobserved

	result, err := svc.GenerateForClass(ctx, f.tenantID, "Grade 4", "", 3, 2026, f.teacherID)
	if err != nil {
		t.Fatalf("GenerateForClass: %v", err)
	}
	if result.Generated != 1 || result.SkippedPublished != 1 || len(result.NoObservations) != 1 {
		t.Fatalf("result = %+v", result)
	}
	_, err = svc.GenerateForClass(ctx, f.tenantID, "Grade 9", "", 3, 2026, f.teacherID)
	refused(t, err, "no learners in that class")
}

// --- Curriculum ---

func TestDefaultLearningAreasAreAddedOnceAndOnlyForAKnownGrade(t *testing.T) {
	f := newFixture(t)
	svc := curriculum.NewService(f.pool)

	// The fixture already has Mathematics for Grade 4; it is not duplicated.
	added, err := svc.AddDefaultLearningAreas(context.Background(), f.tenantID, "Grade 4")
	if err != nil || added != len(curriculum.DefaultLearningAreas("Grade 4"))-1 {
		t.Fatalf("added %d: %v", added, err)
	}
	if added, err = svc.AddDefaultLearningAreas(context.Background(), f.tenantID, "grade  4"); err != nil || added != 0 {
		t.Fatalf("second time added %d: %v", added, err)
	}
	_, err = svc.AddDefaultLearningAreas(context.Background(), f.tenantID, "Form 2")
	refused(t, err, "no standard list")

	// What was added has no description or code, and still lists.
	areas, err := svc.ListLearningAreas(context.Background(), f.tenantID)
	if err != nil || len(areas) != len(curriculum.DefaultLearningAreas("Grade 4")) {
		t.Fatalf("ListLearningAreas: %d %v", len(areas), err)
	}
}

func TestANewSchoolsCompetenciesAndValuesAreSeededOnceAndList(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	svc := curriculum.NewService(f.pool)
	for i := 0; i < 2; i++ {
		tx, err := f.pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := curriculum.SeedDefaults(ctx, tx, f.tenantID); err != nil {
			t.Fatalf("SeedDefaults: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	competencies, err := svc.ListCoreCompetencies(ctx, f.tenantID)
	if err != nil || len(competencies) != 7 {
		t.Fatalf("competencies: %d %v", len(competencies), err)
	}
	values, err := svc.ListValues(ctx, f.tenantID)
	if err != nil || len(values) != 8 {
		t.Fatalf("values: %d %v", len(values), err)
	}
}
