// Package platform manages what sits above a school: school groups, the
// schools themselves, and the platform and group users who can open them.
// Every route here is for platform-scope sessions only.
package platform

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/tenant"
)

// AuthProvider is the part of Supabase Auth this package uses.
type AuthProvider interface {
	CreateUser(ctx context.Context, email, password string) (string, error)
	SetUserPassword(ctx context.Context, userID, password string) error
}

// Service implements platform administration.
type Service struct {
	pool *pgxpool.Pool
	auth AuthProvider
}

func NewService(pool *pgxpool.Pool, auth AuthProvider) *Service {
	return &Service{pool: pool, auth: auth}
}

// ErrNotFound is returned when the record does not exist.
var ErrNotFound = errors.New("not found")

// ValidationError is a refusal the caller can fix; it is shown as written.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// slugFrom makes a short name from a full one: "St. Mary's Academy" -> "st-marys-academy".
func slugFrom(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			if dash && b.Len() > 0 {
				b.WriteByte('-')
			}
			dash = false
			b.WriteRune(r)
		case r == '\'' || r == '’' || r == '.':
			// dropped: "mary's" -> "marys"
		default:
			dash = true
		}
	}
	return b.String()
}

func cleanSlug(slug, name string) (string, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		slug = slugFrom(name)
	}
	if len(slug) < 2 || len(slug) > 100 || !slugPattern.MatchString(slug) {
		return "", invalid("The short name may contain only lowercase letters, numbers and single dashes (for example \"juakali-primary\").")
	}
	return slug, nil
}

func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// --- Groups -----------------------------------------------------------------

// Group is a school group with the number of schools in it.
type Group struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	SchoolCount int       `json:"school_count"`
	UserCount   int       `json:"user_count"`
	CreatedAt   time.Time `json:"created_at"`
}

const groupSelect = `
	SELECT g.id, g.name, g.slug,
	       (SELECT COUNT(*) FROM tenants t WHERE t.group_id = g.id),
	       (SELECT COUNT(*) FROM platform_users u WHERE u.group_id = g.id AND u.is_active),
	       g.created_at
	FROM school_groups g`

func scanGroup(row pgx.Row) (*Group, error) {
	var g Group
	if err := row.Scan(&g.ID, &g.Name, &g.Slug, &g.SchoolCount, &g.UserCount, &g.CreatedAt); err != nil {
		return nil, err
	}
	return &g, nil
}

func (s *Service) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := s.pool.Query(ctx, groupSelect+` ORDER BY g.name`)
	if err != nil {
		return nil, fmt.Errorf("query groups: %w", err)
	}
	defer rows.Close()
	groups := []Group{}
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, *g)
	}
	return groups, rows.Err()
}

func (s *Service) GetGroup(ctx context.Context, id uuid.UUID) (*Group, error) {
	g, err := scanGroup(s.pool.QueryRow(ctx, groupSelect+` WHERE g.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return g, err
}

func (s *Service) CreateGroup(ctx context.Context, name, slug string) (*Group, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, invalid("Give the group a name.")
	}
	slug, err := cleanSlug(slug, name)
	if err != nil {
		return nil, err
	}
	var id uuid.UUID
	err = s.pool.QueryRow(ctx, `INSERT INTO school_groups (name, slug) VALUES ($1, $2) RETURNING id`, name, slug).Scan(&id)
	if uniqueViolation(err) {
		return nil, invalid("Another group already uses the short name %q.", slug)
	}
	if err != nil {
		return nil, fmt.Errorf("insert group: %w", err)
	}
	return s.GetGroup(ctx, id)
}

func (s *Service) RenameGroup(ctx context.Context, id uuid.UUID, name string) (*Group, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, invalid("Give the group a name.")
	}
	tag, err := s.pool.Exec(ctx, `UPDATE school_groups SET name = $2 WHERE id = $1`, id, name)
	if err != nil {
		return nil, fmt.Errorf("update group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetGroup(ctx, id)
}

// --- Schools ----------------------------------------------------------------

// School is a school as the platform manages it.
type School struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	GroupID   *uuid.UUID `json:"group_id,omitempty"`
	GroupName *string    `json:"group_name,omitempty"`
	Modules   []string   `json:"modules"`
}

// SetModules sets exactly which modules a school has.
func (s *Service) SetModules(ctx context.Context, id uuid.UUID, modules []string) (*School, error) {
	known := map[string]bool{}
	for _, m := range tenant.Modules {
		known[m] = true
	}
	chosen := []string{}
	seen := map[string]bool{}
	for _, m := range modules {
		if !known[m] {
			return nil, invalid("%s is not a module.", m)
		}
		if !seen[m] {
			seen[m] = true
			chosen = append(chosen, m)
		}
	}
	tag, err := s.pool.Exec(ctx, `UPDATE tenants SET modules = $2 WHERE id = $1`, id, chosen)
	if err != nil {
		return nil, fmt.Errorf("set modules: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.getSchool(ctx, id)
}

func (s *Service) getSchool(ctx context.Context, id uuid.UUID) (*School, error) {
	var sc School
	err := s.pool.QueryRow(ctx, `
		SELECT t.id, t.name, t.slug, t.group_id, g.name, COALESCE(t.modules, $2::text[])
		FROM tenants t LEFT JOIN school_groups g ON g.id = t.group_id
		WHERE t.id = $1`, id, tenant.Modules).Scan(&sc.ID, &sc.Name, &sc.Slug, &sc.GroupID, &sc.GroupName, &sc.Modules)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query school: %w", err)
	}
	return &sc, nil
}

func (s *Service) checkGroup(ctx context.Context, groupID *uuid.UUID) error {
	if groupID == nil {
		return nil
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM school_groups WHERE id = $1)`, *groupID).Scan(&exists); err != nil {
		return fmt.Errorf("query group: %w", err)
	}
	if !exists {
		return invalid("That group does not exist.")
	}
	return nil
}

func (s *Service) CreateSchool(ctx context.Context, name, slug string, groupID *uuid.UUID) (*School, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, invalid("Give the school a name.")
	}
	slug, err := cleanSlug(slug, name)
	if err != nil {
		return nil, err
	}
	if err := s.checkGroup(ctx, groupID); err != nil {
		return nil, err
	}
	var id uuid.UUID
	err = s.pool.QueryRow(ctx, `INSERT INTO tenants (name, slug, group_id) VALUES ($1, $2, $3) RETURNING id`, name, slug, groupID).Scan(&id)
	if uniqueViolation(err) {
		return nil, invalid("Another school already uses the short name %q.", slug)
	}
	if err != nil {
		return nil, fmt.Errorf("insert school: %w", err)
	}
	return s.getSchool(ctx, id)
}

