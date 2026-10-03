package platform

// Runs against a real Postgres with the migrations applied; skipped unless
// TEST_DATABASE_URL is set (see internal/comms/dispatch_test.go).

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/tenant"
)

type fakeAuth struct {
	created map[string]string // email -> password
	reset   map[string]string // auth id -> password
	exists  map[string]bool
	fail    error
}

func (f *fakeAuth) CreateUser(_ context.Context, email, password string) (string, error) {
	if f.fail != nil {
		return "", f.fail
	}
	if f.exists[email] {
		return "", errors.New(`create user error (status 422): {"error_code":"email_exists","msg":"A user with this email address has already been registered"}`)
	}
	f.created[email] = password
	return uuid.NewString(), nil
}

func (f *fakeAuth) SetUserPassword(_ context.Context, userID, password string) error {
	f.reset[userID] = password
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
	f := &fixture{
		t: t, pool: pool, tag: strings.ReplaceAll(uuid.NewString(), "-", "")[:10],
		auth: &fakeAuth{created: map[string]string{}, reset: map[string]string{}, exists: map[string]bool{}},
	}
	f.svc = NewService(pool, f.auth)
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM platform_users WHERE email LIKE '%' || $1 || '%'`, f.tag)
		_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE slug LIKE '%' || $1 || '%'`, f.tag)
		_, _ = pool.Exec(ctx, `DELETE FROM school_groups WHERE slug LIKE '%' || $1 || '%'`, f.tag)
		pool.Close()
	})
	return f
}

func (f *fixture) email(name string) string { return name + "-" + f.tag + "@example.test" }

func wantValidation(t *testing.T, err error, contains string) {
	t.Helper()
	var v *ValidationError
	if !errors.As(err, &v) || !strings.Contains(v.Message, contains) {
		t.Fatalf("want a validation error containing %q, got %v", contains, err)
	}
}

func TestGroupsAndSchools(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	group, err := f.svc.CreateGroup(ctx, "St. Mary's Trust "+f.tag, "")
	if err != nil {
		t.Fatal(err)
	}
	if group.Slug != "st-marys-trust-"+f.tag {
		t.Errorf("slug = %q, want it derived from the name", group.Slug)
	}
	_, err = f.svc.CreateGroup(ctx, "Another name", group.Slug)
	wantValidation(t, err, "already uses the short name")
	_, err = f.svc.CreateGroup(ctx, "  ", "")
	wantValidation(t, err, "Give the group a name")
	_, err = f.svc.CreateGroup(ctx, "Bad slug", "Has Spaces")
	wantValidation(t, err, "lowercase letters")

	school, err := f.svc.CreateSchool(ctx, "Hilltop Primary "+f.tag, "", &group.ID)
	if err != nil {
		t.Fatal(err)
	}
	if school.GroupName == nil || *school.GroupName != group.Name {
		t.Errorf("school group = %v, want %q", school.GroupName, group.Name)
	}
	missing := uuid.New()
	_, err = f.svc.CreateSchool(ctx, "Orphan "+f.tag, "", &missing)
	wantValidation(t, err, "group does not exist")

	// Moving a school out of its group.
	moved, err := f.svc.UpdateSchool(ctx, school.ID, "Hilltop Academy "+f.tag, nil)
	if err != nil {
		t.Fatal(err)
	}
	if moved.GroupID != nil || !strings.HasPrefix(moved.Name, "Hilltop Academy") {
		t.Errorf("after update: %+v", moved)
	}
	if _, err := f.svc.UpdateSchool(ctx, uuid.New(), "Nobody", nil); !errors.Is(err, ErrNotFound) {
		t.Errorf("updating a missing school = %v, want ErrNotFound", err)
	}

	got, err := f.svc.GetGroup(ctx, group.ID)
	if err != nil || got.SchoolCount != 0 {
		t.Errorf("group school count = %d (err %v), want 0 after the school left", got.SchoolCount, err)
	}
}

