// Package settings implements the tenant-facing Settings area: the school
// profile, the operational configuration a principal maintains, and the
// integration credentials (M-Pesa, Africa's Talking, WhatsApp, Backblaze, Groq,
// Upstash) that make those features work for their school.
//
// Design rules that matter here:
//   - tenant_id always comes from the verified JWT (set by the auth
//     middleware), never from the request body.
//   - writes are restricted to principal/super_admin in the handler.
//   - credentials are encrypted at rest and never returned to the browser;
//     reads report which secret fields exist, not their values.
//   - a school can inherit the platform's environment credentials
//     (use_platform_default) instead of entering its own.
//   - every change is written to audit_logs.
package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Providers lists the supported integration providers.
var Providers = []string{"mpesa", "africastalking", "whatsapp", "backblaze", "groq", "upstash"}

// IsProvider reports whether provider is supported.
func IsProvider(provider string) bool {
	for _, p := range Providers {
		if p == provider {
			return true
		}
	}
	return false
}

// Profile is the school profile shown on the Settings → School tab.
type Profile struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	LogoURL             *string   `json:"logo_url,omitempty"`
	SubscriptionTier    string    `json:"subscription_tier"`
	Phone               *string   `json:"phone,omitempty"`
	Email               *string   `json:"email,omitempty"`
	Address             *string   `json:"address,omitempty"`
	County              *string   `json:"county,omitempty"`
	MPesaShortcode      *string   `json:"mpesa_shortcode,omitempty"`
	MPesaAccountBasis   string    `json:"mpesa_account_basis"`
	MPesaCallbackURL    *string   `json:"mpesa_callback_url,omitempty"`
	WAPhoneNumberID     *string   `json:"wa_phone_number_id,omitempty"`
	WABusinessAccountID *string   `json:"wa_business_account_id,omitempty"`
	ATSenderID          *string   `json:"at_sender_id,omitempty"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ProfilePatch is a partial update; nil fields are left untouched.
type ProfilePatch struct {
	Name                *string `json:"name"`
	LogoURL             *string `json:"logo_url"`
	Phone               *string `json:"phone"`
	Email               *string `json:"email"`
	Address             *string `json:"address"`
	County              *string `json:"county"`
	MPesaShortcode      *string `json:"mpesa_shortcode"`
	MPesaAccountBasis   *string `json:"mpesa_account_basis"`
	MPesaCallbackURL    *string `json:"mpesa_callback_url"`
	WAPhoneNumberID     *string `json:"wa_phone_number_id"`
	WABusinessAccountID *string `json:"wa_business_account_id"`
	ATSenderID          *string `json:"at_sender_id"`
}

// GradingBand is one grade band (e.g. 70-79 = "B", 3 points).
type GradingBand struct {
	Min    int     `json:"min"`
	Label  string  `json:"label"`
	Points float64 `json:"points"`
}

// Settings is the operational configuration a principal maintains.
type Settings struct {
	TenantID                       uuid.UUID     `json:"tenant_id"`
	CurrentTerm                    *string       `json:"current_term,omitempty"`
	CurrentAcademicYear            *string       `json:"current_academic_year,omitempty"`
	GradingScale                   []GradingBand `json:"grading_scale"`
	AttendanceTime                 string        `json:"attendance_time"`
	AttendanceDeadline             string        `json:"attendance_deadline"`
	ReportCardFooter               *string       `json:"report_card_footer,omitempty"`
	ReceiptFooter                  *string       `json:"receipt_footer,omitempty"`
	MPesaEnabled                   bool          `json:"mpesa_enabled"`
	SMSEnabled                     bool          `json:"sms_enabled"`
	WhatsAppEnabled                bool          `json:"whatsapp_enabled"`
	RequireParentConsent           bool          `json:"require_parent_consent"`
	RequireStaffApprovalOnTransfer bool          `json:"require_staff_approval_on_transfer"`
	UpdatedAt                      time.Time     `json:"updated_at"`
	UpdatedBy                      *uuid.UUID    `json:"updated_by,omitempty"`
}

// SettingsPatch is a partial update of the operational configuration; nil
// fields are left untouched.
type SettingsPatch struct {
	CurrentTerm                    *string        `json:"current_term"`
	CurrentAcademicYear            *string        `json:"current_academic_year"`
	GradingScale                   *[]GradingBand `json:"grading_scale"`
	AttendanceTime                 *string        `json:"attendance_time"`
	AttendanceDeadline             *string        `json:"attendance_deadline"`
	ReportCardFooter               *string        `json:"report_card_footer"`
	ReceiptFooter                  *string        `json:"receipt_footer"`
	MPesaEnabled                   *bool          `json:"mpesa_enabled"`
	SMSEnabled                     *bool          `json:"sms_enabled"`
	WhatsAppEnabled                *bool          `json:"whatsapp_enabled"`
	RequireParentConsent           *bool          `json:"require_parent_consent"`
	RequireStaffApprovalOnTransfer *bool          `json:"require_staff_approval_on_transfer"`
}

// Integration is one provider's configuration for a school. Secrets are never
// included: only the names of the secret fields that are set.
type Integration struct {
	Provider           string         `json:"provider"`
	Label              *string        `json:"label,omitempty"`
	IsEnabled          bool           `json:"is_enabled"`
	UsePlatformDefault bool           `json:"use_platform_default"`
	Config             map[string]any `json:"config"`
	SecretFields       []string       `json:"secret_fields"`
	LastTestedAt       *time.Time     `json:"last_tested_at,omitempty"`
	LastTestStatus     string         `json:"last_test_status"`
	LastTestMessage    *string        `json:"last_test_message,omitempty"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// IntegrationInput is what the Settings → Integrations form submits.