// UpdateSchool renames a school and moves it between groups (nil = no group).
func (s *Service) UpdateSchool(ctx context.Context, id uuid.UUID, name string, groupID *uuid.UUID) (*School, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, invalid("Give the school a name.")
	}
	if err := s.checkGroup(ctx, groupID); err != nil {
		return nil, err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE tenants SET name = $2, group_id = $3 WHERE id = $1`, id, name, groupID)
	if err != nil {
		return nil, fmt.Errorf("update school: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.getSchool(ctx, id)
}

// --- Users ------------------------------------------------------------------

// User is a platform or group user.
type User struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	FullName  string     `json:"full_name"`
	Scope     string     `json:"scope"`
	GroupID   *uuid.UUID `json:"group_id,omitempty"`
	GroupName *string    `json:"group_name,omitempty"`
	Role      string     `json:"role"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	// TempPassword is set only in the answer to a create or a password reset.
	// It is never stored here and cannot be shown again.
	TempPassword string `json:"temp_password,omitempty"`
	// Note explains anything the administrator should pass on with the account.
	Note string `json:"note,omitempty"`
}

// UserInput is what creating or changing a user takes.
type UserInput struct {
	Email    string     `json:"email"`
	FullName string     `json:"full_name"`
	Scope    string     `json:"scope"`
	GroupID  *uuid.UUID `json:"group_id"`
	Role     string     `json:"role"`
}

