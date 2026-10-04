package onboarding

// Database tests; skipped unless TEST_DATABASE_URL is set.

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// fakeAuth stands in for Supabase Auth.
type fakeAuth struct {
	created   map[string]string // email -> password
	passwords map[string]string // auth id -> password
	err       error
}

func (f *fakeAuth) CreateUser(_ context.Context, email, password string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if _, exists := f.created[email]; exists {
		return "", errors.New("email_exists: a user with this email address has already been registered")
	}
	f.created[email] = password
	id := uuid.NewString()
	f.passwords[id] = password
	return id, nil
}

func (f *fakeAuth) SetUserPassword(_ context.Context, userID, password string) error {
	f.passwords[userID] = password
	return nil
}

type fixture struct {
	t    *testing.T
	pool *pgxpool.Pool
	auth *fakeAuth
	svc  *Service
	tag  string
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
	auth := &fakeAuth{created: map[string]string{}, passwords: map[string]string{}}
	return &fixture{t: t, pool: pool, auth: auth, svc: NewService(pool, auth), tag: strings.ReplaceAll(uuid.NewString(), "-", "")[:10]}
}

func (f *fixture) email(name string) string { return name + "-" + f.tag + "@example.test" }

func (f *fixture) registration() Registration {
	return Registration{
		SchoolName: "Hilltop Primary " + f.tag, County: "nakuru", Ownership: "private", Phone: "0712 345 678",
		AdminName: "Mary Wanjiku", AdminEmail: f.email("mary"), AdminPhone: "+254712345679",
		Password: "a-long-password", Term: 2,
	}
}

// register creates a school and removes it when the test ends.
func (f *fixture) register(in Registration) *Registered {
	f.t.Helper()
	out, err := f.svc.RegisterSchool(context.Background(), in)
	if err != nil {
		f.t.Fatalf("RegisterSchool: %v", err)
	}
	f.t.Cleanup(func() {
		if _, err := f.pool.Exec(context.Background(), `DELETE FROM tenants WHERE id = $1`, out.SchoolID); err != nil {
			f.t.Errorf("remove test school: %v", err)
		}
	})
	return out
}

func (f *fixture) principal(schoolID uuid.UUID) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	if err := f.pool.QueryRow(context.Background(),
		`SELECT id FROM staff WHERE tenant_id = $1 AND role = 'principal' ORDER BY created_at LIMIT 1`, schoolID).Scan(&id); err != nil {
		f.t.Fatalf("find principal: %v", err)
	}
	return id
}

func wantInvalid(t *testing.T, err error, field, contains string) {
	t.Helper()
	var v *ValidationError
	if !errors.As(err, &v) {
		t.Fatalf("want a validation error containing %q, got %v", contains, err)
	}
	if v.Field != field || !strings.Contains(v.Message, contains) {
		t.Fatalf("validation = field %q message %q; want field %q containing %q", v.Field, v.Message, field, contains)
	}
}

func TestRegisterSchoolCreatesASchoolSomeoneCanSignInTo(t *testing.T) {
	f := newFixture(t)
	in := f.registration()
	out := f.register(in)

	if out.TempPassword != "" {
		t.Errorf("a chosen password was echoed back")
	}
	if f.auth.created[in.AdminEmail] != "a-long-password" {
		t.Errorf("the sign-in account was not created with the chosen password")
	}

	var county, phone, slug, term, year, role string
	var active bool
	if err := f.pool.QueryRow(context.Background(), `
		SELECT t.county, t.phone, t.slug, ts.current_term, ts.current_academic_year, st.role::text, st.is_active
		FROM tenants t JOIN tenant_settings ts ON ts.tenant_id = t.id JOIN staff st ON st.tenant_id = t.id
		WHERE t.id = $1`, out.SchoolID).Scan(&county, &phone, &slug, &term, &year, &role, &active); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if county != "Nakuru" || phone != "+254712345678" || term != "Term 2" || role != "principal" || !active {
		t.Errorf("stored county=%q phone=%q term=%q role=%q active=%v", county, phone, term, role, active)
	}
	if !strings.HasPrefix(slug, "hilltop-primary-") {
		t.Errorf("slug = %q", slug)
	}

	// A second school of the same name gets its own short name.
	again := in
	again.AdminEmail = f.email("second")
	other := f.register(again)
	var otherSlug string
	_ = f.pool.QueryRow(context.Background(), `SELECT slug FROM tenants WHERE id = $1`, other.SchoolID).Scan(&otherSlug)
	if otherSlug == slug || !strings.HasPrefix(otherSlug, slug) {
		t.Errorf("second school's slug = %q, first = %q", otherSlug, slug)
	}
}

