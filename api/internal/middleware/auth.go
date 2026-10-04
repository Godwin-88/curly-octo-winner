package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
	// ContextKeySession is the context key for the Session (who is asking and
	// how far they may reach).
	ContextKeySession contextKey = "session"
)

// Session scopes. A token with no scope claim is a school session: that is
// every token issued before scopes existed.
const (
	ScopeSchool   = "school"
	ScopeGroup    = "group"
	ScopePlatform = "platform"
)

// SchoolHeader names the school a group or platform session wants to work in.
// It is a request, not a grant: Auth narrows it to what the session allows.
// A school session ignores it.
const SchoolHeader = "X-School-ID"

// Session describes the signed-in user's reach.
type Session struct {
	Scope      string
	GroupID    uuid.UUID // ScopeGroup only
	OperatorID uuid.UUID // platform_users.id; zero for school staff
}

// SchoolLookup answers which group a school belongs to. found is false when
// the school does not exist.
type SchoolLookup interface {
	SchoolGroup(ctx context.Context, schoolID uuid.UUID) (groupID *uuid.UUID, found bool, err error)
	// OperatorStaff returns the staff row that stands for a platform or group
	// user inside a school, creating it on first use, and the role they hold
	// now. active is false when the user has been deactivated.
	OperatorStaff(ctx context.Context, schoolID, operatorID uuid.UUID) (staffID uuid.UUID, role string, active bool, err error)
	// StaffAccess returns the role a member of a school's staff holds now.
	// active is false when they have been deactivated or no longer exist.
	StaffAccess(ctx context.Context, schoolID, staffID uuid.UUID) (role string, active bool, err error)
}

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
	// Scope, GroupID and OperatorID are set for platform and group users
	// (platform_users). They are absent on school staff and guardian tokens.
	Scope      string `json:"scope,omitempty"`
	GroupID    string `json:"group_id,omitempty"`
	OperatorID string `json:"operator_id,omitempty"`
	jwt.RegisteredClaims
}

// Auth validates the session JWT and extracts tenant_id, staff_id, and role
// into context. The token may arrive as an Authorization: Bearer header or as
// an HttpOnly session cookie (see TokenFromRequest).
//
// secure controls the Secure flag on rolling-renewal cookies (true in
// production). Cookie-based sessions are renewed once more than half their
// lifetime has elapsed (sliding window on active use); bearer-token API
// clients manage their own tokens and are never touched.
//
// Which school a request acts on:
//
//   - school staff and guardians: the school in their token. Always.
//   - a group user: the school named in X-School-ID, if it belongs to their
//     group; any other school answers 404, exactly like one that does not exist.
//   - a platform user: the school named in X-School-ID, if it exists.
//
// A group or platform request that names no school carries no tenant;
// TenantRequired answers it with SCHOOL_REQUIRED. schools may be nil, in which
// case group and platform sessions can never select a school.
func Auth(jwtSecret string, secure bool, schools SchoolLookup) func(http.Handler) http.Handler {
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

			maybeRenewSession(w, r, claims, tokenStr, jwtSecret, secure)

			if claims.Scope == ScopeGroup || claims.Scope == ScopePlatform {
				ctx, ok := operatorContext(w, r, claims, schools)
				if !ok {
					return
				}
				r = r.WithContext(ctx)
				setIdentityHeaders(r)
				next.ServeHTTP(w, r)
				return
			}
			if claims.Scope != "" && claims.Scope != ScopeSchool {
				httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Unknown session scope")
				return
			}

			// Validate tenant_id is a valid UUID
			tenantID, err := uuid.Parse(claims.TenantID)
			if err != nil {
				httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Invalid tenant_id in token")
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyTenantID, tenantID)
			ctx = context.WithValue(ctx, ContextKeySession, Session{Scope: ScopeSchool})

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

			// The role and whether the account is still active are read on
			// every request, not taken from the token: a user who is
			// deactivated, or whose role is changed, is affected at once
			// rather than when their session next expires.
			role := claims.Role
			if schools != nil {
				current, active, lookupErr := schools.StaffAccess(r.Context(), tenantID, staffID)
				if lookupErr != nil {
					httputil.RespondInternalError(w, lookupErr)
					return
				}
				if !active {
					httputil.RespondUnauthorized(w, "ACCOUNT_DEACTIVATED", "This account can no longer sign in. Ask your school's principal.")
					return
				}
				role = current
			}

			ctx = context.WithValue(ctx, ContextKeyStaffID, staffID)
			ctx = context.WithValue(ctx, ContextKeyStaffRole, role)
			r = r.WithContext(ctx)
			setIdentityHeaders(r)
			next.ServeHTTP(w, r)
		})
	}
}

