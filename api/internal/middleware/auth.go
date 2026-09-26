package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/pkg/httputil"
)

type contextKey string

const (
	// ContextKeyTenantID is the context key for the tenant ID.
	ContextKeyTenantID contextKey = "tenant_id"
	// ContextKeyStaffID is the context key for the staff user ID.
	ContextKeyStaffID contextKey = "staff_id"
	// ContextKeyStaffRole is the context key for the staff role.
	ContextKeyStaffRole contextKey = "staff_role"
	// ContextKeyGuardianID is the context key for the guardian user ID
	// (present only when the bearer token is a guardian token).
	ContextKeyGuardianID contextKey = "guardian_id"
)

// Claims represents the JWT claims structure for Shule360.
//
// Staff tokens (issued by POST /api/v1/login) carry staff_id. Guardian tokens
// (issued by POST /api/v1/auth/guardian/login) carry guardian_id instead, with
// role set to "guardian".
type Claims struct {
	TenantID   string `json:"tenant_id"`
	StaffID    string `json:"staff_id"`
	GuardianID string `json:"guardian_id"`
	Role       string `json:"role"`
	jwt.RegisteredClaims
}

// Auth validates the session JWT and extracts tenant_id, staff_id, and role
// into context. The token may arrive as an Authorization: Bearer header or as
// an HttpOnly session cookie (see TokenFromRequest).
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := TokenFromRequest(r)
			if tokenStr == "" {
				httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Missing credentials")
				return
			}

			claims := &Claims{}

			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Invalid or expired token")
				return
			}

			// Validate tenant_id is a valid UUID
			tenantID, err := uuid.Parse(claims.TenantID)
			if err != nil {
				httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Invalid tenant_id in token")
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyTenantID, tenantID)

			if claims.Role == "guardian" {
				// Guardian token: requires guardian_id claim.
				guardianID, gerr := uuid.Parse(claims.GuardianID)
				if gerr != nil || guardianID == uuid.Nil {
					httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Invalid guardian_id in token")
					return
				}
				ctx = context.WithValue(ctx, ContextKeyGuardianID, guardianID)
				ctx = context.WithValue(ctx, ContextKeyStaffRole, "guardian")
				r = r.WithContext(ctx)
				setIdentityHeaders(r)
				next.ServeHTTP(w, r)
				return
			}

			// Staff token: requires staff_id claim.
			staffID, err := uuid.Parse(claims.StaffID)
			if err != nil {
				httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Invalid staff_id in token")
				return
			}

			ctx = context.WithValue(ctx, ContextKeyStaffID, staffID)
			ctx = context.WithValue(ctx, ContextKeyStaffRole, claims.Role)
			r = r.WithContext(ctx)
			setIdentityHeaders(r)
			next.ServeHTTP(w, r)
		})
	}
}

// setIdentityHeaders copies the authenticated identity into request headers so
// handlers that read X-Staff-ID / X-Guardian-ID / X-Tenant-ID (parent and
// teacher packages) keep working unchanged. Values always come from the
// *verified* JWT, never from client-supplied headers, so they cannot be
// spoofed.
func setIdentityHeaders(r *http.Request) {
	if staffID, ok := GetStaffID(r); ok {
		r.Header.Set("X-Staff-ID", staffID.String())
	}
	if guardianID, ok := GetGuardianID(r); ok {
		r.Header.Set("X-Guardian-ID", guardianID.String())
	}
	if tenantID, ok := GetTenantID(r); ok {
		r.Header.Set("X-Tenant-ID", tenantID.String())
	}
}

// GetGuardianID extracts the guardian ID from the request context.
func GetGuardianID(r *http.Request) (uuid.UUID, bool) {
	id, ok := r.Context().Value(ContextKeyGuardianID).(uuid.UUID)
	return id, ok
}

// RequireRole returns a middleware that restricts access to specific roles.
// Staff roles come from the JWT "role" claim; guardian tokens always carry
// role "guardian", so listing "guardian" in allowedRoles admits guardian
// tokens to otherwise staff-only route groups.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool)
	for _, role := range allowedRoles {
		allowed[role] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(ContextKeyStaffRole).(string)
			if !ok || !allowed[role] {
				httputil.RespondForbidden(w, "FORBIDDEN", "Insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