type IntegrationInput struct {
	Label              *string        `json:"label"`
	IsEnabled          *bool          `json:"is_enabled"`
	UsePlatformDefault *bool          `json:"use_platform_default"`
	Config             map[string]any `json:"config"`
	// Secrets are merged into the stored set: a field left out (or empty)
	// keeps its current value, so the browser never has to receive a secret to
	// change a different one.
	Secrets map[string]string `json:"secrets"`
	// ClearSecrets removes the named secret fields outright.
	ClearSecrets []string `json:"clear_secrets"`
}

// Service is the settings data access layer.
type Service struct {
	pool   *pgxpool.Pool
	sealer *Sealer
}

// NewService creates the settings service. sealer may be nil, in which case
// credential writes are rejected rather than stored in plaintext.
func NewService(pool *pgxpool.Pool, sealer *Sealer) *Service {
	return &Service{pool: pool, sealer: sealer}
}

// defaultGradingScale is used when a school has never customised its scale.
var defaultGradingScale = []GradingBand{
	{Min: 80, Label: "A", Points: 4},
	{Min: 70, Label: "B", Points: 3},
	{Min: 60, Label: "C", Points: 2},
	{Min: 50, Label: "D", Points: 1},
	{Min: 0, Label: "E", Points: 0},
}

// GetSettings returns the operational settings, creating the tenant's row on
// first access so a school never has to be "initialised" before it can use the
// screen.
func (s *Service) GetSettings(ctx context.Context, tenantID uuid.UUID) (*Settings, error) {
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO tenant_settings (tenant_id) VALUES ($1) ON CONFLICT (tenant_id) DO NOTHING
	`, tenantID); err != nil {
		return nil, fmt.Errorf("ensure tenant settings row: %w", err)
	}

	var out Settings
	var scaleJSON []byte
	err := s.pool.QueryRow(ctx, `
		SELECT tenant_id, current_term, current_academic_year, grading_scale,
		       attendance_time, attendance_deadline, report_card_footer, receipt_footer,
		       mpesa_enabled, sms_enabled, whatsapp_enabled,
		       require_parent_consent, require_staff_approval_on_transfer,
		       updated_at, updated_by
		FROM tenant_settings WHERE tenant_id = $1
	`, tenantID).Scan(
		&out.TenantID, &out.CurrentTerm, &out.CurrentAcademicYear, &scaleJSON,
		&out.AttendanceTime, &out.AttendanceDeadline, &out.ReportCardFooter, &out.ReceiptFooter,
		&out.MPesaEnabled, &out.SMSEnabled, &out.WhatsAppEnabled,
		&out.RequireParentConsent, &out.RequireStaffApprovalOnTransfer,
		&out.UpdatedAt, &out.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("query tenant settings: %w", err)
	}
	out.GradingScale = defaultGradingScale
	if len(scaleJSON) > 0 {
		if err := json.Unmarshal(scaleJSON, &out.GradingScale); err != nil {
			// A malformed scale must not break the whole screen: fall back.
			out.GradingScale = defaultGradingScale
		}
	}
	return &out, nil
}

// validClock accepts 24h HH:MM, which is what the attendance window stores.
func validClock(v string) bool {
	if len(v) != 5 || v[2] != ':' {
		return false
	}
	for i, r := range v {
		if i == 2 {
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// UpdateSettings applies a partial update to the operational configuration.
func (s *Service) UpdateSettings(ctx context.Context, tenantID, staffID uuid.UUID, patch SettingsPatch) (*Settings, error) {
	current, err := s.GetSettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if patch.AttendanceTime != nil {
		v := strings.TrimSpace(*patch.AttendanceTime)
		if !validClock(v) {
			return nil, fmt.Errorf("attendance_time must be HH:MM")
		}
		patch.AttendanceTime = &v
	}
	if patch.AttendanceDeadline != nil {
		v := strings.TrimSpace(*patch.AttendanceDeadline)
		if !validClock(v) {
			return nil, fmt.Errorf("attendance_deadline must be HH:MM")
		}
		patch.AttendanceDeadline = &v
	}
	if patch.GradingScale != nil {
		scale := *patch.GradingScale
		if len(scale) == 0 || len(scale) > 20 {
			return nil, fmt.Errorf("grading_scale must contain 1-20 bands")
		}
		for i, band := range scale {
			if strings.TrimSpace(band.Label) == "" {
				return nil, fmt.Errorf("grading band %d needs a label", i+1)
			}
			if band.Min < 0 || band.Min > 100 {
				return nil, fmt.Errorf("grading band %d min must be 0-100", i+1)
			}
		}
		patch.GradingScale = &scale
	}

	scaleJSON := []byte("[]")
	if patch.GradingScale != nil {
		raw, err := json.Marshal(*patch.GradingScale)
		if err != nil {
			return nil, fmt.Errorf("encode grading scale: %w", err)
		}
		scaleJSON = raw
	} else if raw, err := json.Marshal(current.GradingScale); err == nil {
		scaleJSON = raw
	}

	var staffArg any
	if staffID != uuid.Nil {
		staffArg = staffID
	}

	if _, err := s.pool.Exec(ctx, `
		INSERT INTO tenant_settings (
			tenant_id, current_term, current_academic_year, grading_scale,
			attendance_time, attendance_deadline, report_card_footer, receipt_footer,
			mpesa_enabled, sms_enabled, whatsapp_enabled,
			require_parent_consent, require_staff_approval_on_transfer, updated_by
		) VALUES ($1, $2, $3, $4::jsonb,
		          COALESCE($5, '07:30'), COALESCE($6, '09:00'), $7, $8,
		          COALESCE($9, false), COALESCE($10, true), COALESCE($11, false),
		          COALESCE($12, true), COALESCE($13, true), $14)
		ON CONFLICT (tenant_id) DO UPDATE SET
			current_term = COALESCE($2, tenant_settings.current_term),
			current_academic_year = COALESCE($3, tenant_settings.current_academic_year),
			grading_scale = $4,
			attendance_time = COALESCE($5, tenant_settings.attendance_time),
			attendance_deadline = COALESCE($6, tenant_settings.attendance_deadline),
			report_card_footer = COALESCE($7, tenant_settings.report_card_footer),
			receipt_footer = COALESCE($8, tenant_settings.receipt_footer),
			mpesa_enabled = COALESCE($9, tenant_settings.mpesa_enabled),
			sms_enabled = COALESCE($10, tenant_settings.sms_enabled),
			whatsapp_enabled = COALESCE($11, tenant_settings.whatsapp_enabled),
			require_parent_consent = COALESCE($12, tenant_settings.require_parent_consent),
			require_staff_approval_on_transfer = COALESCE($13, tenant_settings.require_staff_approval_on_transfer),
			updated_by = $14,
			updated_at = now()
	`, tenantID,
		patch.CurrentTerm, patch.CurrentAcademicYear, string(scaleJSON),
		patch.AttendanceTime, patch.AttendanceDeadline, patch.ReportCardFooter, patch.ReceiptFooter,
		patch.MPesaEnabled, patch.SMSEnabled, patch.WhatsAppEnabled,
		patch.RequireParentConsent, patch.RequireStaffApprovalOnTransfer, staffArg,
	); err != nil {
		return nil, fmt.Errorf("update tenant settings: %w", err)
	}

	return s.GetSettings(ctx, tenantID)
}

func (s *Service) GetProfile(ctx context.Context, tenantID uuid.UUID) (*Profile, error) {
	var p Profile
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, slug, logo_url, subscription_tier,
		       phone, email, address, county,
		       mpesa_shortcode, mpesa_account_basis, mpesa_callback_url,
		       wa_phone_number_id, wa_business_account_id, at_sender_id, updated_at
		FROM tenants WHERE id = $1
	`, tenantID).Scan(
		&p.ID, &p.Name, &p.Slug, &p.LogoURL, &p.SubscriptionTier,
		&p.Phone, &p.Email, &p.Address, &p.County,
		&p.MPesaShortcode, &p.MPesaAccountBasis, &p.MPesaCallbackURL,
		&p.WAPhoneNumberID, &p.WABusinessAccountID, &p.ATSenderID, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("tenant %s: %w", tenantID, pgx.ErrNoRows)
		}
		return nil, fmt.Errorf("query tenant profile: %w", err)
	}
	return &p, nil
}

