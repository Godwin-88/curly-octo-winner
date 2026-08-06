package security

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Permission represents a single permission in the RBAC catalog.
type Permission struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Description *string   `json:"description,omitempty"`
	Category    *string   `json:"category,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// RolePermission maps a role to a permission code.
type RolePermission struct {
	ID             uuid.UUID `json:"id"`
	Role           string    `json:"role"`
	PermissionCode string    `json:"permission_code"`
	CreatedAt      time.Time `json:"created_at"`
}

// RolePermissionsResponse returns the full permission matrix for a role.
type RolePermissionsResponse struct {
	Role        string       `json:"role"`
	Permissions []Permission `json:"permissions"`
}

// RefreshToken represents a stored JWT refresh token (session).
type RefreshToken struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	StaffID    uuid.UUID  `json:"staff_id"`
	ExpiresAt  time.Time  `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	ReplacedBy *uuid.UUID `json:"replaced_by,omitempty"`
	IPAddress  *string    `json:"ip_address,omitempty"`
	UserAgent  *string    `json:"user_agent,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// AuditLog represents a single audit log entry.
type AuditLog struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	ActorStaffID *uuid.UUID      `json:"actor_staff_id,omitempty"`
	Action       string          `json:"action"`
	EntityType   string          `json:"entity_type"`
	EntityID     *uuid.UUID      `json:"entity_id,omitempty"`
	Details      json.RawMessage `json:"details,omitempty"`
	IPAddress    *string         `json:"ip_address,omitempty"`
	UserAgent    *string         `json:"user_agent,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// DataProcessingRecord is a Kenya Data Protection Act processing register entry.
type DataProcessingRecord struct {
	ID                     uuid.UUID  `json:"id"`
	TenantID               uuid.UUID  `json:"tenant_id"`
	Activity               string     `json:"activity"`
	Purpose                string     `json:"purpose"`
	LegalBasis             string     `json:"legal_basis"`
	DataSubjects           string     `json:"data_subjects"`
	CategoriesOfData       *string    `json:"categories_of_data,omitempty"`
	RetentionPeriod        *string    `json:"retention_period,omitempty"`
	TransferToThirdParties bool       `json:"transfer_to_third_parties"`
	ThirdParties           *string    `json:"third_parties,omitempty"`
	SecurityMeasures       *string    `json:"security_measures,omitempty"`
	RegisteredBy           *uuid.UUID `json:"registered_by,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// ConsentAgreement tracks a guardian's consent for a specific purpose.
type ConsentAgreement struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	GuardianID     *uuid.UUID `json:"guardian_id,omitempty"`
	ConsentType    string     `json:"consent_type"`
	Granted        bool       `json:"granted"`
	GrantedAt      *time.Time `json:"granted_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	IPAddress      *string    `json:"ip_address,omitempty"`
	Source         *string    `json:"source,omitempty"`
	ConsentVersion *string    `json:"consent_version,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ErasureRequest is a data subject rights request (KDPA).
type ErasureRequest struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	SubjectType string     `json:"subject_type"`
	SubjectID   uuid.UUID  `json:"subject_id"`
	RequestedBy string     `json:"requested_by"`
	RequestType string     `json:"request_type"`
	Status      string     `json:"status"`
	Details     *string    `json:"details,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// SecuritySummary aggregates key security/compliance metrics for the dashboard.
type SecuritySummary struct {
	TotalStaff        int64 `json:"total_staff"`
	ActiveSessions    int64 `json:"active_sessions"`
	AuditEvents24h    int64 `json:"audit_events_24h"`
	ConsentGranted    int64 `json:"consent_granted"`
	ConsentRevoked    int64 `json:"consent_revoked"`
	PendingErasure    int64 `json:"pending_erasure"`
	ProcessingRecords int64 `json:"processing_records"`
	PermissionCount   int64 `json:"permission_count"`
}

// CreateProcessingRecordInput is the payload for creating a data processing register entry.
type CreateProcessingRecordInput struct {
	Activity               string  `json:"activity"`
	Purpose                string  `json:"purpose"`
	LegalBasis             string  `json:"legal_basis"`
	DataSubjects           string  `json:"data_subjects"`
	CategoriesOfData       *string `json:"categories_of_data,omitempty"`
	RetentionPeriod        *string `json:"retention_period,omitempty"`
	TransferToThirdParties bool    `json:"transfer_to_third_parties"`
	ThirdParties           *string `json:"third_parties,omitempty"`
	SecurityMeasures       *string `json:"security_measures,omitempty"`
}

// UpdateProcessingRecordInput is the payload for updating a data processing register entry.
type UpdateProcessingRecordInput struct {
	Activity               *string `json:"activity,omitempty"`
	Purpose                *string `json:"purpose,omitempty"`
	LegalBasis             *string `json:"legal_basis,omitempty"`
	DataSubjects           *string `json:"data_subjects,omitempty"`
	CategoriesOfData       *string `json:"categories_of_data,omitempty"`
	RetentionPeriod        *string `json:"retention_period,omitempty"`
	TransferToThirdParties *bool   `json:"transfer_to_third_parties,omitempty"`
	ThirdParties           *string `json:"third_parties,omitempty"`
	SecurityMeasures       *string `json:"security_measures,omitempty"`
}

// GrantConsentInput is the payload for granting a guardian's consent.
type GrantConsentInput struct {
	GuardianID     uuid.UUID `json:"guardian_id"`
	ConsentType    string    `json:"consent_type"`
	IPAddress      *string   `json:"ip_address,omitempty"`
	Source         *string   `json:"source,omitempty"`
	ConsentVersion *string   `json:"consent_version,omitempty"`
}

// CreateErasureRequestInput is the payload for creating a data subject rights request.
type CreateErasureRequestInput struct {
	SubjectType string    `json:"subject_type"`
	SubjectID   uuid.UUID `json:"subject_id"`
	RequestedBy string    `json:"requested_by"`
	RequestType string    `json:"request_type"`
	Details     *string   `json:"details,omitempty"`
}
