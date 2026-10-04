// Package onboarding gets a school from nothing to working: the school and
// its first administrator are created together, the administrator adds the
// people who will use it, and a checklist says what is still to be set up.
package onboarding

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/comms/sms"
	"github.com/shule360/api/internal/finance"
)

// AuthProvider is the part of Supabase Auth this package uses.
type AuthProvider interface {
	CreateUser(ctx context.Context, email, password string) (string, error)
	SetUserPassword(ctx context.Context, userID, password string) error
}

// Service implements school registration and a school's own user management.
type Service struct {
	pool *pgxpool.Pool
	auth AuthProvider
}

func NewService(pool *pgxpool.Pool, auth AuthProvider) *Service {
	return &Service{pool: pool, auth: auth}
}

// ErrNotFound: the record does not exist in this school.
var ErrNotFound = errors.New("not found")

// ValidationError is a request the caller can correct. Field names the form
// field it is about, when there is one.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(field, format string, args ...any) error {
	return &ValidationError{Field: field, Message: fmt.Sprintf(format, args...)}
}

// Counties are Kenya's 47, as a school's address records them.
var Counties = []string{
	"Baringo", "Bomet", "Bungoma", "Busia", "Elgeyo-Marakwet", "Embu", "Garissa", "Homa Bay", "Isiolo", "Kajiado",
	"Kakamega", "Kericho", "Kiambu", "Kilifi", "Kirinyaga", "Kisii", "Kisumu", "Kitui", "Kwale", "Laikipia",
	"Lamu", "Machakos", "Makueni", "Mandera", "Marsabit", "Meru", "Migori", "Mombasa", "Murang'a", "Nairobi",
	"Nakuru", "Nandi", "Narok", "Nyamira", "Nyandarua", "Nyeri", "Samburu", "Siaya", "Taita-Taveta", "Tana River",
	"Tharaka-Nithi", "Trans Nzoia", "Turkana", "Uasin Gishu", "Vihiga", "Wajir", "West Pokot",
}

func knownCounty(name string) (string, bool) {
	for _, c := range Counties {
		if strings.EqualFold(c, strings.TrimSpace(name)) {
			return c, true
		}
	}
	return "", false
}

// Roles a school's administrator can give. super_admin is not among them: it
// is not handed out from inside a school.
var Roles = map[string]string{
	"principal":         "Principal — everything in the school, including users",
	"bursar":            "Bursar — fees, payments and procurement",
	"teacher":           "Teacher — learners, attendance and assessments",
	"hr":                "HR — staff records, leave and payroll",
	"transport_manager": "Transport manager — vehicles, routes and trips",
}

func cleanEmail(field, raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 || strings.ContainsAny(email, " \t\n") || !strings.Contains(email[at:], ".") || len(email) > 255 {
		return "", invalid(field, "Enter a valid email address.")
	}
	return email, nil
}

// cleanPhone accepts a Kenyan mobile number in any usual form and stores
// +254…; empty stays empty.
func cleanPhone(field, raw string) (*string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	phone, err := sms.NormalizeKenyanPhone(raw)
	if err != nil {
		return nil, invalid(field, "Enter a Kenyan phone number, for example 0712 345 678.")
	}
	return &phone, nil
}

var slugUnsafe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	slug := strings.Trim(slugUnsafe.ReplaceAllString(strings.ToLower(strings.ReplaceAll(name, "'", "")), "-"), "-")
	if len(slug) > 60 {
		slug = strings.Trim(slug[:60], "-")
	}
	if slug == "" {
		slug = "school"
	}
	return slug
}

func newPassword() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}
	for i, b := range buf {
		buf[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(buf[:4]) + "-" + string(buf[4:8]) + "-" + string(buf[8:12]) + "-" + string(buf[12:]), nil
}

func alreadyRegistered(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "already") || strings.Contains(text, "email_exists")
}