func TestRegisterSchoolRefusals(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	schools := func() (n int) {
		_ = f.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tenants WHERE name LIKE '%' || $1`, f.tag).Scan(&n)
		return n
	}

	for _, tc := range []struct {
		name, field, contains string
		change                func(*Registration)
	}{
		{"no name", "school_name", "full name", func(r *Registration) { r.SchoolName = " x " }},
		{"unknown county", "county", "Choose the county", func(r *Registration) { r.County = "Atlantis" }},
		{"no ownership", "ownership", "public or private", func(r *Registration) { r.Ownership = "" }},
		{"bad school phone", "school_phone", "Kenyan phone", func(r *Registration) { r.Phone = "12345" }},
		{"bad email", "admin_email", "valid email", func(r *Registration) { r.AdminEmail = "mary@" }},
		{"short password", "password", "at least 10", func(r *Registration) { r.Password = "short" }},
		{"bad term", "term", "1, 2 or 3", func(r *Registration) { r.Term = 4 }},
		{"old year", "year", "current academic year", func(r *Registration) { r.Year = 1999 }},
	} {
		in := f.registration()
		tc.change(&in)
		_, err := f.svc.RegisterSchool(ctx, in)
		if err == nil {
			t.Fatalf("%s: accepted", tc.name)
		}
		wantInvalid(t, err, tc.field, tc.contains)
	}
	if n := schools(); n != 0 {
		t.Fatalf("refused registrations left %d schools behind", n)
	}

	// The sign-in service is down: no school is left that nobody can open.
	f.auth.err = errors.New("connection refused")
	if _, err := f.svc.RegisterSchool(ctx, f.registration()); err == nil {
		t.Fatalf("registration succeeded without a sign-in account")
	}
	f.auth.err = nil
	if n := schools(); n != 0 {
		t.Fatalf("a failed account creation left %d schools behind", n)
	}

	// An email that already signs in somewhere cannot open a second school.
	first := f.register(f.registration())
	_, err := f.svc.RegisterSchool(ctx, f.registration())
	wantInvalid(t, err, "admin_email", "already signs in to "+first.SchoolName)
	if n := schools(); n != 1 {
		t.Fatalf("%d schools after a refused duplicate, want 1", n)
	}
}

func TestPlatformRegistrationGeneratesAPassword(t *testing.T) {
	f := newFixture(t)
	in := f.registration()
	in.Password = ""
	out := f.register(in)
	if len(out.TempPassword) < 16 || f.auth.created[in.AdminEmail] != out.TempPassword {
		t.Fatalf("temp password %q does not match the account's", out.TempPassword)
	}
}

func TestAddingAndManagingUsers(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	school := f.register(f.registration())
	principal := f.principal(school.SchoolID)

	bursar, err := f.svc.AddUser(ctx, school.SchoolID, UserInput{FullName: "Jane Bursar", Email: " " + strings.ToUpper(f.email("jane")), Phone: "0722000111", Role: "bursar"})
	if err != nil {
		t.Fatalf("AddUser: %v", err)
	}
	if bursar.Email != f.email("jane") || !bursar.IsActive || bursar.TempPassword == "" || f.auth.created[bursar.Email] != bursar.TempPassword {
		t.Fatalf("added user = %+v", bursar)
	}
	if got, _ := f.svc.GetUser(ctx, school.SchoolID, bursar.ID); got.TempPassword != "" {
		t.Errorf("the password is readable after creation")
	}

	// One email is one person, in any school.
	_, err = f.svc.AddUser(ctx, school.SchoolID, UserInput{FullName: "Jane Again", Email: f.email("jane"), Role: "teacher"})
	wantInvalid(t, err, "email", "already signs in to")
	_, err = f.svc.AddUser(ctx, school.SchoolID, UserInput{FullName: "Root User", Email: f.email("root"), Role: "super_admin"})
	wantInvalid(t, err, "role", "Choose what this person does")

	// The sign-in service fails: no half-made user remains.
	f.auth.err = errors.New("connection refused")
	if _, err := f.svc.AddUser(ctx, school.SchoolID, UserInput{FullName: "Tom Teacher", Email: f.email("tom"), Role: "teacher"}); err == nil {
		t.Fatalf("a user was added without a sign-in account")
	}
	f.auth.err = nil
	if users, _ := f.svc.ListUsers(ctx, school.SchoolID, ""); len(users) != 2 {
		t.Fatalf("%d users after a failed add, want 2", len(users))
	}

	// The school always keeps a principal, and nobody locks themselves out.
	_, err = f.svc.SetActive(ctx, school.SchoolID, principal, principal, false)
	wantInvalid(t, err, "", "your own account")
	_, err = f.svc.SetActive(ctx, school.SchoolID, bursar.ID, principal, false)
	wantInvalid(t, err, "role", "only principal")
	_, err = f.svc.UpdateUser(ctx, school.SchoolID, bursar.ID, principal, UserInput{FullName: "Mary Wanjiku", Role: "teacher"})
	wantInvalid(t, err, "role", "only principal")

	promoted, err := f.svc.UpdateUser(ctx, school.SchoolID, principal, bursar.ID, UserInput{FullName: "Jane Bursar", Role: "principal"})
	if err != nil || promoted.Role != "principal" {
		t.Fatalf("promote: %+v, %v", promoted, err)
	}
	if _, err := f.svc.SetActive(ctx, school.SchoolID, bursar.ID, principal, false); err != nil {
		t.Fatalf("deactivating one of two principals: %v", err)
	}

	// A new password replaces the old one and is shown once.
	reset, err := f.svc.ResetPassword(ctx, school.SchoolID, bursar.ID)
	if err != nil || reset.TempPassword == "" || reset.TempPassword == bursar.TempPassword {
		t.Fatalf("reset = %+v, %v", reset, err)
	}

	// Another school sees none of this.
	otherReg := f.registration()
	otherReg.AdminEmail = f.email("other")
	other := f.register(otherReg)
	if _, err := f.svc.GetUser(ctx, other.SchoolID, bursar.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetUser from another school = %v", err)
	}
	if _, err := f.svc.ResetPassword(ctx, other.SchoolID, bursar.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("ResetPassword from another school = %v", err)
	}
	if _, err := f.svc.SetActive(ctx, other.SchoolID, f.principal(other.SchoolID), bursar.ID, false); !errors.Is(err, ErrNotFound) {
		t.Errorf("SetActive from another school = %v", err)
	}
}

func TestStatusReflectsWhatIsRecorded(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	school := f.register(f.registration())

	status, err := f.svc.Status(ctx, school.SchoolID)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	done := map[string]bool{}
	for _, step := range status.Steps {
		done[step.ID] = step.Done
	}
	if done["users"] || done["learners"] || done["fees"] || done["curriculum"] {
		t.Fatalf("a brand new school has steps done: %+v", done)
	}

	if _, err := f.svc.AddUser(ctx, school.SchoolID, UserInput{FullName: "Tom Teacher", Email: f.email("tom"), Role: "teacher"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO learners (tenant_id, upi, full_name, grade) VALUES ($1, $2, 'A Learner', 'Grade 4')`,
		school.SchoolID, "T"+strings.ToUpper(f.tag)); err != nil {
		t.Fatal(err)
	}
	status, _ = f.svc.Status(ctx, school.SchoolID)
	for _, step := range status.Steps {
		done[step.ID] = step.Done
	}
	if !done["users"] || !done["learners"] || done["fees"] {
		t.Fatalf("after adding a user and a learner: %+v", done)
	}
	if _, err := f.svc.Status(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Errorf("Status of a missing school = %v", err)
	}
}

