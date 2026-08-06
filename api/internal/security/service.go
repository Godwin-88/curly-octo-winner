package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles security & compliance operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a new security service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// --- RBAC: Permissions ---

// ListPermissions returns all permissions in the catalog.
func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, code, description, category, created_at
		FROM permissions
		ORDER BY category, code
	`)
	if err != nil {
		return nil, fmt.Errorf("query permissions: %w", err)
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Description, &p.Category, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

// ListRoles returns all distinct roles with their permission counts.
func (s *Service) ListRoles(ctx context.Context) ([]RolePermissionsResponse, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT role FROM role_permissions ORDER BY role
	`)
	if err != nil {
		return nil, fmt.Errorf("query roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]RolePermissionsResponse, 0, len(roles))
	for _, role := range roles {
		perms, err := s.GetRolePermissions(ctx, role)
		if err != nil {
			return nil, err
		}
		result = append(result, RolePermissionsResponse{Role: role, Permissions: perms})
	}
	return result, nil
}

// GetRolePermissions returns the permission matrix for a role.
func (s *Service) GetRolePermissions(ctx context.Context, role string) ([]Permission, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id, p.code, p.description, p.category, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_code = p.code
		WHERE rp.role = $1
		ORDER BY p.category, p.code
	`, role)
	if err != nil {
		return nil, fmt.Errorf("query role permissions: %w", err)
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Description, &p.Category, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan role permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

// GrantRolePermission grants a permission code to a role.
func (s *Service) GrantRolePermission(ctx context.Context, role, permissionCode string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO role_permissions (role, permission_code)
		VALUES ($1, $2)
		ON CONFLICT (role, permission_code) DO NOTHING
	`, role, permissionCode)
	if err != nil {
		return fmt.Errorf("grant role permission: %w", err)
	}
	return nil
}