func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// emailTaken reports who already signs in with an email. Sign-in finds a
// person by email alone, so one email is one person across every school.
func (s *Service) emailTaken(ctx context.Context, q interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}, email string) (string, error) {
	var where string
	err := q.QueryRow(ctx, `
		SELECT t.name FROM staff st JOIN tenants t ON t.id = st.tenant_id WHERE lower(st.email) = $1
		UNION ALL
		SELECT 'the Shule360 platform' FROM platform_users WHERE lower(email) = $1
		LIMIT 1`, email).Scan(&where)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return where, err
}

// --- Registering a school ---------------------------------------------------

// Registration is everything needed to create a school someone can sign in to.
type Registration struct {
	SchoolName string `json:"school_name"`
	County     string `json:"county"`
	// Ownership: "public" (government) or "private".
	Ownership string `json:"ownership"`
	Phone     string `json:"school_phone"`
	Email     string `json:"school_email"`
	Address   string `json:"address"`
	Term      int    `json:"term"`
	Year      int    `json:"year"`

	AdminName  string `json:"admin_name"`
	AdminEmail string `json:"admin_email"`
	AdminPhone string `json:"admin_phone"`
	// Password is chosen by a person registering their own school. Left empty
	// (a platform administrator creating a school for someone), one is
	// generated and returned once.
	Password string `json:"password"`

	GroupID *uuid.UUID `json:"group_id,omitempty"`
}

// Registered is the outcome of a registration.
type Registered struct {
	SchoolID     uuid.UUID `json:"school_id"`
	SchoolName   string    `json:"school_name"`
	AdminEmail   string    `json:"admin_email"`
	TempPassword string    `json:"temp_password,omitempty"`
}

