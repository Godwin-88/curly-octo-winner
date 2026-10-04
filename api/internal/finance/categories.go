package finance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shule360/api/pkg/httputil"
)

// FeeCategory is something a school charges for. The list is the school's
// own: nothing here is built in, and a school adds, renames and retires
// entries as it needs.
type FeeCategory struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsOptional  bool      `json:"is_optional"`
	IsActive    bool      `json:"is_active"`
	// How many fee structures charge it now.
	InUse     int       `json:"in_use"`
	CreatedAt time.Time `json:"created_at"`
}

type FeeCategoryInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	IsOptional  *bool   `json:"is_optional,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// DefaultFeeCategories is what a new school starts with. It is a starting
// point only: every entry can be renamed or retired, and more added.
//
// A public school does not charge tuition, so its list has the levies public
// schools do collect.
func DefaultFeeCategories(ownership string) []FeeCategoryInput {
	optional, required := true, false
	item := func(name string, isOptional bool) FeeCategoryInput {
		o := required
		if isOptional {
			o = optional
		}
		return FeeCategoryInput{Name: name, IsOptional: &o}
	}
	if ownership == "public" {
		return []FeeCategoryInput{
			item("Lunch programme", true), item("Activity levy", false), item("Assessment and exam levy", false),
			item("Development levy", false), item("Remedial teaching", true), item("Transport", true), item("Boarding", true),
		}
	}
	return []FeeCategoryInput{
		item("Tuition", false), item("Activity fee", false), item("Lunch", true),
		item("Transport", true), item("Boarding", true), item("Caution money", false),
	}
}

const categoryColumns = `c.id, c.name, c.description, c.is_optional, c.is_active,
	(SELECT COUNT(DISTINCT i.fee_structure_id) FROM fee_structure_items i
	  WHERE i.tenant_id = c.tenant_id AND lower(i.name) = lower(c.name)),
	c.created_at`

func scanCategory(row pgx.Row) (*FeeCategory, error) {
	var c FeeCategory
	if err := row.Scan(&c.ID, &c.Name, &c.Description, &c.IsOptional, &c.IsActive, &c.InUse, &c.CreatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

// ListFeeCategories returns the school's fee items. status "all" includes the
// retired ones.
func (s *Service) ListFeeCategories(ctx context.Context, tenantID uuid.UUID, status string) ([]FeeCategory, error) {
	query := `SELECT ` + categoryColumns + ` FROM fee_categories c WHERE c.tenant_id = $1`
	switch status {
	case "all":
	case "retired":
		query += ` AND NOT c.is_active`
	default:
		query += ` AND c.is_active`
	}
	rows, err := s.pool.Query(ctx, query+` ORDER BY c.is_active DESC, c.sort_order, c.name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []FeeCategory{}
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (s *Service) GetFeeCategory(ctx context.Context, tenantID, id uuid.UUID) (*FeeCategory, error) {
	c, err := scanCategory(s.pool.QueryRow(ctx,
		`SELECT `+categoryColumns+` FROM fee_categories c WHERE c.tenant_id = $1 AND c.id = $2`, tenantID, id))
	return c, notFound(err)
}

func cleanCategoryName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	if name == "" {
		return "", invalid("Give the fee item a name, such as Lunch or Uniform.")
	}
	if len(name) > 100 {
		return "", invalid("That name is too long.")
	}
	return name, nil
}

// CreateFeeCategory adds something the school charges for.
func (s *Service) CreateFeeCategory(ctx context.Context, tenantID uuid.UUID, in FeeCategoryInput) (*FeeCategory, error) {
	name, err := cleanCategoryName(in.Name)
	if err != nil {
		return nil, err
	}
	var id uuid.UUID
	err = s.pool.QueryRow(ctx, `
		INSERT INTO fee_categories (tenant_id, name, description, is_optional, sort_order)
		VALUES ($1, $2, $3, $4, (SELECT COALESCE(MAX(sort_order), 0) + 1 FROM fee_categories WHERE tenant_id = $1))
		RETURNING id`, tenantID, name, trimmed(in.Description), in.IsOptional != nil && *in.IsOptional).Scan(&id)
	if httputil.IsUniqueViolation(err) {
		return nil, conflict("The school already has a fee item called %s. If it was retired, bring it back instead.", name)
	}
	if err != nil {
		return nil, fmt.Errorf("insert fee item: %w", err)
	}
	return s.GetFeeCategory(ctx, tenantID, id)
}

// UpdateFeeCategory renames a fee item, changes whether it is optional, or
// retires it. Fee structures and invoices already made keep the name and
// amount they were made with.
func (s *Service) UpdateFeeCategory(ctx context.Context, tenantID, id uuid.UUID, in FeeCategoryInput) (*FeeCategory, error) {
	var name *string
	if strings.TrimSpace(in.Name) != "" {
		clean, err := cleanCategoryName(in.Name)
		if err != nil {
			return nil, err
		}
		name = &clean
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE fee_categories SET
			name = COALESCE($3, name),
			description = COALESCE($4, description),
			is_optional = COALESCE($5, is_optional),
			is_active = COALESCE($6, is_active)
		WHERE tenant_id = $1 AND id = $2`, tenantID, id, name, in.Description, in.IsOptional, in.IsActive)
	if httputil.IsUniqueViolation(err) {
		return nil, conflict("The school already has a fee item called %s.", *name)
	}
	if err != nil {
		return nil, fmt.Errorf("update fee item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetFeeCategory(ctx, tenantID, id)
}

// SeedFeeCategories gives a school its starting list, leaving alone anything
// it already has. It runs inside the transaction that creates the school.
func SeedFeeCategories(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, ownership string) error {
	for order, c := range DefaultFeeCategories(ownership) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO fee_categories (tenant_id, name, is_optional, sort_order)
			VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
			tenantID, c.Name, c.IsOptional != nil && *c.IsOptional, order+1); err != nil {
			return fmt.Errorf("seed fee items: %w", err)
		}
	}
	return nil
}