// RevokeRolePermission removes a permission code from a role.
func (s *Service) RevokeRolePermission(ctx context.Context, role, permissionCode string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM role_permissions
		WHERE role = $1 AND permission_code = $2
	`, role, permissionCode)
	if err != nil {
		return fmt.Errorf("revoke role permission: %w", err)
	}
	return nil
}

// --- Refresh Tokens (Sessions) ---

// HashToken returns the SHA-256 hex digest of a refresh token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// StoreRefreshToken persists a new refresh token (session).
func (s *Service) StoreRefreshToken(ctx context.Context, tenantID, staffID uuid.UUID, tokenHash string, expiresAt time.Time, ip, userAgent *string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (tenant_id, staff_id, token_hash, expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tenantID, staffID, tokenHash, expiresAt, ip, userAgent)
	if err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

// GetRefreshToken looks up a session by token hash.
func (s *Service) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	var t RefreshToken
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, staff_id, expires_at, revoked_at, replaced_by,
		       ip_address::text, user_agent, created_at, last_used_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(
		&t.ID, &t.TenantID, &t.StaffID, &t.ExpiresAt, &t.RevokedAt, &t.ReplacedBy,
		&t.IPAddress, &t.UserAgent, &t.CreatedAt, &t.LastUsedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("refresh token not found: %w", err)
		}
		return nil, fmt.Errorf("query refresh token: %w", err)
	}
	return &t, nil
}

// RevokeRefreshToken revokes a session (logout / rotation).
func (s *Service) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// RotateRefreshToken revokes the old token and records the replacement.
func (s *Service) RotateRefreshToken(ctx context.Context, oldHash, newHash string, newExpiresAt time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rotate tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var oldID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL
		RETURNING id
	`, oldHash).Scan(&oldID)
	if err != nil {
		return fmt.Errorf("revoke old refresh token: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (tenant_id, staff_id, token_hash, expires_at, replaced_by)
		SELECT tenant_id, staff_id, $2, $3, $1
		FROM refresh_tokens WHERE id = $1
	`, oldID, newHash, newExpiresAt)
	if err != nil {
		return fmt.Errorf("insert rotated refresh token: %w", err)
	}

	return tx.Commit(ctx)
}

// ListActiveSessions lists non-revoked, non-expired sessions for a staff member.
func (s *Service) ListActiveSessions(ctx context.Context, tenantID, staffID uuid.UUID) ([]RefreshToken, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, staff_id, expires_at, revoked_at, replaced_by,
		       ip_address::text, user_agent, created_at, last_used_at
		FROM refresh_tokens
		WHERE tenant_id = $1 AND staff_id = $2
		  AND revoked_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC
	`, tenantID, staffID)
	if err != nil {
		return nil, fmt.Errorf("query active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []RefreshToken
	for rows.Next() {
		var t RefreshToken
		if err := rows.Scan(&t.ID, &t.TenantID, &t.StaffID, &t.ExpiresAt, &t.RevokedAt, &t.ReplacedBy,
			&t.IPAddress, &t.UserAgent, &t.CreatedAt, &t.LastUsedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, t)
	}
	return sessions, rows.Err()
}

// --- Audit Log ---

// LogAudit writes an audit log entry.
func (s *Service) LogAudit(ctx context.Context, tenantID uuid.UUID, actorStaffID *uuid.UUID, action, entityType string, entityID *uuid.UUID, details map[string]any, ip, userAgent *string) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		detailsJSON = []byte("{}")
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO audit_logs (tenant_id, actor_staff_id, action, entity_type, entity_id, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, tenantID, actorStaffID, action, entityType, entityID, detailsJSON, ip, userAgent)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

// ListAuditLogs returns audit log entries for a tenant with optional filters.
func (s *Service) ListAuditLogs(ctx context.Context, tenantID uuid.UUID, entityType, action string, limit, offset int) ([]AuditLog, error) {
	query := `
		SELECT id, tenant_id, actor_staff_id, action, entity_type, entity_id, details,
		       ip_address::text, user_agent, created_at
		FROM audit_logs
		WHERE tenant_id = $1
	`
	args := []any{tenantID}
	argIdx := 2

	if entityType != "" {
		query += fmt.Sprintf(" AND entity_type = $%d", argIdx)
		args = append(args, entityType)
		argIdx++
	}
	if action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, action)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(&l.ID, &l.TenantID, &l.ActorStaffID, &l.Action, &l.EntityType, &l.EntityID,
			&l.Details, &l.IPAddress, &l.UserAgent, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// --- Data Processing Register (KDPA) ---

const dprColumns = `id, tenant_id, activity, purpose, legal_basis, data_subjects,
	categories_of_data, retention_period, transfer_to_third_parties, third_parties,
	security_measures, registered_by, created_at, updated_at`

func scanDPR(row pgx.Row) (*DataProcessingRecord, error) {
	var r DataProcessingRecord
	err := row.Scan(&r.ID, &r.TenantID, &r.Activity, &r.Purpose, &r.LegalBasis, &r.DataSubjects,
		&r.CategoriesOfData, &r.RetentionPeriod, &r.TransferToThirdParties, &r.ThirdParties,
		&r.SecurityMeasures, &r.RegisteredBy, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ListProcessingRecords returns all data processing register entries for a tenant.
func (s *Service) ListProcessingRecords(ctx context.Context, tenantID uuid.UUID) ([]DataProcessingRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+dprColumns+`
		FROM data_processing_register
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query processing records: %w", err)
	}
	defer rows.Close()

	var records []DataProcessingRecord
	for rows.Next() {
		r, err := scanDPR(rows)
		if err != nil {
			return nil, fmt.Errorf("scan processing record: %w", err)
		}
		records = append(records, *r)
	}
	return records, rows.Err()
}

// GetProcessingRecord returns a single processing register entry.
func (s *Service) GetProcessingRecord(ctx context.Context, tenantID, id uuid.UUID) (*DataProcessingRecord, error) {
	r, err := scanDPR(s.pool.QueryRow(ctx, `
		SELECT `+dprColumns+`
		FROM data_processing_register
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("processing record not found: %w", err)
		}
		return nil, fmt.Errorf("query processing record: %w", err)
	}
	return r, nil
}

// CreateProcessingRecord inserts a new data processing register entry.
func (s *Service) CreateProcessingRecord(ctx context.Context, tenantID uuid.UUID, input CreateProcessingRecordInput, registeredBy *uuid.UUID) (*DataProcessingRecord, error) {
	r, err := scanDPR(s.pool.QueryRow(ctx, `
		INSERT INTO data_processing_register (
			tenant_id, activity, purpose, legal_basis, data_subjects, categories_of_data,
			retention_period, transfer_to_third_parties, third_parties, security_measures, registered_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING `+dprColumns,
		tenantID, input.Activity, input.Purpose, input.LegalBasis, input.DataSubjects,
		input.CategoriesOfData, input.RetentionPeriod, input.TransferToThirdParties,
		input.ThirdParties, input.SecurityMeasures, registeredBy))
	if err != nil {
		return nil, fmt.Errorf("insert processing record: %w", err)
	}
	return r, nil
}

// UpdateProcessingRecord updates a data processing register entry.
func (s *Service) UpdateProcessingRecord(ctx context.Context, tenantID, id uuid.UUID, input UpdateProcessingRecordInput) (*DataProcessingRecord, error) {
	r, err := scanDPR(s.pool.QueryRow(ctx, `
		UPDATE data_processing_register
		SET activity = COALESCE($3, activity),
		    purpose = COALESCE($4, purpose),
		    legal_basis = COALESCE($5, legal_basis),
		    data_subjects = COALESCE($6, data_subjects),
		    categories_of_data = COALESCE($7, categories_of_data),
		    retention_period = COALESCE($8, retention_period),
		    transfer_to_third_parties = COALESCE($9, transfer_to_third_parties),
		    third_parties = COALESCE($10, third_parties),
		    security_measures = COALESCE($11, security_measures),
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2
		RETURNING `+dprColumns,
		tenantID, id, input.Activity, input.Purpose, input.LegalBasis, input.DataSubjects,
		input.CategoriesOfData, input.RetentionPeriod, input.TransferToThirdParties,
		input.ThirdParties, input.SecurityMeasures))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("processing record not found: %w", err)
		}
		return nil, fmt.Errorf("update processing record: %w", err)
	}
	return r, nil
}

// DeleteProcessingRecord removes a data processing register entry.
func (s *Service) DeleteProcessingRecord(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM data_processing_register
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete processing record: %w", err)
	}
	return nil
}