// A public school does not start with tuition on its list; a private one
// does. Either way the list is only a start.
func TestANewSchoolStartsWithFeeItemsForItsKind(t *testing.T) {
	f := newFixture(t)
	names := func(schoolID uuid.UUID) string {
		rows, err := f.pool.Query(context.Background(), `SELECT name FROM fee_categories WHERE tenant_id = $1 ORDER BY sort_order`, schoolID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var all []string
		for rows.Next() {
			var n string
			_ = rows.Scan(&n)
			all = append(all, n)
		}
		return strings.Join(all, ", ")
	}

	private := f.register(f.registration())
	if got := names(private.SchoolID); !strings.HasPrefix(got, "Tuition, ") {
		t.Errorf("private school starts with %q", got)
	}

	in := f.registration()
	in.Ownership, in.AdminEmail = "public", f.email("head")
	public := f.register(in)
	got := names(public.SchoolID)
	if strings.Contains(got, "Tuition") || !strings.Contains(got, "Lunch programme") {
		t.Errorf("public school starts with %q", got)
	}
	var ownership string
	_ = f.pool.QueryRow(context.Background(), `SELECT ownership FROM tenants WHERE id = $1`, public.SchoolID).Scan(&ownership)
	if ownership != "public" {
		t.Errorf("ownership = %q", ownership)
	}
}
