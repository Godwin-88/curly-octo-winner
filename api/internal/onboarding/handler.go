package onboarding

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
	"github.com/shule360/api/pkg/upstash"
)

// Who may register a school themselves (SIGNUP_MODE).
const (
	// SignupOpen: anyone with the address.
	SignupOpen = "open"
	// SignupCode: anyone who was given the code in SIGNUP_CODE.
	SignupCode = "code"
	// SignupClosed: nobody; schools are created by a platform administrator.
	SignupClosed = "closed"
)

// Handler serves registration, the setup checklist and a school's users.
type Handler struct {
	service *Service
	redis   *upstash.RedisClient
	mode    string
	code    string
}

func NewHandler(service *Service, redis *upstash.RedisClient, mode, code string) *Handler {
	return &Handler{service: service, redis: redis, mode: mode, code: strings.TrimSpace(code)}
}

// MountPublic registers the routes that need no session: asking whether a
// school can be registered, and registering one.
func (h *Handler) MountPublic(r chi.Router) {
	r.Get("/signup", h.signupOptions)
	r.Post("/signup", h.signup)
}

// MountPlatform registers what a platform administrator does: creating a
// school together with its first administrator. Mount behind RequirePlatform.
func (h *Handler) MountPlatform(r chi.Router) {
	r.Post("/platform/onboard-school", h.onboardSchool)
}

// MountSchool registers what is done inside a school. The checklist is for
// every member of staff; users are managed by a principal.
func (h *Handler) MountSchool(r chi.Router) {
	r.Get("/onboarding/status", h.status)
	r.Route("/school/users", func(r chi.Router) {
		r.Use(middleware.RequireRole("principal", "super_admin"))
		r.Get("/", h.listUsers)
		r.Post("/", h.addUser)
		r.Get("/{id}", h.getUser)
		r.Patch("/{id}", h.updateUser)
		r.Post("/{id}/deactivate", h.setActive(false))
		r.Post("/{id}/activate", h.setActive(true))
		r.Post("/{id}/reset-password", h.resetPassword)
	})
}

func respond(w http.ResponseWriter, err error) {
	var validation *ValidationError
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.RespondNotFound(w, "NOT_FOUND", "Not found")
	case errors.As(err, &validation):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "INVALID", "error": validation.Message, "field": validation.Field})
	default:
		httputil.RespondInternalError(w, err)
	}
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(into); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "The request could not be read.")
		return false
	}
	return true
}

func (h *Handler) signupOptions(w http.ResponseWriter, _ *http.Request) {
	httputil.RespondOK(w, map[string]any{
		"open":       h.mode != SignupClosed,
		"needs_code": h.mode == SignupCode,
		"counties":   Counties,
	})
}

// signup lets a person register their own school and become its principal.
func (h *Handler) signup(w http.ResponseWriter, r *http.Request) {
	if h.mode == SignupClosed {
		httputil.RespondForbidden(w, "SIGNUP_CLOSED", "Schools are set up by the Shule360 team. Contact us to get started.")
		return
	}
	var req struct {
		Registration
		Code string `json:"code"`
	}
	if !decode(w, r, &req) {
		return
	}
	// The same limit as sign-in, and it fails closed: this endpoint creates
	// accounts, so it must not be a way to create thousands of them.
	if !middleware.CheckLoginRateLimit(r.Context(), w, h.redis, middleware.ClientIP(r), "signup:"+strings.ToLower(strings.TrimSpace(req.AdminEmail)), 900, 5, 3) {
		return
	}
	if h.mode == SignupCode && (h.code == "" || subtle.ConstantTimeCompare([]byte(strings.TrimSpace(req.Code)), []byte(h.code)) != 1) {
		respond(w, invalid("code", "That registration code is not right. Ask the Shule360 team for one."))
		return
	}
	if req.Password == "" {
		respond(w, invalid("password", "Choose a password of at least 10 characters."))
		return
	}
	// A person registering their own school never chooses a group.
	req.GroupID = nil

	created, err := h.service.RegisterSchool(r.Context(), req.Registration)
	if err != nil {
		respond(w, err)
		return
	}
	httputil.RespondCreated(w, created)
}

// onboardSchool is a platform administrator creating a school for someone:
// the administrator's password is generated and shown once.
func (h *Handler) onboardSchool(w http.ResponseWriter, r *http.Request) {
	var req Registration
	if !decode(w, r, &req) {
		return
	}
	req.Password = ""
	created, err := h.service.RegisterSchool(r.Context(), req)
	if err != nil {
		respond(w, err)
		return
	}
	httputil.RespondCreated(w, created)
}

func school(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
	}
	return tenantID, ok
}

func pathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", "Not found")
		return uuid.Nil, false
	}
	return id, true
}

func actor(r *http.Request) uuid.UUID {
	id, _ := middleware.GetStaffID(r)
	return id
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	status, err := h.service.Status(r.Context(), tenantID)
	if err != nil {
		respond(w, err)
		return
	}
	httputil.RespondOK(w, status)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	users, err := h.service.ListUsers(r.Context(), tenantID, r.URL.Query().Get("search"))
	if err != nil {
		respond(w, err)
		return
	}
	httputil.RespondOK(w, users)
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	user, err := h.service.GetUser(r.Context(), tenantID, id)
	if err != nil {
		respond(w, err)
		return
	}
	httputil.RespondOK(w, user)
}

func (h *Handler) addUser(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	var req UserInput
	if !decode(w, r, &req) {
		return
	}
	user, err := h.service.AddUser(r.Context(), tenantID, req)
	if err != nil {
		respond(w, err)
		return
	}
	httputil.RespondCreated(w, user)
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req UserInput
	if !decode(w, r, &req) {
		return
	}
	user, err := h.service.UpdateUser(r.Context(), tenantID, actor(r), id, req)
	if err != nil {
		respond(w, err)
		return
	}
	httputil.RespondOK(w, user)
}

func (h *Handler) setActive(active bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := school(w, r)
		if !ok {
			return
		}
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		user, err := h.service.SetActive(r.Context(), tenantID, actor(r), id, active)
		if err != nil {
			respond(w, err)
			return
		}
		httputil.RespondOK(w, user)
	}
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := school(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	user, err := h.service.ResetPassword(r.Context(), tenantID, id)
	if err != nil {
		respond(w, err)
		return
	}
	httputil.RespondOK(w, user)
}