// --- Consent Management ---

const consentColumns = `id, tenant_id, guardian_id, consent_type::text, granted, granted_at,
	revoked_at, ip_address::text, source, consent_version, created_at`

func scanConsent(row pgx.Row) (*ConsentAgreement, error) {
	var c ConsentAgreement
	err := row.Scan(&c.ID, &c.TenantID, &c.GuardianID, &c.ConsentType, &c.Granted, &c.GrantedAt,
		&c.RevokedAt, &c.IPAddress, &c.Source, &c.ConsentVersion, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListConsentAgreements returns consent records for a tenant, optionally filtered by guardian.
func (s *Service) ListConsentAgreements(ctx context.Context, tenantID uuid.UUID, guardianID *uuid.UUID) ([]ConsentAgreement, error) {
	query := `
		SELECT ` + consentColumns + `
		FROM consent_agreements
		WHERE tenant_id = $1
	`
	args := []any{tenantID}
	if guardianID != nil {
		query += " AND guardian_id = $2"
		args = append(args, *guardianID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query consent agreements: %w", err)
	}
	defer rows.Close()

	var consents []ConsentAgreement
	for rows.Next() {
		c, err := scanConsent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan consent agreement: %w", err)
		}
		consents = append(consents, *c)
	}
	return consents, rows.Err()
}

// GrantConsent records explicit consent for a guardian (KDPA).
func (s *Service) GrantConsent(ctx context.Context, tenantID uuid.UUID, input GrantConsentInput) (*ConsentAgreement, error) {
	c, err := scanConsent(s.pool.QueryRow(ctx, `
		INSERT INTO consent_agreements (tenant_id, guardian_id, consent_type, granted, granted_at, ip_address, source, consent_version)
		VALUES ($1, $2, $3, true, now(), $4, $5, $6)
		ON CONFLICT (guardian_id, consent_type)
		DO UPDATE SET granted = true, granted_at = now(), revoked_at = NULL,
		              ip_address = EXCLUDED.ip_address, source = EXCLUDED.source,
		              consent_version = EXCLUDED.consent_version
		RETURNING `+consentColumns,
		tenantID, input.GuardianID, input.ConsentType, input.IPAddress, input.Source, input.ConsentVersion))
	if err != nil {
		return nil, fmt.Errorf("grant consent: %w", err)
	}
	return c, nil
}

// RevokeConsent revokes a guardian's consent (KDPA right to withdraw).
func (s *Service) RevokeConsent(ctx context.Context, tenantID uuid.UUID, guardianID uuid.UUID, consentType string) (*ConsentAgreement, error) {
	c, err := scanConsent(s.pool.QueryRow(ctx, `
		UPDATE consent_agreements
		SET granted = false, revoked_at = now()
		WHERE tenant_id = $1 AND guardian_id = $2 AND consent_type = $3
		RETURNING `+consentColumns,
		tenantID, guardianID, consentType))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("consent agreement not found: %w", err)
		}
		return nil, fmt.Errorf("revoke consent: %w", err)
	}
	return c, nil
}