// RegisterSchool creates the school, its settings and its first
// administrator, then that person's sign-in account. If the account cannot be
// created the school is removed again: a school nobody can sign in to is
// worse than none.
func (s *Service) RegisterSchool(ctx context.Context, in Registration) (*Registered, error) {
	in.SchoolName = strings.Join(strings.Fields(in.SchoolName), " ")
	if len(in.SchoolName) < 3 {
		return nil, invalid("school_name", "Enter the school's full name.")
	}
	if len(in.SchoolName) > 200 {
		return nil, invalid("school_name", "That name is too long.")
	}
	county, ok := knownCounty(in.County)
	if !ok {
		return nil, invalid("county", "Choose the county the school is in.")
	}
	in.Ownership = strings.ToLower(strings.TrimSpace(in.Ownership))
	if in.Ownership != "public" && in.Ownership != "private" {
		return nil, invalid("ownership", "Say whether the school is public or private.")
	}
	schoolPhone, err := cleanPhone("school_phone", in.Phone)
	if err != nil {
		return nil, err
	}
	var schoolEmail *string
	if strings.TrimSpace(in.Email) != "" {
		e, err := cleanEmail("school_email", in.Email)
		if err != nil {
			return nil, err
		}
		schoolEmail = &e
	}
	now := time.Now()
	if in.Year == 0 {
		in.Year = now.Year()
	}
	if in.Term == 0 {
		in.Term = 1
	}
	if in.Term < 1 || in.Term > 3 {
		return nil, invalid("term", "Term must be 1, 2 or 3.")
	}
	if in.Year < now.Year()-1 || in.Year > now.Year()+1 {
		return nil, invalid("year", "Enter the current academic year.")
	}

	in.AdminName = strings.Join(strings.Fields(in.AdminName), " ")
	if len(in.AdminName) < 3 {
		return nil, invalid("admin_name", "Enter your full name.")
	}
	adminEmail, err := cleanEmail("admin_email", in.AdminEmail)
	if err != nil {
		return nil, err
	}
	adminPhone, err := cleanPhone("admin_phone", in.AdminPhone)
	if err != nil {
		return nil, err
	}
	password, generated := in.Password, false
	if password == "" {
		if password, err = newPassword(); err != nil {
			return nil, err
		}
		generated = true
	} else if len(password) < 10 {
		return nil, invalid("password", "Choose a password of at least 10 characters.")
	} else if len(password) > 72 {
		return nil, invalid("password", "Choose a password of at most 72 characters.")
	}

	if where, err := s.emailTaken(ctx, s.pool, adminEmail); err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	} else if where != "" {
		return nil, invalid("admin_email", "%s already signs in to %s. Sign in with it, or use a different email.", adminEmail, where)
	}

	// 1. The school, its settings and its administrator: one transaction.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if in.GroupID != nil {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM school_groups WHERE id = $1)`, *in.GroupID).Scan(&exists); err != nil {
			return nil, err
		}
		if !exists {
			return nil, invalid("group_id", "That group does not exist.")
		}
	}

	var schoolID uuid.UUID
	base := slugify(in.SchoolName)
	for attempt := 0; ; attempt++ {
		slug := base
		if attempt > 0 {
			slug = fmt.Sprintf("%s-%d", base, attempt+1)
		}
		// A savepoint, so a taken short name does not abort the transaction.
		err = pgx.BeginFunc(ctx, tx, func(sp pgx.Tx) error {
			return sp.QueryRow(ctx, `
				INSERT INTO tenants (name, slug, county, phone, email, address, group_id, ownership)
				VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8) RETURNING id`,
				in.SchoolName, slug, county, schoolPhone, schoolEmail, strings.TrimSpace(in.Address), in.GroupID, in.Ownership).Scan(&schoolID)
		})
		if err == nil {
			break
		}
		if !uniqueViolation(err) || attempt >= 20 {
			return nil, fmt.Errorf("insert school: %w", err)
		}
	}

	var staffID uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO staff (tenant_id, full_name, email, phone, role)
		VALUES ($1, $2, $3, $4, 'principal') RETURNING id`,
		schoolID, in.AdminName, adminEmail, adminPhone).Scan(&staffID); err != nil {
		return nil, fmt.Errorf("insert administrator: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO tenant_settings (tenant_id, current_term, current_academic_year, updated_by)
		VALUES ($1, $2, $3, $4)`,
		schoolID, fmt.Sprintf("Term %d", in.Term), fmt.Sprintf("%d", in.Year), staffID); err != nil {
		return nil, fmt.Errorf("insert settings: %w", err)
	}
	// What it charges for starts from the usual list for its kind of school;
	// the school changes it freely afterwards.
	if err := finance.SeedFeeCategories(ctx, tx, schoolID, in.Ownership); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// 2. The sign-in account, outside the transaction.
	undo := func() {
		if _, err := s.pool.Exec(context.WithoutCancel(ctx), `DELETE FROM tenants WHERE id = $1`, schoolID); err != nil {
			// Left behind: a school with an administrator who cannot sign in.
			// "Reset password" on that user from the platform repairs it.
			slog.Error("onboarding: could not remove a school after a failed registration", "school_id", schoolID, "error", err)
		}
	}
	authID, err := s.auth.CreateUser(ctx, adminEmail, password)
	if err != nil {
		undo()
		if alreadyRegistered(err) {
			return nil, invalid("admin_email", "%s already has a sign-in account. Sign in with it, or use a different email.", adminEmail)
		}
		return nil, fmt.Errorf("create sign-in account: %w", err)
	}
	if _, err := s.pool.Exec(context.WithoutCancel(ctx),
		`UPDATE staff SET supabase_user_id = $2::uuid WHERE id = $1`, staffID, authID); err != nil {
		// Sign-in finds the person by email, so this is not fatal.
		slog.Warn("onboarding: could not store the sign-in id", "staff_id", staffID, "error", err)
	}

	out := &Registered{SchoolID: schoolID, SchoolName: in.SchoolName, AdminEmail: adminEmail}
	if generated {
		out.TempPassword = password
	}
	return out, nil
}

// --- Setup checklist --------------------------------------------------------

// Step is one thing a school sets up, and whether it has.
type Step struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Done   bool   `json:"done"`
	// Where in the web app it is done.
	Href string `json:"href"`
	// Optional steps do not hold the school back.
	Optional bool `json:"optional,omitempty"`
}

// Status is where a school is in setting itself up.
type Status struct {
	SchoolName string `json:"school_name"`
	Steps      []Step `json:"steps"`
	Done       int    `json:"done"`
	Total      int    `json:"total"`
}

