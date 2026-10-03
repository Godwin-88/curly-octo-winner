package platform

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler serves /platform. Mount it behind middleware.Auth and
// middleware.RequirePlatform.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Mount(r chi.Router) {
	r.Route("/platform", func(r chi.Router) {
		r.Use(middleware.RequirePlatform)

		r.Get("/groups", h.listGroups)
		r.Post("/groups", h.createGroup)
		r.Get("/groups/{id}", h.getGroup)
		r.Patch("/groups/{id}", h.renameGroup)

		r.Post("/schools", h.createSchool)
		r.Patch("/schools/{id}", h.updateSchool)
		r.Put("/schools/{id}/modules", h.setModules)

		r.Get("/users", h.listUsers)
		r.Post("/users", h.createUser)
		r.Get("/users/{id}", h.getUser)
		r.Patch("/users/{id}", h.updateUser)
		r.Post("/users/{id}/deactivate", h.setActive(false))
		r.Post("/users/{id}/activate", h.setActive(true))
		r.Post("/users/{id}/reset-password", h.resetPassword)
	})
}

func respondError(w http.ResponseWriter, err error) {
	var validation *ValidationError
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.RespondNotFound(w, "NOT_FOUND", "Not found")
	case errors.As(err, &validation):
		httputil.RespondBadRequest(w, "INVALID_REQUEST", validation.Message)
	default:
		httputil.RespondInternalError(w, err)
	}
}

func pathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid id")
		return uuid.Nil, false
	}
	return id, true
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := json.NewDecoder(r.Body).Decode(into); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return false
	}
	return true
}

func (h *Handler) listGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.ListGroups(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	httputil.RespondOK(w, groups)
}

func (h *Handler) getGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	group, err := h.service.GetGroup(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	httputil.RespondOK(w, group)
}

func (h *Handler) createGroup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if !decode(w, r, &req) {
		return
	}
	group, err := h.service.CreateGroup(r.Context(), req.Name, req.Slug)
	if err != nil {
		respondError(w, err)
		return
	}
	httputil.RespondCreated(w, group)
}

func (h *Handler) renameGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &req) {
		return
	}
	group, err := h.service.RenameGroup(r.Context(), id, req.Name)
	if err != nil {
		respondError(w, err)
		return
	}
	httputil.RespondOK(w, group)
}

type schoolRequest struct {
	Name    string     `json:"name"`
	Slug    string     `json:"slug"`
	GroupID *uuid.UUID `json:"group_id"`
}

func (h *Handler) createSchool(w http.ResponseWriter, r *http.Request) {
	var req schoolRequest
	if !decode(w, r, &req) {
		return
	}
	school, err := h.service.CreateSchool(r.Context(), req.Name, req.Slug, req.GroupID)
	if err != nil {
		respondError(w, err)
		return
	}
	httputil.RespondCreated(w, school)
}

func (h *Handler) updateSchool(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req schoolRequest
	if !decode(w, r, &req) {
		return
	}
	school, err := h.service.UpdateSchool(r.Context(), id, req.Name, req.GroupID)
	if err != nil {
		respondError(w, err)
		return
	}
	httputil.RespondOK(w, school)
}

func (h *Handler) setModules(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req struct {
		Modules []string `json:"modules"`
	}
	if !decode(w, r, &req) {
		return
	}
	school, err := h.service.SetModules(r.Context(), id, req.Modules)
	if err != nil {
		respondError(w, err)
		return
	}
	httputil.RespondOK(w, school)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	httputil.RespondOK(w, users)
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	user, err := h.service.GetUser(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	httputil.RespondOK(w, user)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req UserInput
	if !decode(w, r, &req) {
		return
	}
	user, err := h.service.CreateUser(r.Context(), req)
	if err != nil {
		respondError(w, err)
		return
	}
	httputil.RespondCreated(w, user)
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req UserInput
	if !decode(w, r, &req) {
		return
	}
	user, err := h.service.UpdateUser(r.Context(), middleware.GetSession(r).OperatorID, id, req)
	if err != nil {
		respondError(w, err)
		return
	}
	httputil.RespondOK(w, user)
}

func (h *Handler) setActive(active bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		user, err := h.service.SetActive(r.Context(), middleware.GetSession(r).OperatorID, id, active)
		if err != nil {
			respondError(w, err)
			return
		}
		httputil.RespondOK(w, user)
	}
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	user, err := h.service.ResetPassword(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}
	httputil.RespondOK(w, user)
}
