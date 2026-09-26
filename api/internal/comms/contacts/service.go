package contacts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Contact is a person the school can message.
type Contact struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	FullName     string     `json:"full_name"`
	Phone        string     `json:"phone"`
	Email        *string    `json:"email,omitempty"`
	Relationship *string    `json:"relationship,omitempty"`
	GradeStream  *string    `json:"grade_stream,omitempty"`
	Tags         []string   `json:"tags"`
	Notes        *string    `json:"notes,omitempty"`
	Source       string     `json:"source"`
	GuardianID   *uuid.UUID `json:"guardian_id,omitempty"`
	IsActive     bool       `json:"is_active"`
	IsOptedOut   bool       `json:"is_opted_out"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ContactInput is the validated, normalized payload for a create or update.
// The phone is always E.164 by the time it reaches the service.
type ContactInput struct {
	FullName     string
	Phone        string // already normalized
	Email        string
	Relationship string
	GradeStream  string
	Tags         []string
	Notes        string
	IsOptedOut   bool
	GuardianID   *uuid.UUID
}

// Sentinel errors the handler maps to status codes.
var (
	ErrNotFound      = errors.New("contact not found")
	ErrAlreadyExists = errors.New("a contact with this phone number already exists")
)

const maxContactsPerPage = 500

// Service is the contact book data access layer.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a contacts service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

const contactColumns = `id, tenant_id, full_name, phone, email, relationship,
	grade_stream, tags, notes, source, guardian_id, is_active, is_opted_out,
	created_at, updated_at`

func scanContact(row pgx.Row) (*Contact, error) {
	var c Contact
	err := row.Scan(
		&c.ID, &c.TenantID, &c.FullName, &c.Phone, &c.Email, &c.Relationship,
		&c.GradeStream, &c.Tags, &c.Notes, &c.Source, &c.GuardianID,
		&c.IsActive, &c.IsOptedOut, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if c.Tags == nil {
		c.Tags = []string{}
	}
	return &c, nil
}

// List returns the tenant's contacts.
//
// Filters: search matches name/phone/email, tag requires membership, and
// status is "active" (default), "opted_out", "inactive" or "all".
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, search, tag, status string, limit, offset int) ([]Contact, int, error) {
	if limit <= 0 || limit > maxContactsPerPage {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	search = strings.TrimSpace(search)

	// Arguments are appended only as they are actually referenced, so the
	// placeholder numbering stays dense. Appending an unused $2 would make
	// Postgres reject the whole query.
	conditions := []string{"tenant_id = $1"}
	args := []any{tenantID}
	next := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	switch status {
	case "", "active":
		conditions = append(conditions, "is_active = true")
	case "inactive":
		conditions = append(conditions, "is_active = false")
	case "opted_out":
		conditions = append(conditions, "is_opted_out = true")
	case "all":
		// no status filter
	default:
		return nil, 0, fmt.Errorf("unknown status filter: %q", status)
	}

	if tag != "" {
		conditions = append(conditions, fmt.Sprintf("tags @> ARRAY[%s]::text[]", next(strings.ToLower(tag))))
	}
	if search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		p := next(pattern)
		conditions = append(conditions, fmt.Sprintf(
			"(LOWER(full_name) LIKE %s OR LOWER(phone) LIKE %s OR LOWER(COALESCE(email, '')) LIKE %s)", p, p, p))
	}
	where := strings.Join(conditions, " AND ")

	// Total for the "x of y" pager. Returned even when the page is empty.
	var total int
	if err := s.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM contacts WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count contacts: %w", err)
	}

	limitArg := next(limit)
	offsetArg := next(offset)

	rows, err := s.pool.Query(ctx,
		"SELECT "+contactColumns+" FROM contacts WHERE "+where+
			" ORDER BY full_name, id LIMIT "+limitArg+" OFFSET "+offsetArg, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list contacts: %w", err)
	}
	defer rows.Close()

	out := make([]Contact, 0, 16)
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan contact: %w", err)
		}
		out = append(out, *c)
	}
	return out, total, rows.Err()
}

// Summary powers the stat tiles at the top of the contacts screen.
type Summary struct {
	Total     int            `json:"total"`
	Active    int            `json:"active"`
	OptedOut  int            `json:"opted_out"`
	WithPhone int            `json:"with_phone"`
	ByTag     map[string]int `json:"by_tag"`
}

func (s *Service) Summary(ctx context.Context, tenantID uuid.UUID) (*Summary, error) {
	sum := &Summary{ByTag: make(map[string]int)}

	err := s.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE is_active),
			COUNT(*) FILTER (WHERE is_active AND is_opted_out),
			COUNT(*) FILTER (WHERE is_active AND phone <> '')
		FROM contacts
		WHERE tenant_id = $1
	`, tenantID).Scan(&sum.Active, &sum.OptedOut, &sum.WithPhone)
	if err != nil {
		return nil, fmt.Errorf("summarize contacts: %w", err)
	}
	sum.Total = sum.Active

	rows, err := s.pool.Query(ctx, `
		SELECT tag, COUNT(*)
		FROM contacts c, unnest(c.tags) AS tag
		WHERE c.tenant_id = $1 AND c.is_active
		GROUP BY tag
		ORDER BY COUNT(*) DESC, tag
		LIMIT 12
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("summarize tags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tag string
		var n int
		if err := rows.Scan(&tag, &n); err != nil {
			return nil, fmt.Errorf("scan tag count: %w", err)
		}
		sum.ByTag[tag] = n
	}
	return sum, rows.Err()
}

// Get returns a single contact scoped to the tenant.
func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*Contact, error) {
	c, err := scanContact(s.pool.QueryRow(ctx,
		"SELECT "+contactColumns+" FROM contacts WHERE tenant_id = $1 AND id = $2",
		tenantID, id))
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get contact: %w", err)
	}
	return c, nil
}

// Create inserts a contact entered through the UI. A duplicate phone in the
// same tenant is reported as ErrAlreadyExists so the UI can offer an update
// instead of silently creating a second row for the same human.
func (s *Service) Create(ctx context.Context, tenantID, actorID uuid.UUID, in ContactInput) (*Contact, error) {
	return s.CreateWithSource(ctx, tenantID, actorID, in, "manual")
}

// CreateWithSource inserts a contact and records where it came from
// ("manual" or "import"), so a school can tell curated entries from bulk data.
func (s *Service) CreateWithSource(ctx context.Context, tenantID, actorID uuid.UUID, in ContactInput, source string) (*Contact, error) {
	if source != "import" && source != "manual" && source != "guardian" {
		source = "manual"
	}
	c, err := scanContact(s.pool.QueryRow(ctx, `
		INSERT INTO contacts (tenant_id, full_name, phone, email, relationship,
			grade_stream, tags, notes, source, guardian_id, is_opted_out, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING `+contactColumns,
		tenantID, in.FullName, in.Phone, nullIfEmpty(in.Email), nullIfEmpty(in.Relationship),
		nullIfEmpty(in.GradeStream), in.Tags, nullIfEmpty(in.Notes), source, in.GuardianID, in.IsOptedOut, actorID))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("insert contact: %w", err)
	}
	return c, nil
}

// Update edits an existing contact. Every field is replaced with the submitted
// value (a blank field clears it) so the edit form is predictable.
func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, in ContactInput) (*Contact, error) {
	c, err := scanContact(s.pool.QueryRow(ctx, `
		UPDATE contacts SET
			full_name = $3,
			phone = $4,
			email = $5,
			relationship = $6,
			grade_stream = $7,
			tags = $8,
			notes = $9,
			is_opted_out = $10,
			guardian_id = $11
		WHERE tenant_id = $1 AND id = $2
		RETURNING `+contactColumns,
		tenantID, id, in.FullName, in.Phone, nullIfEmpty(in.Email), nullIfEmpty(in.Relationship),
		nullIfEmpty(in.GradeStream), in.Tags, nullIfEmpty(in.Notes), in.IsOptedOut, in.GuardianID))
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, fmt.Errorf("update contact: %w", err)
	}
	return c, nil
}

// Delete soft-deletes a contact. Delivery logs reference the contact, so the
// row is kept but excluded from audiences and the default listing.
func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		"UPDATE contacts SET is_active = false WHERE tenant_id = $1 AND id = $2 AND is_active = true",
		tenantID, id)
	if err != nil {
		return fmt.Errorf("delete contact: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Restore reactivates a soft-deleted contact.
func (s *Service) Restore(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx,
		"UPDATE contacts SET is_active = true WHERE tenant_id = $1 AND id = $2 AND is_active = false",
		tenantID, id)
	if err != nil {
		return fmt.Errorf("restore contact: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FindByPhone looks up a contact by its normalized number, including archived
// rows (so a re-import can revive a contact the school removed by mistake).
func (s *Service) FindByPhone(ctx context.Context, tenantID uuid.UUID, phone string) (*Contact, error) {
	c, err := scanContact(s.pool.QueryRow(ctx,
		"SELECT "+contactColumns+" FROM contacts WHERE tenant_id = $1 AND phone = $2",
		tenantID, phone))
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find contact by phone: %w", err)
	}
	return c, nil
}

// NamesByPhone returns a phone -> stored name map for the given numbers, used
// by the import dry run to tell the user which rows already exist.
func (s *Service) NamesByPhone(ctx context.Context, tenantID uuid.UUID, phones []string) (map[string]string, error) {
	if len(phones) == 0 {
		return map[string]string{}, nil
	}

	rows, err := s.pool.Query(ctx,
		"SELECT phone, full_name FROM contacts WHERE tenant_id = $1 AND phone = ANY($2::text[])",
		tenantID, phones)
	if err != nil {
		return nil, fmt.Errorf("look up contacts by phone: %w", err)
	}
	defer rows.Close()

	out := make(map[string]string, len(phones))
	for rows.Next() {
		var phone, name string
		if err := rows.Scan(&phone, &name); err != nil {
			return nil, fmt.Errorf("scan phone lookup: %w", err)
		}
		out[phone] = name
	}
	return out, rows.Err()
}

// Merge applies an imported row onto an existing contact: a blank name keeps
// the stored one, and tags/notes are unioned rather than replaced, so a partial
// spreadsheet cannot wipe data the school curated in the UI.
func (s *Service) Merge(ctx context.Context, tenantID, id uuid.UUID, in ContactInput) (*Contact, error) {
	c, err := scanContact(s.pool.QueryRow(ctx, `
		UPDATE contacts SET
			full_name = COALESCE(NULLIF($3, ''), full_name),
			email = COALESCE($4, email),
			relationship = COALESCE($5, relationship),
			grade_stream = COALESCE($6, grade_stream),
			tags = (SELECT COALESCE(array_agg(DISTINCT t), '{}') FROM unnest(tags || $7) AS t),
			notes = COALESCE($8, notes),
			is_opted_out = $9
		WHERE tenant_id = $1 AND id = $2
		RETURNING `+contactColumns,
		tenantID, id, in.FullName, nullIfEmpty(in.Email), nullIfEmpty(in.Relationship),
		nullIfEmpty(in.GradeStream), in.Tags, nullIfEmpty(in.Notes), in.IsOptedOut))
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("merge contact: %w", err)
	}
	return c, nil
}

// nullIfEmpty maps "" to a nil pointer so the column stores NULL instead of an
// empty string, and so the COALESCE fallbacks in Merge behave.
func nullIfEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// isUniqueViolation reports whether err is a Postgres unique-constraint error.
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return strings.Contains(err.Error(), "23505")
}