// Status works the checklist out from what is actually recorded, so it can
// never claim a step the school has not done.
func (s *Service) Status(ctx context.Context, tenantID uuid.UUID) (*Status, error) {
	var (
		name                                     string
		profile, mpesa, ownSMS                   bool
		users, learners, guardians               int
		areas, structures, invoices, smsMessages int
	)
	err := s.pool.QueryRow(ctx, `
		SELECT t.name,
		       t.phone IS NOT NULL AND t.county IS NOT NULL AND t.address IS NOT NULL,
		       (SELECT COUNT(*) FROM staff WHERE tenant_id = t.id AND is_active AND platform_user_id IS NULL),
		       (SELECT COUNT(*) FROM learners WHERE tenant_id = t.id AND is_active),
		       (SELECT COUNT(*) FROM guardians WHERE tenant_id = t.id),
		       (SELECT COUNT(*) FROM learning_areas WHERE tenant_id = t.id),
		       (SELECT COUNT(*) FROM fee_structures WHERE tenant_id = t.id),
		       (SELECT COUNT(*) FROM invoices WHERE tenant_id = t.id),
		       (SELECT COUNT(*) FROM messages WHERE tenant_id = t.id),
		       EXISTS (SELECT 1 FROM tenant_integrations WHERE tenant_id = t.id AND provider = 'mpesa' AND is_enabled AND NOT use_platform_default AND secrets_encrypted IS NOT NULL),
		       EXISTS (SELECT 1 FROM tenant_integrations WHERE tenant_id = t.id AND provider = 'africastalking' AND is_enabled AND NOT use_platform_default AND secrets_encrypted IS NOT NULL)
		FROM tenants t WHERE t.id = $1`, tenantID).Scan(
		&name, &profile, &users, &learners, &guardians, &areas, &structures, &invoices, &smsMessages, &mpesa, &ownSMS)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	plural := func(n int, one, many string) string {
		if n == 1 {
			return fmt.Sprintf("1 %s", one)
		}
		return fmt.Sprintf("%d %s", n, many)
	}
	steps := []Step{
		{ID: "profile", Title: "Complete the school's details", Done: profile, Href: "/settings",
			Detail: "Phone, address and county appear on receipts and report cards."},
		{ID: "users", Title: "Add the people who will use Shule360", Done: users > 1, Href: "/school/users",
			Detail: plural(users, "person can", "people can") + " sign in. Add the bursar and the teachers."},
		{ID: "learners", Title: "Add learners and their parents", Done: learners > 0, Href: "/learners/import",
			Detail: plural(learners, "learner", "learners") + " and " + plural(guardians, "parent", "parents") + " recorded. Import the class lists from a spreadsheet."},
		{ID: "curriculum", Title: "Set up the learning areas", Done: areas > 0, Href: "/academic/curriculum",
			Detail: plural(areas, "learning area", "learning areas") + " set up. Assessments and report cards are recorded against them."},
		{ID: "fees", Title: "Set the fees and bill the learners", Done: structures > 0 && invoices > 0, Href: "/finance/fees",
			Detail: plural(structures, "fee structure", "fee structures") + ", " + plural(invoices, "invoice", "invoices") + ". Create a fee structure per grade, then bill the grade."},
		{ID: "mpesa", Title: "Connect the school's M-Pesa paybill", Done: mpesa, Href: "/settings", Optional: true,
			Detail: "So that fees paid by M-Pesa reach the school's own account and are matched to learners."},
		{ID: "sms", Title: "Send the first SMS to parents", Done: smsMessages > 0, Href: "/communications/messages", Optional: !ownSMS && smsMessages == 0,
			Detail: plural(smsMessages, "message", "messages") + " sent so far."},
	}
	out := &Status{SchoolName: name, Steps: steps, Total: len(steps)}
	for _, step := range steps {
		if step.Done {
			out.Done++
		}
	}
	return out, nil
}

// --- A school's users -------------------------------------------------------

// User is a person who can sign in to a school.
type User struct {
	ID        uuid.UUID `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	Phone     *string   `json:"phone,omitempty"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	// Set once, on the response that created or reset the account.
	TempPassword string `json:"temp_password,omitempty"`
	Note         string `json:"note,omitempty"`
}

