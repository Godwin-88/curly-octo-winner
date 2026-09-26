package middleware

import (
	"net/http"
	"strings"
	"time"
)

// Session cookie names. The Go API issues the session JWT both in the JSON
// response body (for API clients that use Authorization headers) and in an
// HttpOnly cookie. The web app proxies /api/v1/* to this API same-origin
// (web/next.config.js rewrites), so these cookies are first-party for the
// frontend domain and invisible to JavaScript — an XSS payload can no longer
// exfiltrate a valid session token.
const (
	CookieStaffSession    = "shule360_session"
	CookieGuardianSession = "shule360_guardian_session"
)

// sessionCookieTTL matches the 24h expiry stamped into the JWT itself.
const sessionCookieTTL = 24 * time.Hour

// SessionTTL exposes the session lifetime for other packages (e.g. the
// rolling-renewal logic in the auth middleware).
const SessionTTL = sessionCookieTTL

// SetSessionCookie writes an HttpOnly, SameSite=Lax session cookie.
// secure should be true in production (HTTPS-only); it is disabled for
// development so the cookie also works over plain-http localhost.
func SetSessionCookie(w http.ResponseWriter, name, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionCookieTTL.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie expires a session cookie. MaxAge -1 tells the browser to
// drop the cookie immediately.
func ClearSessionCookie(w http.ResponseWriter, name string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// TokenFromRequest extracts the session JWT from a request, trying in order:
//
//  1. Authorization: Bearer <token>  (API clients, teacher PWA direct mode)
//  2. shule360_session cookie        (staff web session)
//  3. shule360_guardian_session cookie (guardian web session)
//
// The token type (staff vs guardian) is determined by its claims, not by the
// channel it arrived on, so a cookie carried token and a bearer token are
// validated identically.
func TokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		parts := strings.SplitN(h, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	for _, name := range []string{CookieStaffSession, CookieGuardianSession} {
		if c, err := r.Cookie(name); err == nil && c.Value != "" {
			return c.Value
		}
	}
	return ""
}
