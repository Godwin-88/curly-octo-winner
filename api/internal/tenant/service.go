package tenant

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tenant represents a school (multi-tenant record).
type Tenant struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	LogoURL             *string   `json:"logo_url,omitempty"`
	SubscriptionTier    string    `json:"subscription_tier"`
	WAPhoneNumberID     *string   `json:"wa_phone_number_id,omitempty"`
	WABusinessAccountID *string   `json:"wa_business_account_id,omitempty"`
	ATSenderID          *string   `json:"at_sender_id,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// Service handles tenant-related operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a new tenant service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// GetByID retrieves a tenant by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	var t Tenant
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, slug, logo_url, subscription_tier,
		       wa_phone_number_id, wa_business_account_id, at_sender_id,
		       created_at, updated_at
		FROM tenants
		WHERE id = $1
	`, id).Scan(
		&t.ID, &t.Name, &t.Slug, &t.LogoURL, &t.SubscriptionTier,
		&t.WAPhoneNumberID, &t.WABusinessAccountID, &t.ATSenderID,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tenant not found: %w", err)
		}
		return nil, fmt.Errorf("query tenant by id: %w", err)
	}
	return &t, nil
}

// GetBySlug retrieves a tenant by slug.
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Tenant, error) {
	var t Tenant
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, slug, logo_url, subscription_tier,
		       wa_phone_number_id, wa_business_account_id, at_sender_id,
		       created_at, updated_at
		FROM tenants
		WHERE slug = $1
	`, slug).Scan(
		&t.ID, &t.Name, &t.Slug, &t.LogoURL, &t.SubscriptionTier,
		&t.WAPhoneNumberID, &t.WABusinessAccountID, &t.ATSenderID,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tenant not found: %w", err)
		}
		return nil, fmt.Errorf("query tenant by slug: %w", err)
	}
	return &t, nil
}

// ListAll returns all tenants (admin only).
func (s *Service) ListAll(ctx context.Context) ([]Tenant, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, slug, logo_url, subscription_tier,
		       wa_phone_number_id, wa_business_account_id, at_sender_id,
		       created_at, updated_at
		FROM tenants
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query tenants: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Slug, &t.LogoURL, &t.SubscriptionTier,
			&t.WAPhoneNumberID, &t.WABusinessAccountID, &t.ATSenderID,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}

// Create inserts a new tenant.
func (s *Service) Create(ctx context.Context, name, slug string, senderID *string) (*Tenant, error) {
	var t Tenant
	err := s.pool.QueryRow(ctx, `
		INSERT INTO tenants (name, slug, at_sender_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, slug, logo_url, subscription_tier,
		          wa_phone_number_id, wa_business_account_id, at_sender_id,
		          created_at, updated_at
	`, name, slug, senderID).Scan(
		&t.ID, &t.Name, &t.Slug, &t.LogoURL, &t.SubscriptionTier,
		&t.WAPhoneNumberID, &t.WABusinessAccountID, &t.ATSenderID,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert tenant: %w", err)
	}
	return &t, nil
}

// UpdateSettings updates tenant WhatsApp/SMS settings.
func (s *Service) UpdateSettings(ctx context.Context, id uuid.UUID, waPhoneNumberID, waBusinessAccountID, atSenderID *string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tenants
		SET wa_phone_number_id = $2,
		    wa_business_account_id = $3,
		    at_sender_id = $4,
		    updated_at = now()
		WHERE id = $1
	`, id, waPhoneNumberID, waBusinessAccountID, atSenderID)
	if err != nil {
		return fmt.Errorf("update tenant settings: %w", err)
	}
	return nil
}

// School is a school as the context bar lists it.
type School struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	GroupID   *uuid.UUID `json:"group_id,omitempty"`
	GroupName *string    `json:"group_name,omitempty"`
	// Modules the school has. Always the full list: a school with no list of
	// its own has every module.
	Modules []string `json:"modules"`
}

// Modules that can be switched on or off per school. Anything not listed here
// is part of every school.
var Modules = []string{"communications", "finance"}

// effectiveModules turns the stored list (nil = everything) into the list
// that applies.
func effectiveModules(stored []string) []string {
	if stored == nil {
		return append([]string{}, Modules...)
	}
	return stored
}

// ModuleEnabled implements middleware.ModuleLookup.
func (s *Service) ModuleEnabled(ctx context.Context, schoolID uuid.UUID, module string) (bool, error) {
	var enabled bool
	err := s.pool.QueryRow(ctx,
		`SELECT modules IS NULL OR $2 = ANY(modules) FROM tenants WHERE id = $1`, schoolID, module).Scan(&enabled)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return enabled, err
}

// ListSchools returns the schools a session may open: one school (schoolID),
// the schools of a group (groupID), or every school (both nil).
func (s *Service) ListSchools(ctx context.Context, schoolID, groupID *uuid.UUID) ([]School, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.name, t.slug, t.group_id, g.name, t.modules
		FROM tenants t
		LEFT JOIN school_groups g ON g.id = t.group_id
		WHERE ($1::uuid IS NULL OR t.id = $1)
		  AND ($2::uuid IS NULL OR t.group_id = $2)
		ORDER BY g.name NULLS LAST, t.name
	`, schoolID, groupID)
	if err != nil {
		return nil, fmt.Errorf("query schools: %w", err)
	}
	defer rows.Close()

	schools := []School{}
	for rows.Next() {
		var sc School
		var stored []string
		if err := rows.Scan(&sc.ID, &sc.Name, &sc.Slug, &sc.GroupID, &sc.GroupName, &stored); err != nil {
			return nil, fmt.Errorf("scan school: %w", err)
		}
		sc.Modules = effectiveModules(stored)
		schools = append(schools, sc)
	}
	return schools, rows.Err()
}

// SchoolGroup implements middleware.SchoolLookup.
func (s *Service) SchoolGroup(ctx context.Context, schoolID uuid.UUID) (*uuid.UUID, bool, error) {
	var groupID *uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT group_id FROM tenants WHERE id = $1`, schoolID).Scan(&groupID)
	if err == pgx.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("query school: %w", err)
	}
	return groupID, true, nil
}

// OperatorStaff implements middleware.SchoolLookup: the staff row that stands
// for a platform or group user inside one school (see 041_platform_admin.sql).
// It is created the first time they open the school and kept in step with
// their name and role after that.
func (s *Service) OperatorStaff(ctx context.Context, schoolID, operatorID uuid.UUID) (uuid.UUID, string, bool, error) {
	var staffID uuid.UUID
	var role string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO staff (tenant_id, full_name, email, role, platform_user_id, is_active)
		SELECT $1,
		       p.full_name || CASE p.scope WHEN 'platform' THEN ' (Shule360 platform)' ELSE ' (group office)' END,
		       'platform+' || p.id || '@shule360.invalid',
		       p.role::staff_role,
		       p.id,
		       true
		FROM platform_users p
		WHERE p.id = $2 AND p.is_active = true
		ON CONFLICT (tenant_id, platform_user_id) WHERE platform_user_id IS NOT NULL
		DO UPDATE SET full_name = EXCLUDED.full_name, role = EXCLUDED.role, is_active = true
		RETURNING id, role::text
	`, schoolID, operatorID).Scan(&staffID, &role)
	if err == pgx.ErrNoRows {
		return uuid.Nil, "", false, nil // no such user, or deactivated
	}
	if err != nil {
		return uuid.Nil, "", false, fmt.Errorf("resolve operator staff: %w", err)
	}
	return staffID, role, true, nil
}