// UserInput is what an administrator enters for a user.
type UserInput struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
}

// The staff rows that stand for platform and group users inside a school are
// not the school's to manage, so they are left out everywhere here.
const userColumns = `id, full_name, email, phone, role::text, is_active, created_at`
const ownStaff = ` FROM staff WHERE tenant_id = $1 AND platform_user_id IS NULL `

func scanUser(row pgx.Row) (*User, error) {
	var u User
	if err := row.Scan(&u.ID, &u.FullName, &u.Email, &u.Phone, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// ListUsers returns the school's users, active first.
func (s *Service) ListUsers(ctx context.Context, tenantID uuid.UUID, search string) ([]User, error) {
	query := `SELECT ` + userColumns + ownStaff
	args := []any{tenantID}
	if search = strings.TrimSpace(search); search != "" {
		args = append(args, "%"+search+"%")
		query += ` AND (full_name ILIKE $2 OR email ILIKE $2)`
	}
	rows, err := s.pool.Query(ctx, query+` ORDER BY is_active DESC, full_name`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

// GetUser returns one of the school's users.
func (s *Service) GetUser(ctx context.Context, tenantID, id uuid.UUID) (*User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userColumns+ownStaff+` AND id = $2`, tenantID, id))
}

func (in *UserInput) check(needEmail bool) (email string, phone *string, err error) {
	in.FullName = strings.Join(strings.Fields(in.FullName), " ")
	if len(in.FullName) < 3 {
		return "", nil, invalid("full_name", "Enter the person's full name.")
	}
	if _, ok := Roles[in.Role]; !ok {
		return "", nil, invalid("role", "Choose what this person does in the school.")
	}
	if phone, err = cleanPhone("phone", in.Phone); err != nil {
		return "", nil, err
	}
	if needEmail {
		if email, err = cleanEmail("email", in.Email); err != nil {
			return "", nil, err
		}
	}
	return email, phone, nil
}

// AddUser creates a person who can sign in to the school. Their password is
// generated and returned once, for the administrator to hand over.
func (s *Service) AddUser(ctx context.Context, tenantID uuid.UUID, in UserInput) (*User, error) {
	email, phone, err := in.check(true)
	if err != nil {
		return nil, err
	}
	if where, err := s.emailTaken(ctx, s.pool, email); err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	} else if where != "" {
		return nil, invalid("email", "%s already signs in to %s. One email is one person; use a different one.", email, where)
	}

	// The row first: an account without a row is refused at sign-in, so a
	// failure after this point leaves nothing that works by accident.
	var id uuid.UUID
	err = s.pool.QueryRow(ctx, `
		INSERT INTO staff (tenant_id, full_name, email, phone, role, is_active)
		VALUES ($1, $2, $3, $4, $5::staff_role, false) RETURNING id`,
		tenantID, in.FullName, email, phone, in.Role).Scan(&id)
	if uniqueViolation(err) {
		return nil, invalid("email", "%s is already a user of this school.", email)
	}
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	password, err := newPassword()
	if err != nil {
		return nil, err
	}
	note := ""
	var authID *string
	if created, err := s.auth.CreateUser(ctx, email, password); err == nil {
		authID = &created
	} else if alreadyRegistered(err) {
		password = ""
		note = "This email already had a sign-in account, so its existing password still applies. Use \"Reset password\" if they do not know it."
	} else {
		if _, delErr := s.pool.Exec(context.WithoutCancel(ctx), `DELETE FROM staff WHERE id = $1`, id); delErr != nil {
			slog.Error("onboarding: could not remove a user after a failed account creation", "staff_id", id, "error", delErr)
		}
		return nil, fmt.Errorf("create sign-in account: %w", err)
	}
	if _, err := s.pool.Exec(context.WithoutCancel(ctx),
		`UPDATE staff SET is_active = true, supabase_user_id = $2::uuid WHERE id = $1`, id, authID); err != nil {
		return nil, fmt.Errorf("activate user: %w", err)
	}

	user, err := s.GetUser(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	user.TempPassword, user.Note = password, note
	return user, nil
}

// keepOnePrincipal refuses a change that would leave the school with nobody
// who can manage its users.
func (s *Service) keepOnePrincipal(ctx context.Context, tenantID, leaving uuid.UUID) error {
	var others int
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM staff
		WHERE tenant_id = $1 AND platform_user_id IS NULL AND is_active
		  AND role IN ('principal', 'super_admin') AND id <> $2`, tenantID, leaving).Scan(&others); err != nil {
		return err
	}
	if others == 0 {
		return invalid("role", "This is the school's only principal. Make someone else a principal first, so the school is never left without one.")
	}
	return nil
}

// UpdateUser changes a user's name, phone and role. The email is the
// sign-in, so it is not changed here.
func (s *Service) UpdateUser(ctx context.Context, tenantID, actorID, id uuid.UUID, in UserInput) (*User, error) {
	_, phone, err := in.check(false)
	if err != nil {
		return nil, err
	}
	current, err := s.GetUser(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if current.Role == "super_admin" {
		return nil, invalid("role", "This account is managed by Shule360, not from inside the school.")
	}
	if current.Role == "principal" && in.Role != "principal" {
		if id == actorID {
			return nil, invalid("role", "You cannot take the principal role away from yourself. Ask another principal to do it.")
		}
		if err := s.keepOnePrincipal(ctx, tenantID, id); err != nil {
			return nil, err
		}
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE staff SET full_name = $3, phone = $4, role = $5::staff_role
		WHERE tenant_id = $1 AND id = $2 AND platform_user_id IS NULL`, tenantID, id, in.FullName, phone, in.Role); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return s.GetUser(ctx, tenantID, id)
}

// SetActive lets a user sign in, or stops them. A deactivated user is signed
// out at their next request and keeps everything they recorded.
func (s *Service) SetActive(ctx context.Context, tenantID, actorID, id uuid.UUID, active bool) (*User, error) {
	current, err := s.GetUser(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if !active {
		if id == actorID {
			return nil, invalid("", "You cannot deactivate your own account.")
		}
		if current.Role == "super_admin" {
			return nil, invalid("", "This account is managed by Shule360, not from inside the school.")
		}
		if current.Role == "principal" {
			if err := s.keepOnePrincipal(ctx, tenantID, id); err != nil {
				return nil, err
			}
		}
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE staff SET is_active = $3 WHERE tenant_id = $1 AND id = $2 AND platform_user_id IS NULL`,
		tenantID, id, active); err != nil {
		return nil, fmt.Errorf("set active: %w", err)
	}
	return s.GetUser(ctx, tenantID, id)
}

// ResetPassword gives a user a new generated password, returned once.
func (s *Service) ResetPassword(ctx context.Context, tenantID, id uuid.UUID) (*User, error) {
	user, err := s.GetUser(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if user.Role == "super_admin" {
		return nil, invalid("", "This account is managed by Shule360, not from inside the school.")
	}
	var authID *string
	if err := s.pool.QueryRow(ctx, `SELECT supabase_user_id::text FROM staff WHERE id = $1`, id).Scan(&authID); err != nil {
		return nil, err
	}
	password, err := newPassword()
	if err != nil {
		return nil, err
	}
	if authID != nil && *authID != "" {
		if err := s.auth.SetUserPassword(ctx, *authID, password); err != nil {
			return nil, fmt.Errorf("set password: %w", err)
		}
	} else {
		// A person recorded before accounts were created with the record.
		created, err := s.auth.CreateUser(ctx, user.Email, password)
		if err != nil {
			if alreadyRegistered(err) {
				return nil, invalid("", "%s has a sign-in account this school did not create, so its password cannot be reset from here. Contact Shule360 support.", user.Email)
			}
			return nil, fmt.Errorf("create sign-in account: %w", err)
		}
		if _, err := s.pool.Exec(context.WithoutCancel(ctx),
			`UPDATE staff SET supabase_user_id = $2::uuid WHERE id = $1`, id, created); err != nil {
			return nil, fmt.Errorf("store sign-in id: %w", err)
		}
	}
	user.TempPassword = password
	return user, nil
}