func TestCreateUser(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	group, err := f.svc.CreateGroup(ctx, "Trust "+f.tag, "")
	if err != nil {
		t.Fatal(err)
	}

	user, err := f.svc.CreateUser(ctx, UserInput{Email: "  " + strings.ToUpper(f.email("jane")) + " ", FullName: "Jane Doe", Scope: "group", GroupID: &group.ID})
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != f.email("jane") || user.Role != "principal" || user.Scope != "group" {
		t.Errorf("created %+v", user)
	}
	// The password is generated, handed to the sign-in provider, and returned once.
	if user.TempPassword == "" || f.auth.created[user.Email] != user.TempPassword {
		t.Errorf("temp password %q was not the one given to the sign-in provider", user.TempPassword)
	}
	if again, _ := f.svc.GetUser(ctx, user.ID); again.TempPassword != "" {
		t.Error("the password can be read back after creation")
	}

	// Refusals.
	_, err = f.svc.CreateUser(ctx, UserInput{Email: f.email("jane"), FullName: "Jane Again", Scope: "platform"})
	wantValidation(t, err, "already has a platform account")
	_, err = f.svc.CreateUser(ctx, UserInput{Email: "not-an-email", FullName: "X", Scope: "platform"})
	wantValidation(t, err, "valid email")
	_, err = f.svc.CreateUser(ctx, UserInput{Email: f.email("a"), FullName: "X", Scope: "group"})
	wantValidation(t, err, "Choose the group")
	_, err = f.svc.CreateUser(ctx, UserInput{Email: f.email("b"), FullName: "X", Scope: "platform", GroupID: &group.ID})
	wantValidation(t, err, "not tied to a group")
	_, err = f.svc.CreateUser(ctx, UserInput{Email: f.email("c"), FullName: "X", Scope: "galaxy"})
	wantValidation(t, err, "across the platform or in one group")
	_, err = f.svc.CreateUser(ctx, UserInput{Email: f.email("d"), FullName: "X", Scope: "platform", Role: "emperor"})
	wantValidation(t, err, "is not a role")

	// An email that is school staff would sign in as that staff member.
	school, err := f.svc.CreateSchool(ctx, "School "+f.tag, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO staff (tenant_id, full_name, email) VALUES ($1, 'Teacher', $2)`, school.ID, f.email("teacher")); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.CreateUser(ctx, UserInput{Email: f.email("teacher"), FullName: "Teacher", Scope: "platform"})
	wantValidation(t, err, "already a staff member of School")

	// The sign-in provider is down: no half-made user is left behind.
	f.auth.fail = errors.New("connection refused")
	if _, err := f.svc.CreateUser(ctx, UserInput{Email: f.email("e"), FullName: "E", Scope: "platform"}); err == nil {
		t.Fatal("expected an error when the sign-in account cannot be created")
	}
	f.auth.fail = nil
	var count int
	_ = f.pool.QueryRow(ctx, `SELECT COUNT(*) FROM platform_users WHERE email = $1`, f.email("e")).Scan(&count)
	if count != 0 {
		t.Error("a user row was created without a sign-in account")
	}

	// The email already has a sign-in account: it is kept, and the answer says so.
	f.auth.exists[f.email("known")] = true
	known, err := f.svc.CreateUser(ctx, UserInput{Email: f.email("known"), FullName: "Known", Scope: "platform"})
	if err != nil {
		t.Fatal(err)
	}
	if known.TempPassword != "" || !strings.Contains(known.Note, "existing password") {
		t.Errorf("existing account: password %q, note %q", known.TempPassword, known.Note)
	}
	// ... and its password cannot be reset from here.
	_, err = f.svc.ResetPassword(ctx, known.ID)
	wantValidation(t, err, "cannot be reset here")

	reset, err := f.svc.ResetPassword(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reset.TempPassword == "" || reset.TempPassword == user.TempPassword || len(f.auth.reset) != 1 {
		t.Errorf("reset password %q (was %q), provider calls %d", reset.TempPassword, user.TempPassword, len(f.auth.reset))
	}
}

func TestAPlatformAdministratorIsAlwaysLeft(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()

	// Whatever other tests or seed data left behind is set aside for this test.
	if _, err := f.pool.Exec(ctx, `UPDATE platform_users SET is_active = false WHERE scope = 'platform' AND is_active AND email NOT LIKE '%' || $1 || '%'`, f.tag); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `UPDATE platform_users SET is_active = true WHERE scope = 'platform' AND email NOT LIKE '%' || $1 || '%'`, f.tag)
	})

	group, _ := f.svc.CreateGroup(ctx, "Trust "+f.tag, "")
	first, err := f.svc.CreateUser(ctx, UserInput{Email: f.email("first"), FullName: "First", Scope: "platform"})
	if err != nil {
		t.Fatal(err)
	}
	other := uuid.New() // some other administrator asking

	_, err = f.svc.SetActive(ctx, other, first.ID, false)
	wantValidation(t, err, "only active platform administrator")
	_, err = f.svc.UpdateUser(ctx, other, first.ID, UserInput{FullName: "First", Scope: "group", GroupID: &group.ID})
	wantValidation(t, err, "only active platform administrator")

	second, err := f.svc.CreateUser(ctx, UserInput{Email: f.email("second"), FullName: "Second", Scope: "platform"})
	if err != nil {
		t.Fatal(err)
	}
	// Nobody removes their own access.
	_, err = f.svc.SetActive(ctx, first.ID, first.ID, false)
	wantValidation(t, err, "your own account")
	_, err = f.svc.UpdateUser(ctx, first.ID, first.ID, UserInput{FullName: "First", Scope: "group", GroupID: &group.ID})
	wantValidation(t, err, "your own platform access")

	// With a second administrator, the first can be deactivated by them.
	done, err := f.svc.SetActive(ctx, second.ID, first.ID, false)
	if err != nil || done.IsActive {
		t.Fatalf("deactivate: %+v, %v", done, err)
	}
	back, err := f.svc.SetActive(ctx, second.ID, first.ID, true)
	if err != nil || !back.IsActive {
		t.Fatalf("reactivate: %+v, %v", back, err)
	}
}

// A platform user inside a school acts as a staff row of their own, which is
// not staff of the school.
func TestOperatorStaffRow(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	schools := tenant.NewService(f.pool)
	school, err := f.svc.CreateSchool(ctx, "School "+f.tag, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	user, err := f.svc.CreateUser(ctx, UserInput{Email: f.email("ops"), FullName: "Jane Doe", Scope: "platform", Role: "bursar"})
	if err != nil {
		t.Fatal(err)
	}

	staffID, role, active, err := schools.OperatorStaff(ctx, school.ID, user.ID)
	if err != nil || !active || role != "bursar" {
		t.Fatalf("OperatorStaff = %s %q %v %v", staffID, role, active, err)
	}
	again, _, _, _ := schools.OperatorStaff(ctx, school.ID, user.ID)
	if again != staffID {
		t.Error("a second request created a second staff row")
	}

	var name, email string
	if err := f.pool.QueryRow(ctx, `SELECT full_name, email FROM staff WHERE id = $1`, staffID).Scan(&name, &email); err != nil {
		t.Fatal(err)
	}
	if name != "Jane Doe (Shule360 platform)" || !strings.HasSuffix(email, "@shule360.invalid") {
		t.Errorf("staff row = %q <%s>", name, email)
	}

	// A role change is picked up on the next request.
	if _, err := f.svc.UpdateUser(ctx, uuid.New(), user.ID, UserInput{FullName: "Jane Doe", Scope: "platform", Role: "principal"}); err != nil {
		// She is not the only administrator only if others exist; a role change keeps her platform scope, so this must succeed.
		t.Fatal(err)
	}
	if _, role, _, _ := schools.OperatorStaff(ctx, school.ID, user.ID); role != "principal" {
		t.Errorf("role after change = %q, want principal", role)
	}

	// Deactivated: refused, and nothing is created for an unknown user either.
	if _, err := f.pool.Exec(ctx, `UPDATE platform_users SET is_active = false WHERE id = $1`, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, active, err := schools.OperatorStaff(ctx, school.ID, user.ID); err != nil || active {
		t.Errorf("deactivated user: active=%v err=%v, want inactive", active, err)
	}
	if _, _, active, _ := schools.OperatorStaff(ctx, school.ID, uuid.New()); active {
		t.Error("an unknown user was given a staff row")
	}
}
