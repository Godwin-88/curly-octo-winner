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
	"github.com/shule360/api/pkg/httputil"
	supabaseclient "github.com/shule360/api/pkg/supabase"
)

// LoginRequest represents the login request body.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response body.
type LoginResponse struct {
	Token string      `json:"token"`
	Staff StaffBrief  `json:"staff"`
}

// StaffBrief is a minimal staff representation returned on login.
type StaffBrief struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Phone     string `json:"phone,omitempty"`
}

// Handler handles authentication endpoints.
type Handler struct {
	supabase *supabaseclient.Client
	cfg      *config.Config
}

// NewHandler creates a new auth handler.
func NewHandler(supabase *supabaseclient.Client, cfg *config.Config) *Handler {
	return &Handler{supabase: supabase, cfg: cfg}
}

// Mount registers auth routes.
func (h *Handler) Mount(r chi.Router) {
	r.Post("/login", h.Login)
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

	httputil.RespondOK(w, LoginResponse{
		Token: token,
		Staff: StaffBrief{
			ID:        staff.ID,
			TenantID:  staff.TenantID,
			FullName:  staff.FullName,
			Email:     staff.Email,
			Role:      staff.Role,
			Phone:     staff.Phone,
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
