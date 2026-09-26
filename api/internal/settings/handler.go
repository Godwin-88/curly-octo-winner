package settings

// HTTP surface for the Settings area.
//
//	GET    /api/v1/settings                        profile + operational config
//	PATCH  /api/v1/settings                        (principal / super admin)
//	GET    /api/v1/settings/integrations           all providers, secrets redacted
//	PUT    /api/v1/settings/integrations/{provider} (principal / super admin)
//	DELETE /api/v1/settings/integrations/{provider} (principal / super admin)
//	POST   /api/v1/settings/integrations/{provider}/test
//
// Reads are open to any signed-in staff member (a teacher seeing whether M-Pesa
// is configured is harmless and useful); writes are restricted by the router to
// principal/super_admin, which mirrors how a school assigns responsibility.

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	appmiddleware "github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler serves the settings routes.
type Handler struct {
	service  *Service
	platform PlatformCredentials
}

// NewHandler creates the settings handler. platform carries the
// environment-level credentials used when a school inherits the defaults.
func NewHandler(service *Service, platform PlatformCredentials) *Handler {
	return &Handler{service: service, platform: platform}
}

// Mount registers the settings routes on an authenticated router.
//
// Reads are open to any signed-in staff member: seeing whether M-Pesa is
// configured for the school is harmless, and it lets non-admin staff
// understand why a feature is switched off. Writes are restricted to the roles
// passed in writeRoles (the server passes principal/super_admin) using
// per-route middleware.
//
// Everything lives in ONE r.Route call on purpose: chi shares a single radix
// tree across Group()s, so mounting the same pattern twice panics at startup.
func (h *Handler) Mount(r chi.Router, writeRoles ...string) {
	r.Route("/settings", func(r chi.Router) {
		r.Get("/", h.getAll)
		r.Get("/integrations", h.listIntegrations)

		if len(writeRoles) > 0 {
			admin := r.With(appmiddleware.RequireRole(writeRoles...))
			admin.Patch("/", h.update)
			admin.Put("/integrations/{provider}", h.saveIntegration)
			admin.Delete("/integrations/{provider}", h.deleteIntegration)
			admin.Post("/integrations/{provider}/test", h.testIntegration)
		}
	})
}

// getAll returns the school profile and operational configuration in one call,
// so the screen renders from a single request.
func (h *Handler) getAll(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appmiddleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant not found")
		return
	}

	profile, err := h.service.GetProfile(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	settings, err := h.service.GetSettings(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}

	httputil.RespondOK(w, map[string]any{
		"profile":   profile,
		"settings":  settings,
		"providers": Providers,
	})
}

// update applies profile and/or operational changes in one request.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appmiddleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant not found")
		return
	}

	var body struct {
		Profile  *ProfilePatch  `json:"profile"`
		Settings *SettingsPatch `json:"settings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	if body.Profile == nil && body.Settings == nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Provide a profile and/or settings object")
		return
	}

	actor := actorID(r)

	if body.Profile != nil {
		if _, err := h.service.UpdateProfile(r.Context(), tenantID, *body.Profile); err != nil {
			httputil.RespondBadRequest(w, "INVALID_PROFILE", err.Error())
			return
		}
		_ = h.service.LogAudit(r.Context(), tenantID, actor, "UPDATE", "tenant_profile",
			map[string]any{"section": "school profile"})
	}

	var settings *Settings
	if body.Settings != nil {
		updated, err := h.service.UpdateSettings(r.Context(), tenantID, actorStaff(actor), *body.Settings)
		if err != nil {
			httputil.RespondBadRequest(w, "INVALID_SETTINGS", err.Error())
			return
		}
		settings = updated
		_ = h.service.LogAudit(r.Context(), tenantID, actor, "UPDATE", "tenant_settings",
			map[string]any{"section": "operational settings"})
	}

	profile, err := h.service.GetProfile(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	if settings == nil {
		if settings, err = h.service.GetSettings(r.Context(), tenantID); err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
	}

	httputil.RespondOK(w, map[string]any{"profile": profile, "settings": settings})
}

// listIntegrations returns every provider for the school with secrets redacted.
func (h *Handler) listIntegrations(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appmiddleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant not found")
		return
	}
	integrations, err := h.service.ListIntegrations(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, integrations)
}

// saveIntegration stores a provider's configuration and credentials.
func (h *Handler) saveIntegration(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appmiddleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant not found")
		return
	}
	provider := chi.URLParam(r, "provider")
	if !IsProvider(provider) {
		httputil.RespondBadRequest(w, "UNKNOWN_PROVIDER", "Supported providers: "+strings.Join(Providers, ", "))
		return
	}

	var in IntegrationInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	saved, err := h.service.SaveIntegration(r.Context(), tenantID, actorStaff(actorID(r)), provider, in)
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_INTEGRATION", err.Error())
		return
	}
	// The audit entry records WHICH credentials were set, never their values.
	_ = h.service.LogAudit(r.Context(), tenantID, actorID(r), "UPDATE", "tenant_integration",
		map[string]any{"provider": provider, "secrets_set": saved.SecretFields})

	httputil.RespondOK(w, saved)
}

// deleteIntegration removes a provider's configuration.
func (h *Handler) deleteIntegration(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appmiddleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant not found")
		return
	}
	provider := chi.URLParam(r, "provider")
	if err := h.service.DeleteIntegration(r.Context(), tenantID, provider); err != nil {
		httputil.RespondNotFound(w, "NOT_CONFIGURED", err.Error())
		return
	}
	_ = h.service.LogAudit(r.Context(), tenantID, actorID(r), "DELETE", "tenant_integration",
		map[string]any{"provider": provider})
	httputil.RespondOK(w, map[string]string{"status": "deleted"})
}

// testIntegration performs a live connectivity check for a provider and
// remembers the outcome, so the screen shows whether it works.
func (h *Handler) testIntegration(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := appmiddleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant not found")
		return
	}
	provider := chi.URLParam(r, "provider")
	if !IsProvider(provider) {
		httputil.RespondBadRequest(w, "UNKNOWN_PROVIDER", "Supported providers: "+strings.Join(Providers, ", "))
		return
	}

	result, err := h.service.TestProvider(r.Context(), tenantID, provider, h.platform)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}

	status := "failed"
	if result.OK {
		status = "ok"
	}
	_ = h.service.RecordTestResult(r.Context(), tenantID, provider, status, result.Message)
	_ = h.service.LogAudit(r.Context(), tenantID, actorID(r), "TEST", "tenant_integration",
		map[string]any{"provider": provider, "ok": result.OK})

	httputil.RespondOK(w, result)
}

// actorID returns the verified staff member behind the request.
func actorID(r *http.Request) *uuid.UUID {
	if id, ok := appmiddleware.GetStaffID(r); ok {
		return &id
	}
	return nil
}

// actorStaff unwraps a pointer for the service signatures that take a value.
func actorStaff(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}