// operatorContext builds the request context for a group or platform session,
// resolving the school it asked for. It writes the response and returns false
// when the request must stop.
func operatorContext(w http.ResponseWriter, r *http.Request, claims *Claims, schools SchoolLookup) (context.Context, bool) {
	operatorID, err := uuid.Parse(claims.OperatorID)
	if err != nil || operatorID == uuid.Nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Invalid operator_id in token")
		return nil, false
	}
	session := Session{Scope: claims.Scope, OperatorID: operatorID}
	if claims.Scope == ScopeGroup {
		groupID, err := uuid.Parse(claims.GroupID)
		if err != nil || groupID == uuid.Nil {
			httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Invalid group_id in token")
			return nil, false
		}
		session.GroupID = groupID
	}

	ctx := context.WithValue(r.Context(), ContextKeySession, session)
	ctx = context.WithValue(ctx, ContextKeyStaffRole, claims.Role)

	asked := r.Header.Get(SchoolHeader)
	if asked == "" {
		return ctx, true // portfolio request: no school chosen
	}
	schoolID, err := uuid.Parse(asked)
	if err != nil || schoolID == uuid.Nil {
		httputil.RespondBadRequest(w, "INVALID_SCHOOL", SchoolHeader+" is not a valid school id")
		return nil, false
	}
	if schools == nil {
		httputil.RespondNotFound(w, "SCHOOL_NOT_FOUND", "School not found")
		return nil, false
	}
	groupID, found, err := schools.SchoolGroup(r.Context(), schoolID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return nil, false
	}
	// Outside the session's reach is reported exactly like "does not exist".
	if !found || (claims.Scope == ScopeGroup && (groupID == nil || *groupID != session.GroupID)) {
		httputil.RespondNotFound(w, "SCHOOL_NOT_FOUND", "School not found")
		return nil, false
	}

	// Inside a school the operator acts as a staff row of their own, so every
	// screen that records "who did this" works for them too. The role is read
	// now, not from the token: a demotion or deactivation takes effect at once.
	staffID, role, active, err := schools.OperatorStaff(r.Context(), schoolID, operatorID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return nil, false
	}
	if !active {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "This account has been deactivated")
		return nil, false
	}
	ctx = context.WithValue(ctx, ContextKeyStaffID, staffID)
	ctx = context.WithValue(ctx, ContextKeyStaffRole, role)
	return context.WithValue(ctx, ContextKeyTenantID, schoolID), true
}

// RequirePlatform admits only platform-scope sessions: the people who manage
// schools, groups and the users above a school.
func RequirePlatform(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetSession(r).Scope != ScopePlatform {
			httputil.RespondForbidden(w, "FORBIDDEN", "Only platform administrators can do this")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetSession returns who is asking and how far they may reach.
func GetSession(r *http.Request) Session {
	if s, ok := r.Context().Value(ContextKeySession).(Session); ok {
		return s
	}
	return Session{Scope: ScopeSchool}
}

// setIdentityHeaders copies the authenticated identity into request headers so
// handlers that read X-Staff-ID / X-Guardian-ID / X-Tenant-ID (parent and
// teacher packages) keep working unchanged. Values always come from the
// *verified* JWT, never from client-supplied headers, so they cannot be
// spoofed: whatever the client sent under these names is removed first, which
// matters for a session that carries no staff id or no school.
func setIdentityHeaders(r *http.Request) {
	r.Header.Del("X-Staff-ID")
	r.Header.Del("X-Guardian-ID")
	r.Header.Del("X-Tenant-ID")
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

// maybeRenewSession slides the session forward: once more than half the 24h
// TTL has elapsed, a request that authenticated via cookie gets a fresh cookie
// with a new 24h expiry (same claims, re-signed). Sessions thus stay alive
// with regular use and hard-stop after 24h of complete inactivity — the
// stolen-cookie window stays bounded while active users are never logged out
// mid-task. Bearer-token API clients are never renewed.
func maybeRenewSession(w http.ResponseWriter, r *http.Request, claims *Claims, currentToken, jwtSecret string, secure bool) {
	if r.Header.Get("Authorization") != "" {
		return
	}
	cookieName := CookieStaffSession
	if claims.Role == "guardian" {
		cookieName = CookieGuardianSession
	}
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" || c.Value != currentToken {
		return // not a cookie-authenticated request (or token identity mismatch)
	}
	if claims.ExpiresAt == nil {
		return
	}
	remaining := time.Until(claims.ExpiresAt.Time)
	if remaining > SessionTTL/2 {
		return // still fresh
	}

	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(SessionTTL))
	claims.IssuedAt = jwt.NewNumericDate(time.Now())
	claims.NotBefore = jwt.NewNumericDate(time.Now())
	renewed := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := renewed.SignedString([]byte(jwtSecret))
	if err != nil {
		return // renewal is best-effort; the current session remains valid
	}
	SetSessionCookie(w, cookieName, signed, secure)
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

// ModuleLookup answers whether a school has a module.
type ModuleLookup interface {
	ModuleEnabled(ctx context.Context, schoolID uuid.UUID, module string) (bool, error)
}

// RequireModule refuses a request for a module the school in context does not
// have. It runs after Auth, which decides the school. The answer is read on
// every request, so switching a module off takes effect at once.
func RequireModule(lookup ModuleLookup, module, label string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, ok := r.Context().Value(ContextKeyTenantID).(uuid.UUID)
			if !ok || tenantID == uuid.Nil {
				// No school chosen: the tenant middleware answers that.
				next.ServeHTTP(w, r)
				return
			}
			enabled, err := lookup.ModuleEnabled(r.Context(), tenantID, module)
			if err != nil {
				httputil.RespondInternalError(w, err)
				return
			}
			if !enabled {
				httputil.RespondForbidden(w, "MODULE_NOT_ENABLED", label+" is not part of this school's plan.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