// UpdateProfile applies a partial profile update.
func (s *Service) UpdateProfile(ctx context.Context, tenantID uuid.UUID, patch ProfilePatch) (*Profile, error) {
	if patch.Name != nil {
		name := strings.TrimSpace(*patch.Name)
		if name == "" {
			return nil, fmt.Errorf("school name is required")
		}
		patch.Name = &name
	}
	if patch.MPesaAccountBasis != nil {
		basis := strings.ToLower(strings.TrimSpace(*patch.MPesaAccountBasis))
		switch basis {
		case "phone", "account", "customer":
		default:
			return nil, fmt.Errorf("mpesa_account_basis must be phone, account or customer")
		}
		patch.MPesaAccountBasis = &basis
	}

	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		UPDATE tenants SET
			name = COALESCE($2, name),
			logo_url = COALESCE($3, logo_url),
			phone = COALESCE($4, phone),
			email = COALESCE($5, email),
			address = COALESCE($6, address),
			county = COALESCE($7, county),
			mpesa_shortcode = COALESCE($8, mpesa_shortcode),
			mpesa_account_basis = COALESCE($9, mpesa_account_basis),
			mpesa_callback_url = COALESCE($10, mpesa_callback_url),
			wa_phone_number_id = COALESCE($11, wa_phone_number_id),
			wa_business_account_id = COALESCE($12, wa_business_account_id),
			at_sender_id = COALESCE($13, at_sender_id),
			updated_at = now()
		WHERE id = $1
		RETURNING id
	`, tenantID, patch.Name, patch.LogoURL, patch.Phone, patch.Email, patch.Address,
		patch.County, patch.MPesaShortcode, patch.MPesaAccountBasis, patch.MPesaCallbackURL,
		patch.WAPhoneNumberID, patch.WABusinessAccountID, patch.ATSenderID).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("update tenant profile: %w", err)
	}
	return s.GetProfile(ctx, tenantID)
}

// ListIntegrations returns every supported provider for the tenant. Providers
// the school has never configured are still returned (with defaults) so the UI
// can render a stable set of cards.
func (s *Service) ListIntegrations(ctx context.Context, tenantID uuid.UUID) ([]Integration, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT provider, label, is_enabled, use_platform_default, config,
		       secrets_encrypted, last_tested_at, last_test_status, last_test_message, updated_at
		FROM tenant_integrations WHERE tenant_id = $1
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query tenant integrations: %w", err)
	}
	defer rows.Close()

	byProvider := map[string]Integration{}
	for rows.Next() {
		var (
			in     Integration
			config []byte
			sealed *string
			status *string
		)
		if err := rows.Scan(&in.Provider, &in.Label, &in.IsEnabled, &in.UsePlatformDefault,
			&config, &sealed, &in.LastTestedAt, &status, &in.LastTestMessage, &in.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tenant integration: %w", err)
		}
		in.Config = map[string]any{}
		if len(config) > 0 {
			_ = json.Unmarshal(config, &in.Config)
		}
		in.SecretFields = s.secretFieldNames(sealed)
		in.LastTestStatus = "never"
		if status != nil && *status != "" {
			in.LastTestStatus = *status
		}
		byProvider[in.Provider] = in
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]Integration, 0, len(Providers))
	for _, provider := range Providers {
		if in, ok := byProvider[provider]; ok {
			out = append(out, in)
			continue
		}
		out = append(out, Integration{
			Provider:           provider,
			UsePlatformDefault: true,
			Config:             map[string]any{},
			SecretFields:       []string{},
			LastTestStatus:     "never",
		})
	}
	return out, nil
}

// secretFieldNames reports which secret fields are stored, without revealing
// any value: the sealed blob is opened only long enough to read its keys.
func (s *Service) secretFieldNames(sealed *string) []string {
	fields := []string{}
	if sealed == nil || *sealed == "" || s.sealer == nil {
		return fields
	}
	plain, err := s.sealer.Open(*sealed)
	if err != nil {
		return fields
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(plain), &m); err != nil {
		return fields
	}
	for k, v := range m {
		if v != "" {
			fields = append(fields, k)
		}
	}
	sort.Strings(fields)
	return fields
}

// SaveIntegration creates or updates a provider's configuration, merging
// secrets so the browser never has to receive an existing credential.
func (s *Service) SaveIntegration(ctx context.Context, tenantID, staffID uuid.UUID, provider string, in IntegrationInput) (*Integration, error) {
	if !IsProvider(provider) {
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}

	existing, err := s.getIntegrationRow(ctx, tenantID, provider)
	if err != nil {
		return nil, err
	}

	// Merge secrets: start from what is stored, drop cleared fields, add the
	// new non-empty ones. Values the browser leaves blank stay untouched.
	merged := map[string]string{}
	if existing.sealed != nil && *existing.sealed != "" && s.sealer != nil {
		if plain, err := s.sealer.Open(*existing.sealed); err == nil {
			_ = json.Unmarshal([]byte(plain), &merged)
		}
	}
	for _, field := range in.ClearSecrets {
		delete(merged, field)
	}
	changed := make([]string, 0, len(in.Secrets))
	for field, value := range in.Secrets {
		if strings.TrimSpace(value) == "" {
			continue
		}
		merged[field] = value
		changed = append(changed, field)
	}
	sort.Strings(changed)

	var sealed *string
	switch {
	case len(merged) > 0:
		if s.sealer == nil {
			return nil, fmt.Errorf("credential storage is unavailable on this server (set SETTINGS_ENCRYPTION_KEY)")
		}
		raw, err := json.Marshal(merged)
		if err != nil {
			return nil, fmt.Errorf("encode credentials: %w", err)
		}
		blob, err := s.sealer.Seal(string(raw))
		if err != nil {
			return nil, fmt.Errorf("encrypt credentials: %w", err)
		}
		sealed = &blob
	case len(in.ClearSecrets) > 0:
		sealed = nil
	}

	configJSON, err := json.Marshal(in.Config)
	if err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}
	var staffArg any
	if staffID != uuid.Nil {
		staffArg = staffID
	}

	isEnabled := true
	if in.IsEnabled != nil {
		isEnabled = *in.IsEnabled
	}
	useDefault := true
	if in.UsePlatformDefault != nil {
		useDefault = *in.UsePlatformDefault
	}

	if _, err := s.pool.Exec(ctx, `
		INSERT INTO tenant_integrations (
			tenant_id, provider, label, is_enabled, use_platform_default,
			config, secrets_encrypted, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
		ON CONFLICT (tenant_id, provider) DO UPDATE SET
			label = COALESCE($3, tenant_integrations.label),
			is_enabled = $4,
			use_platform_default = $5,
			config = $6,
			secrets_encrypted = COALESCE($7, tenant_integrations.secrets_encrypted),
			updated_by = $8,
			updated_at = now()
	`, tenantID, provider, in.Label, isEnabled, useDefault, string(configJSON), sealed, staffArg); err != nil {
		return nil, fmt.Errorf("save tenant integration: %w", err)
	}

	return s.getIntegration(ctx, tenantID, provider)
}

// DeleteIntegration removes a provider's configuration and credentials.
func (s *Service) DeleteIntegration(ctx context.Context, tenantID uuid.UUID, provider string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM tenant_integrations WHERE tenant_id = $1 AND provider = $2`, tenantID, provider)
	if err != nil {
		return fmt.Errorf("delete tenant integration: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("no %s configuration for this school", provider)
	}
	return nil
}

type integrationRow struct {
	integration Integration
	config      []byte
	sealed      *string
	status      *string
}

func (s *Service) getIntegrationRow(ctx context.Context, tenantID uuid.UUID, provider string) (*integrationRow, error) {
	var (
		row    integrationRow
		config []byte
	)
	err := s.pool.QueryRow(ctx, `
		SELECT provider, label, is_enabled, use_platform_default, config,
		       secrets_encrypted, last_tested_at, last_test_status, last_test_message, updated_at
		FROM tenant_integrations WHERE tenant_id = $1 AND provider = $2
	`, tenantID, provider).Scan(
		&row.integration.Provider, &row.integration.Label, &row.integration.IsEnabled,
		&row.integration.UsePlatformDefault, &config, &row.sealed,
		&row.integration.LastTestedAt, &row.status, &row.integration.LastTestMessage,
		&row.integration.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return &integrationRow{}, nil // nothing stored yet
	}
	if err != nil {
		return nil, fmt.Errorf("query integration: %w", err)
	}
	row.config = config
	return &row, nil
}

func (s *Service) getIntegration(ctx context.Context, tenantID uuid.UUID, provider string) (*Integration, error) {
	row, err := s.getIntegrationRow(ctx, tenantID, provider)
	if err != nil {
		return nil, err
	}
	out := row.integration
	out.Config = map[string]any{}
	if len(row.config) > 0 {
		_ = json.Unmarshal(row.config, &out.Config)
	}
	out.SecretFields = s.secretFieldNames(row.sealed)
	out.LastTestStatus = "never"
	if row.status != nil && *row.status != "" {
		out.LastTestStatus = *row.status
	}
	return &out, nil
}

// RecordTestResult stores the outcome of a connectivity test so a principal can
// see at a glance whether each provider actually works.
func (s *Service) RecordTestResult(ctx context.Context, tenantID uuid.UUID, provider, status, message string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE tenant_integrations
		SET last_tested_at = now(), last_test_status = $3, last_test_message = $4, updated_at = now()
		WHERE tenant_id = $1 AND provider = $2
	`, tenantID, provider, status, nullIfEmpty(message))
	if err != nil {
		return fmt.Errorf("record integration test: %w", err)
	}
	return nil
}

// ResolveSecrets returns the effective configuration for a provider: the
// school's own stored values, or the platform defaults when the school
// inherits them. This is the hook other services use to honour per-tenant
// credentials at runtime.
func (s *Service) ResolveSecrets(ctx context.Context, tenantID uuid.UUID, provider string) (config map[string]any, secrets map[string]string, usePlatformDefault bool, err error) {
	row, err := s.getIntegrationRow(ctx, tenantID, provider)
	if err != nil {
		return nil, nil, false, err
	}
	if row.integration.Provider == "" {
		return nil, nil, true, nil // not configured: platform defaults apply
	}
	config = map[string]any{}
	if len(row.config) > 0 {
		_ = json.Unmarshal(row.config, &config)
	}
	if row.sealed == nil || *row.sealed == "" {
		return config, nil, row.integration.UsePlatformDefault, nil
	}
	if s.sealer == nil {
		return nil, nil, false, fmt.Errorf("credential storage is unavailable on this server")
	}
	plain, err := s.sealer.Open(*row.sealed)
	if err != nil {
		return nil, nil, false, err
	}
	secrets = map[string]string{}
	if err := json.Unmarshal([]byte(plain), &secrets); err != nil {
		return nil, nil, false, fmt.Errorf("decode credentials: %w", err)
	}
	return config, secrets, row.integration.UsePlatformDefault, nil
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

// LogAudit records a settings change in the tenant's audit log.
func (s *Service) LogAudit(ctx context.Context, tenantID uuid.UUID, actor *uuid.UUID, action, entityType string, details map[string]any) error {
	raw, err := json.Marshal(details)
	if err != nil {
		raw = []byte("{}")
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO audit_logs (tenant_id, actor_staff_id, action, entity_type, details)
		VALUES ($1, $2, $3, $4, $5)
	`, tenantID, actor, action, entityType, raw); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}