const userSelect = `
	SELECT u.id, u.email, u.full_name, u.scope, u.group_id, g.name, u.role, u.is_active, u.created_at
	FROM platform_users u
	LEFT JOIN school_groups g ON g.id = u.group_id`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.FullName, &u.Scope, &u.GroupID, &u.GroupName, &u.Role, &u.IsActive, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, userSelect+` ORDER BY u.is_active DESC, u.full_name`)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	u, err := scanUser(s.pool.QueryRow(ctx, userSelect+` WHERE u.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

var validRoles = map[string]bool{
	"super_admin": true, "principal": true, "bursar": true, "hr": true, "transport_manager": true, "teacher": true,
}

// normalise checks a user's scope, group and role together.
func (s *Service) normalise(ctx context.Context, in *UserInput) error {
	in.FullName = strings.TrimSpace(in.FullName)
	if in.FullName == "" {
		return invalid("Enter the person's name.")
	}
	switch in.Scope {
	case "platform":
		if in.GroupID != nil {
			return invalid("A platform user is not tied to a group. Remove the group, or make them a group user.")
		}
		if in.Role == "" {
			in.Role = "super_admin"
		}
	case "group":
		if in.GroupID == nil {
			return invalid("Choose the group this person belongs to.")
		}
		if err := s.checkGroup(ctx, in.GroupID); err != nil {
			return err
		}
		if in.Role == "" {
			in.Role = "principal"
		}
	default:
		return invalid("Choose whether this person works across the platform or in one group.")
	}
	if !validRoles[in.Role] {
		return invalid("%q is not a role.", in.Role)
	}
	return nil
}

// CreateUser adds a platform or group user and their sign-in account.
//
// The password is generated here and returned once. If the email already has a
// sign-in account, that account is kept (the password is not changed) and the
// answer says so.
func (s *Service) CreateUser(ctx context.Context, in UserInput) (*User, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if at := strings.Index(in.Email, "@"); at <= 0 || at == len(in.Email)-1 || strings.ContainsAny(in.Email, " \t") {
		return nil, invalid("Enter a valid email address.")
	}
	if err := s.normalise(ctx, &in); err != nil {
		return nil, err
	}

	// Sign-in looks for school staff first, so an email that is already a
	// staff member would sign in as that staff member, never as this user.
	var staffSchool string
	err := s.pool.QueryRow(ctx, `
		SELECT t.name FROM staff st JOIN tenants t ON t.id = st.tenant_id
		WHERE lower(st.email) = $1 AND st.platform_user_id IS NULL LIMIT 1`, in.Email).Scan(&staffSchool)
	if err == nil {
		return nil, invalid("%s is already a staff member of %s. Use a different email for the platform account.", in.Email, staffSchool)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check staff email: %w", err)
	}

	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM platform_users WHERE lower(email) = $1)`, in.Email).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check user email: %w", err)
	}
	if exists {
		return nil, invalid("%s already has a platform account.", in.Email)
	}

	// The sign-in account is created before the row: a row without an account
	// could never sign in, while an account without a row is simply refused.
	password, err := newPassword()
	if err != nil {
		return nil, err
	}
	var authID *string
	note := ""
	if id, err := s.auth.CreateUser(ctx, in.Email, password); err == nil {
		authID = &id
	} else if alreadyRegistered(err) {
		password = ""
		note = "This email already had a sign-in account; its existing password still applies."
	} else {
		return nil, fmt.Errorf("create sign-in account: %w", err)
	}

	var id uuid.UUID
	err = s.pool.QueryRow(ctx, `
		INSERT INTO platform_users (email, full_name, scope, group_id, role, supabase_user_id)
		VALUES ($1, $2, $3, $4, $5, $6::uuid) RETURNING id
	`, in.Email, in.FullName, in.Scope, in.GroupID, in.Role, authID).Scan(&id)
	if uniqueViolation(err) {
		return nil, invalid("%s already has a platform account.", in.Email)
	}
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	user, err := s.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	user.TempPassword = password
	user.Note = note
	return user, nil
}

func alreadyRegistered(err error) bool {
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "already") || strings.Contains(text, "email_exists")
}

// UpdateUser changes a user's name, reach and role. actorID is who is asking.
func (s *Service) UpdateUser(ctx context.Context, actorID, id uuid.UUID, in UserInput) (*User, error) {
	current, err := s.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.normalise(ctx, &in); err != nil {
		return nil, err
	}
	if current.Scope == "platform" && in.Scope != "platform" {
		if id == actorID {
			return nil, invalid("You cannot remove your own platform access. Ask another platform administrator.")
		}
		if err := s.keepOnePlatformUser(ctx, id); err != nil {
			return nil, err
		}
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE platform_users SET full_name = $2, scope = $3, group_id = $4, role = $5 WHERE id = $1
	`, id, in.FullName, in.Scope, in.GroupID, in.Role); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return s.GetUser(ctx, id)
}

// SetActive deactivates or reactivates a user. A deactivated user is refused
// on their next request, whatever their session says.
func (s *Service) SetActive(ctx context.Context, actorID, id uuid.UUID, active bool) (*User, error) {
	current, err := s.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	if !active {
		if id == actorID {
			return nil, invalid("You cannot deactivate your own account.")
		}
		if current.Scope == "platform" && current.IsActive {
			if err := s.keepOnePlatformUser(ctx, id); err != nil {
				return nil, err
			}
		}
	}
	if _, err := s.pool.Exec(ctx, `UPDATE platform_users SET is_active = $2 WHERE id = $1`, id, active); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return s.GetUser(ctx, id)
}

// keepOnePlatformUser refuses a change that would leave nobody able to manage
// the platform.
func (s *Service) keepOnePlatformUser(ctx context.Context, leaving uuid.UUID) error {
	var others int
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM platform_users WHERE scope = 'platform' AND is_active AND id <> $1
	`, leaving).Scan(&others); err != nil {
		return fmt.Errorf("count platform users: %w", err)
	}
	if others == 0 {
		return invalid("This is the only active platform administrator. Add another before changing this one.")
	}
	return nil
}

// ResetPassword gives a user a new generated password, returned once.
func (s *Service) ResetPassword(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	var authID *string
	if err := s.pool.QueryRow(ctx, `SELECT supabase_user_id::text FROM platform_users WHERE id = $1`, id).Scan(&authID); err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	if authID == nil {
		return nil, invalid("This account's sign-in was created outside this screen, so its password cannot be reset here. Reset it in Supabase Auth.")
	}
	password, err := newPassword()
	if err != nil {
		return nil, err
	}
	if err := s.auth.SetUserPassword(ctx, *authID, password); err != nil {
		return nil, fmt.Errorf("reset password: %w", err)
	}
	user.TempPassword = password
	return user, nil
}

// newPassword returns a 16-character password from an alphabet with no
// look-alike characters (no 0/O, 1/l/I), so it can be read out or typed.
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
