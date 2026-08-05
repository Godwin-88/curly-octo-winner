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
