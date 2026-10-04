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
		pool.Close()
	})
	return f
}

func (f *fixture) email(name string) string { return name + "-" + f.tag + "@example.test" }

// group returns one of the two groups.
func (f *fixture) group(ownership string) *Group {
	f.t.Helper()
	groups, err := f.svc.ListGroups(context.Background())
	if err != nil {
		f.t.Fatal(err)
	}
	for i := range groups {
		if groups[i].Ownership == ownership {
			return &groups[i]
		}
	}
	f.t.Fatalf("there is no %s group", ownership)
	return nil
}

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

	// There are two groups, public first, and the database refuses a third.
	groups, err := f.svc.ListGroups(ctx)
	if err != nil || len(groups) != 2 || groups[0].Ownership != "public" || groups[1].Ownership != "private" {
		t.Fatalf("groups = %+v, %v", groups, err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO school_groups (name, slug, ownership) VALUES ('Third', $1, 'private')`, "third-"+f.tag); err == nil {
		t.Fatal("a third group was accepted")
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO school_groups (name, slug) VALUES ('Third', $1)`, "third-"+f.tag); err == nil {
		t.Fatal("a group that is neither public nor private was accepted")
	}
	public, private := f.group("public"), f.group("private")

	// A school is in the group for its kind, and nowhere else.
	school, err := f.svc.CreateSchool(ctx, "Hilltop Primary "+f.tag, "", "public")
	if err != nil {
		t.Fatal(err)
	}
	if school.Slug != "hilltop-primary-"+f.tag {
		t.Errorf("slug = %q, want it derived from the name", school.Slug)
	}
	if school.GroupID == nil || *school.GroupID != public.ID {
		t.Fatalf("a public school is in group %v, want the public group", school.GroupID)
	}
	_, err = f.svc.CreateSchool(ctx, "Another "+f.tag, school.Slug, "")
	wantValidation(t, err, "already uses the short name")
	_, err = f.svc.CreateSchool(ctx, "Odd "+f.tag, "", "charter")
	wantValidation(t, err, "public or private")
	// Writing the group directly does not put a school in the wrong one.
	if _, err := f.pool.Exec(ctx, `UPDATE tenants SET group_id = $2 WHERE id = $1`, school.ID, private.ID); err != nil {
		t.Fatal(err)
	}
	if again, _ := f.svc.getSchool(ctx, school.ID); *again.GroupID != public.ID {
		t.Fatal("a public school was moved into the private group by hand")
	}

	// Changing its kind moves it.
	moved, err := f.svc.SetOwnership(ctx, school.ID, "private")
	if err != nil || moved.GroupID == nil || *moved.GroupID != private.ID {
		t.Fatalf("after becoming private: %+v, %v", moved, err)
	}
	_, err = f.svc.SetOwnership(ctx, school.ID, "charter")
	wantValidation(t, err, "public or private")

	// Renaming a school leaves it where it is.
	moved, err = f.svc.UpdateSchool(ctx, school.ID, "Hilltop Academy "+f.tag)
	if err != nil || *moved.GroupID != private.ID || !strings.HasPrefix(moved.Name, "Hilltop Academy") {
		t.Fatalf("after rename: %+v, %v", moved, err)
	}
	if _, err := f.svc.UpdateSchool(ctx, uuid.New(), "Nobody"); !errors.Is(err, ErrNotFound) {
		t.Errorf("updating a missing school = %v, want ErrNotFound", err)
	}

	// A group can be renamed and described; its kind is not editable.
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `UPDATE school_groups SET name = $2, description = NULLIF($3, '') WHERE id = $1`, public.ID, public.Name, public.Description)
	})
	renamed, err := f.svc.UpdateGroup(ctx, public.ID, "Government schools", "Funded by the state.")
	if err != nil || renamed.Name != "Government schools" || renamed.Description != "Funded by the state." || renamed.Ownership != "public" {
		t.Fatalf("UpdateGroup = %+v, %v", renamed, err)
	}
	_, err = f.svc.UpdateGroup(ctx, public.ID, "  ", "")
	wantValidation(t, err, "Give the group a name")
	if _, err := f.svc.UpdateGroup(ctx, uuid.New(), "Nobody", ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("updating a missing group = %v, want ErrNotFound", err)
	}

	// A new school has every module; the platform can narrow it to exactly
	// the ones bought, and the answer is read back by the API's own check.
	if len(moved.Modules) != len(tenant.Modules) {
		t.Errorf("a new school has modules %v, want all of %v", moved.Modules, tenant.Modules)
	}
	narrowed, err := f.svc.SetModules(ctx, school.ID, []string{"communications", "communications"})
	if err != nil || len(narrowed.Modules) != 1 || narrowed.Modules[0] != "communications" {
		t.Fatalf("SetModules = %+v, %v", narrowed, err)
	}
	lookup := tenant.NewService(f.pool)
	if on, _ := lookup.ModuleEnabled(ctx, school.ID, "finance"); on {
		t.Errorf("finance is still enabled after it was taken away")
	}
	if on, _ := lookup.ModuleEnabled(ctx, school.ID, "communications"); !on {
		t.Errorf("communications was switched off although it was kept")
	}
	_, err = f.svc.SetModules(ctx, school.ID, []string{"payroll"})
	wantValidation(t, err, "payroll is not a module")
	if _, err := f.svc.SetModules(ctx, uuid.New(), nil); !errors.Is(err, ErrNotFound) {
		t.Errorf("setting modules on a missing school = %v, want ErrNotFound", err)
	}
}

func TestCreateUser(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	group := f.group("private")

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
	missing := uuid.New()
	_, err = f.svc.CreateUser(ctx, UserInput{Email: f.email("a2"), FullName: "X", Scope: "group", GroupID: &missing})
	wantValidation(t, err, "public schools or private schools")
	_, err = f.svc.CreateUser(ctx, UserInput{Email: f.email("b"), FullName: "X", Scope: "platform", GroupID: &group.ID})
	wantValidation(t, err, "not tied to a group")
	_, err = f.svc.CreateUser(ctx, UserInput{Email: f.email("c"), FullName: "X", Scope: "galaxy"})
	wantValidation(t, err, "across the platform or in one group")
	_, err = f.svc.CreateUser(ctx, UserInput{Email: f.email("d"), FullName: "X", Scope: "platform", Role: "emperor"})
	wantValidation(t, err, "is not a role")

	// An email that is school staff would sign in as that staff member.
	school, err := f.svc.CreateSchool(ctx, "School "+f.tag, "", "")
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

	group := f.group("private")
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
	school, err := f.svc.CreateSchool(ctx, "School "+f.tag, "", "")
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
