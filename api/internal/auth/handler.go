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
	"github.com/google/uuid"

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

	// Step 1: Find the Supabase Auth user by email
	supabaseUser, err := h.findSupabaseUser(r.Context(), req.Email)
	if err != nil {
		httputil.RespondUnauthorized(w, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	// Step 2: Verify the password by calling the GoTrue token endpoint
	if !h.verifyPassword(r.Context(), supabaseUser.ID, req.Email, req.Password) {
		httputil.RespondUnauthorized(w, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	// Step 3: Look up the staff record
	staff, err := h.findStaffBySupabaseID(r.Context(), supabaseUser.ID)
	if err != nil {
		httputil.RespondUnauthorized(w, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	// Step 4: Generate JWT
	token, err := h.generateToken(staff)
	if err != nil {
		httputil.RespondInternalError(w, fmt.Errorf("failed to generate token: %w", err))
		return
	}

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

// supabaseUser represents a minimal Supabase Auth user.
type supabaseUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// findSupabaseUser looks up a Supabase Auth user by email via the Admin API.
func (h *Handler) findSupabaseUser(ctx context.Context, email string) (*supabaseUser, error) {
	url := h.supabase.URL() + "/auth/v1/admin/users?email=" + email
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", h.supabase.ServiceKey())
	req.Header.Set("Authorization", "Bearer "+h.supabase.ServiceKey())

	resp, err := h.supabase.HTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("lookup supabase user: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read lookup response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lookup error (status %d): %s", resp.StatusCode, string(body))
	}

	var users []supabaseUser
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("unmarshal users: %w", err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("no user found")
	}
	return &users[0], nil
}

// verifyPassword verifies the password by calling the GoTrue token endpoint.
func (h *Handler) verifyPassword(ctx context.Context, userID, email, password string) bool {
	url := h.supabase.URL() + "/auth/v1/token?grant_type=password"
	payload, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return false
	}
	req.Header.Set("apikey", h.supabase.ServiceKey())
	req.Header.Set("Authorization", "Bearer "+h.supabase.ServiceKey())
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.supabase.HTTPClient().Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
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

// findStaffBySupabaseID looks up the staff record by supabase_user_id.
func (h *Handler) findStaffBySupabaseID(ctx context.Context, supabaseUserID string) (*staffRow, error) {
	uid, err := uuid.Parse(supabaseUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid supabase user id: %w", err)
	}

	row := h.supabase.Pool.QueryRow(ctx,
		"SELECT id, tenant_id, full_name, email, role::text, COALESCE(phone, '') FROM staff WHERE supabase_user_id = $1 AND is_active = true",
		uid,
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
