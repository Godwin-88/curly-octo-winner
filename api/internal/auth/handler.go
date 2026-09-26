package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	"github.com/shule360/api/internal/config"
	appmiddleware "github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
	supabaseclient "github.com/shule360/api/pkg/supabase"
	"github.com/shule360/api/pkg/upstash"
)

// LoginRequest represents the login request body.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response body.
type LoginResponse struct {
	Token string     `json:"token"`
	Staff StaffBrief `json:"staff"`
}

// StaffBrief is a minimal staff representation returned on login.
type StaffBrief struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Phone    string `json:"phone,omitempty"`
}

// Handler handles authentication endpoints.
type Handler struct {
	supabase *supabaseclient.Client
	cfg      *config.Config
	redis    *upstash.RedisClient
}

// NewHandler creates a new auth handler. redis may be nil, in which case login
// rate limiting fails closed (login becomes unavailable until Redis returns).
func NewHandler(supabase *supabaseclient.Client, cfg *config.Config, redis *upstash.RedisClient) *Handler {
	return &Handler{supabase: supabase, cfg: cfg, redis: redis}
}

// Mount registers public auth routes.
//
// Paths are part of the client contract (web/lib/auth.tsx): POST /api/v1/login
// signs in, POST /api/v1/auth/logout clears the session cookie. Logout is
// deliberately public — clearing an invalid or expired cookie must always
// succeed, so it cannot sit behind the Auth middleware.
func (h *Handler) Mount(r chi.Router) {
	r.Post("/login", h.Login)
	r.Post("/auth/logout", h.Logout)
}

// MountPrivate registers authenticated auth routes. Call this from inside the
// Auth middleware group so /auth/me can trust the verified identity in the
// request context. Matches GET ${API_BASE}/auth/me in web/lib/auth.tsx.
func (h *Handler) MountPrivate(r chi.Router) {
	r.Get("/auth/me", h.Me)
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Email and password are required")
		return
	}

	// Fail-closed brute-force protection: 5 attempts per IP and per account
	// per 15-minute window.
	if !appmiddleware.CheckLoginRateLimit(r.Context(), w, h.redis, appmiddleware.ClientIP(r), req.Email, 900, 5, 5) {
		return
	}

	// Step 1: Verify credentials via Supabase Auth password grant
	supabaseToken, err := h.verifyPassword(r.Context(), req.Email, req.Password)
	if err != nil {
		httputil.RespondUnauthorized(w, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	// Step 2: Look up the staff record by email
	staff, err := h.findStaffByEmail(r.Context(), req.Email)
	if err != nil {
		httputil.RespondUnauthorized(w, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	// Step 3: Generate JWT
	token, err := h.generateToken(staff)
	if err != nil {
		httputil.RespondInternalError(w, fmt.Errorf("failed to generate token: %w", err))
		return
	}

	_ = supabaseToken // Available for future use (e.g., refresh tokens)

	// Issue the session JWT as an HttpOnly cookie in addition to the JSON
	// body. The web app consumes the cookie (same-origin proxy); API clients
	// keep using the bearer token from the body.
	appmiddleware.SetSessionCookie(w, appmiddleware.CookieStaffSession, token, h.cfg.IsProduction())

	httputil.RespondOK(w, LoginResponse{
		Token: token,
		Staff: StaffBrief{
			ID:       staff.ID,
			TenantID: staff.TenantID,
			FullName: staff.FullName,
			Email:    staff.Email,
			Role:     staff.Role,
			Phone:    staff.Phone,
		},
	})
}

// verifyPassword verifies credentials by calling the Supabase Auth password grant endpoint.
// Returns the Supabase access token on success.
func (h *Handler) verifyPassword(ctx context.Context, email, password string) (string, error) {
	url := h.supabase.URL() + "/auth/v1/token?grant_type=password"
	payload, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("apikey", h.supabase.ServiceKey())
	req.Header.Set("Authorization", "Bearer "+h.supabase.ServiceKey())
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.supabase.HTTPClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	return result.AccessToken, nil
}

// staffRow represents a staff record from the database.
type staffRow struct {
	ID       string
	TenantID string
	FullName string
	Email    string
	Role     string
	Phone    string
}

// findStaffByEmail looks up the staff record by email.
func (h *Handler) findStaffByEmail(ctx context.Context, email string) (*staffRow, error) {
	row := h.supabase.Pool.QueryRow(ctx,
		"SELECT id, tenant_id, full_name, email, role::text, COALESCE(phone, '') FROM staff WHERE email = $1 AND is_active = true",
		email,
	)

	var s staffRow
	if err := row.Scan(&s.ID, &s.TenantID, &s.FullName, &s.Email, &s.Role, &s.Phone); err != nil {
		return nil, err
	}
	return &s, nil
}

// generateToken creates a signed JWT for the staff user.
func (h *Handler) generateToken(staff *staffRow) (string, error) {
	claims := jwt.MapClaims{
		"tenant_id": staff.TenantID,
		"staff_id":  staff.ID,
		"role":      staff.Role,
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.cfg.JWTSecret))
}

// ioReadAll reads all data from an io.Reader.
func ioReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// Logout handles POST /api/v1/auth/logout. It is public: clearing an invalid
// or expired session cookie is always safe. The JWT itself cannot be revoked
// server-side (stateless verification), so logout only drops the cookie — the
// 24h expiry bounds any stolen-token window.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	appmiddleware.ClearSessionCookie(w, appmiddleware.CookieStaffSession, h.cfg.IsProduction())
	httputil.RespondOK(w, map[string]string{"status": "logged_out"})
}

// Me handles GET /api/v1/auth/me. Mounted inside the Auth middleware group:
// the staff identity comes from the verified JWT (never from the client) and
// the profile is re-read from the database so role/name changes take effect
// without waiting for token expiry.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	staffID, ok := appmiddleware.GetStaffID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Not a staff session")
		return
	}

	var s staffRow
	err := h.supabase.Pool.QueryRow(r.Context(),
		`SELECT id, tenant_id, full_name, email, role::text, COALESCE(phone, '') FROM staff WHERE id = $1 AND is_active = true`,
		staffID,
	).Scan(&s.ID, &s.TenantID, &s.FullName, &s.Email, &s.Role, &s.Phone)
	if err != nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Staff account not found or deactivated")
		return
	}

	httputil.RespondOK(w, map[string]any{"staff": StaffBrief{
		ID:       s.ID,
		TenantID: s.TenantID,
		FullName: s.FullName,
		Email:    s.Email,
		Role:     s.Role,
		Phone:    s.Phone,
	}})
}