// --- Erasure Requests (Right to be Forgotten) ---

const erasureColumns = `id, tenant_id, subject_type, subject_id, requested_by, request_type,
	status, details, completed_at, created_at`

func scanErasure(row pgx.Row) (*ErasureRequest, error) {
	var e ErasureRequest
	err := row.Scan(&e.ID, &e.TenantID, &e.SubjectType, &e.SubjectID, &e.RequestedBy, &e.RequestType,
		&e.Status, &e.Details, &e.CompletedAt, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ListErasureRequests returns data subject rights requests for a tenant.
func (s *Service) ListErasureRequests(ctx context.Context, tenantID uuid.UUID, status string) ([]ErasureRequest, error) {
	query := `
		SELECT ` + erasureColumns + `
		FROM erasure_requests
		WHERE tenant_id = $1
	`
	args := []any{tenantID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query erasure requests: %w", err)
	}
	defer rows.Close()

	var requests []ErasureRequest
	for rows.Next() {
		e, err := scanErasure(rows)
		if err != nil {
			return nil, fmt.Errorf("scan erasure request: %w", err)
		}
		requests = append(requests, *e)
	}
	return requests, rows.Err()
}

// CreateErasureRequest records a new data subject rights request.
func (s *Service) CreateErasureRequest(ctx context.Context, tenantID uuid.UUID, input CreateErasureRequestInput) (*ErasureRequest, error) {
	e, err := scanErasure(s.pool.QueryRow(ctx, `
		INSERT INTO erasure_requests (tenant_id, subject_type, subject_id, requested_by, request_type, details)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+erasureColumns,
		tenantID, input.SubjectType, input.SubjectID, input.RequestedBy, input.RequestType, input.Details))
	if err != nil {
		return nil, fmt.Errorf("insert erasure request: %w", err)
	}
	return e, nil
}

// UpdateErasureRequestStatus updates the status of a data subject rights request.
func (s *Service) UpdateErasureRequestStatus(ctx context.Context, tenantID, id uuid.UUID, status string) (*ErasureRequest, error) {
	e, err := scanErasure(s.pool.QueryRow(ctx, `
		UPDATE erasure_requests
		SET status = $3,
		    completed_at = CASE WHEN $3 = 'completed' THEN now() ELSE completed_at END
		WHERE tenant_id = $1 AND id = $2
		RETURNING `+erasureColumns,
		tenantID, id, status))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("erasure request not found: %w", err)
		}
		return nil, fmt.Errorf("update erasure request: %w", err)
	}
	return e, nil
}

// --- Summary ---

// GetSummary returns aggregate security/compliance metrics for the dashboard.
func (s *Service) GetSummary(ctx context.Context, tenantID uuid.UUID) (*SecuritySummary, error) {
	var sum SecuritySummary
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM staff WHERE tenant_id = $1),
			(SELECT COUNT(*) FROM refresh_tokens WHERE tenant_id = $1 AND revoked_at IS NULL AND expires_at > now()),
			(SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1 AND created_at > now() - interval '24 hours'),
			(SELECT COUNT(*) FROM consent_agreements WHERE tenant_id = $1 AND granted = true),
			(SELECT COUNT(*) FROM consent_agreements WHERE tenant_id = $1 AND granted = false),
			(SELECT COUNT(*) FROM erasure_requests WHERE tenant_id = $1 AND status = 'pending'),
			(SELECT COUNT(*) FROM data_processing_register WHERE tenant_id = $1),
			(SELECT COUNT(*) FROM permissions)
	`, tenantID).Scan(
		&sum.TotalStaff, &sum.ActiveSessions, &sum.AuditEvents24h,
		&sum.ConsentGranted, &sum.ConsentRevoked, &sum.PendingErasure,
		&sum.ProcessingRecords, &sum.PermissionCount,
	)
	if err != nil {
		return nil, fmt.Errorf("query security summary: %w", err)
	}
	return &sum, nil
}
