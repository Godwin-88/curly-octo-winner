package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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
)

// Claims represents the JWT claims structure for Shule360.
type Claims struct {
	TenantID string `json:"tenant_id"`
	StaffID  string `json:"staff_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Auth validates the JWT and extracts tenant_id, staff_id, and role into context.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httputil.RespondUnauthorized(w, "UNAUTHORIZED", "missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Invalid Authorization header format")
				return
			}

			tokenStr := parts[1]
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

			staffID, err := uuid.Parse(claims.StaffID)
			if err != nil {
				httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Invalid staff_id in token")
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyTenantID, tenantID)
			ctx = context.WithValue(ctx, ContextKeyStaffID, staffID)
			ctx = context.WithValue(ctx, ContextKeyStaffRole, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns a middleware that restricts access to specific roles.
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
