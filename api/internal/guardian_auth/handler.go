package guardian_auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	appmiddleware "github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
	"github.com/shule360/api/pkg/upstash"
)

type LoginRequest struct {
	Phone    string `json:"phone"`
	PIN      string `json:"pin"`
	TenantID string `json:"tenant_id"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	Guardian  Guardian  `json:"guardian"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Guardian struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	FullName string    `json:"full_name"`
	Phone    string    `json:"phone"`
	Email    string    `json:"email,omitempty"`
}

type Handler struct {
	pool      *pgxpool.Pool
	jwtSecret string
	redis     *upstash.RedisClient
	// secure is passed to session cookies (HTTPS-only in production).
	secure bool
}

func NewHandler(pool *pgxpool.Pool, jwtSecret string, redis *upstash.RedisClient, secure bool) *Handler {
	return &Handler{pool: pool, jwtSecret: jwtSecret, redis: redis, secure: secure}
}

func (h *Handler) Mount(r chi.Router) {
	r.Route("/auth/guardian", func(r chi.Router) {
		// Public: school list for the parent-portal sign-in form.
		r.Get("/schools", h.schools)
		r.Post("/login", h.login)
	})
}

// MountPrivate registers authenticated guardian routes. Call from inside the
// guardian RequireRole group so /me and /logout can trust the verified
// guardian identity in the request context.
func (h *Handler) MountPrivate(r chi.Router) {
	r.Route("/auth/guardian", func(r chi.Router) {
		r.Get("/me", h.me)
		r.Post("/logout", h.logout)
	})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	if req.Phone == "" || req.PIN == "" {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "phone and pin are required")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_TENANT", "Invalid tenant_id")
		return
	}

	// Fail-closed brute-force protection for guardian PINs: 5 attempts per IP
	// and per phone number per 15-minute window.
	if !appmiddleware.CheckLoginRateLimit(r.Context(), w, h.redis, appmiddleware.ClientIP(r), req.TenantID+":"+req.Phone, 900, 5, 5) {
		return
	}

	var guardian struct {
		ID       uuid.UUID
		TenantID uuid.UUID
		FullName string
		Phone    string
		Email    string
		PINHash  string
	}
	err = h.pool.QueryRow(r.Context(), `
		SELECT id, tenant_id, full_name, phone_primary, COALESCE(email, ''), COALESCE(pin_hash, '')
		FROM guardians
		WHERE tenant_id = $1 AND phone_primary = $2
	`, tenantID, req.Phone).Scan(&guardian.ID, &guardian.TenantID, &guardian.FullName, &guardian.Phone, &guardian.Email, &guardian.PINHash)
	if err != nil {
		httputil.RespondUnauthorized(w, "INVALID_CREDENTIALS", "Invalid phone or PIN")
		return
	}

	if guardian.PINHash == "" || !h.verifyPIN(req.PIN, guardian.PINHash) {
		httputil.RespondUnauthorized(w, "INVALID_CREDENTIALS", "Invalid phone or PIN")
		return
	}

	claims := jwt.MapClaims{
		"guardian_id": guardian.ID.String(),
		"tenant_id":   guardian.TenantID.String(),
		"role":        "guardian",
		"exp":         time.Now().Add(24 * time.Hour).Unix(),
		"iat":         time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}

	_, _ = h.pool.Exec(r.Context(), `
		INSERT INTO guardian_sessions (guardian_id, tenant_id, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, guardian.ID, guardian.TenantID, tokenString, time.Now().Add(24*time.Hour))

	// Session cookie (HttpOnly) alongside the JSON body — see auth.Login.
	appmiddleware.SetSessionCookie(w, appmiddleware.CookieGuardianSession, tokenString, h.secure)

	httputil.RespondOK(w, LoginResponse{
		Token: tokenString,
		Guardian: Guardian{
			ID:       guardian.ID,
			TenantID: guardian.TenantID,
			FullName: guardian.FullName,
			Phone:    guardian.Phone,
			Email:    guardian.Email,
		},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
}

func (h *Handler) verifyPIN(pin, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin))
	return err == nil
}

// me handles GET /api/v1/auth/guardian/me (mounted inside the Auth +
// guardian-role group). Identity comes from the verified JWT context; the
// profile is re-read from the database so the parent portal always shows
// current details and deactivated guardians are rejected immediately.
func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	guardianID, ok := appmiddleware.GetGuardianID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Not a guardian session")
		return
	}

	var g Guardian
	err := h.pool.QueryRow(r.Context(), `
		SELECT id, tenant_id, full_name, phone_primary, COALESCE(email, '')
		FROM guardians
		WHERE id = $1
	`, guardianID).Scan(&g.ID, &g.TenantID, &g.FullName, &g.Phone, &g.Email)
	if err != nil {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Guardian account not found")
		return
	}

	httputil.RespondOK(w, map[string]any{"guardian": g})
}

// logout handles POST /api/v1/auth/guardian/logout. Unlike staff sessions
// (stateless JWTs), guardian sessions are tracked in the guardian_sessions
// table, so logout actually revokes them: all live sessions for this guardian
// are deleted and the HttpOnly cookie is cleared.
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if guardianID, ok := appmiddleware.GetGuardianID(r); ok {
		_, _ = h.pool.Exec(r.Context(),
			`DELETE FROM guardian_sessions WHERE guardian_id = $1`, guardianID)
	}
	appmiddleware.ClearSessionCookie(w, appmiddleware.CookieGuardianSession, h.secure)
	httputil.RespondOK(w, map[string]string{"status": "logged_out"})
}

// schools handles GET /api/v1/auth/guardian/schools.
// Public endpoint that powers the school picker on the parent portal sign-in
// form, so guardians never need to know their tenant UUID. Returns only the
// minimal non-sensitive fields (id + display name).
func (h *Handler) schools(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `SELECT id, name FROM tenants ORDER BY name`)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	defer rows.Close()

	type school struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}
	schools := []school{}
	for rows.Next() {
		var s school
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		schools = append(schools, s)
	}
	httputil.RespondOK(w, schools)
}
