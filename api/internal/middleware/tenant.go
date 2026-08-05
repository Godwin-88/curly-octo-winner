package middleware

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/shule360/api/pkg/httputil"
)

// TenantRequired verifies that a tenant_id is present in the request context.
// This should be used after Auth middleware.
func TenantRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := r.Context().Value(ContextKeyTenantID).(uuid.UUID)
		if !ok || tenantID == uuid.Nil {
			httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found in context")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetTenantID extracts the tenant ID from the request context.
func GetTenantID(r *http.Request) (uuid.UUID, bool) {
	id, ok := r.Context().Value(ContextKeyTenantID).(uuid.UUID)
	return id, ok
}

// GetStaffID extracts the staff ID from the request context.
func GetStaffID(r *http.Request) (uuid.UUID, bool) {
	id, ok := r.Context().Value(ContextKeyStaffID).(uuid.UUID)
	return id, ok
}

// GetStaffRole extracts the staff role from the request context.
func GetStaffRole(r *http.Request) (string, bool) {
	role, ok := r.Context().Value(ContextKeyStaffRole).(string)
	return role, ok
}
