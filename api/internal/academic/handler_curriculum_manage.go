package academic

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/academic/curriculum"
	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// This file holds the update/delete half of curriculum management. The list and
// create handlers live in handler.go; keeping the management operations
// together makes the "what can this item do" story easy to follow.

// curriculumID reads the {id} path parameter.
func curriculumID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid curriculum item ID")
		return uuid.Nil, false
	}
	return id, true
}

// curriculumConflict builds the "already in use" message for a curriculum item.
//
// The KICD code is unique per tenant, so a duplicate is a normal thing for a
// data clerk to do while entering a long list of codes — it deserves a plain
// sentence naming the code, not a raw constraint error.
func curriculumConflict(code, kind string) string {
	if code == "" {
		return "That " + kind + " already exists."
	}
	return "KICD code " + code + " is already used by another " + kind + ". Use a different code."
}

// respondCurriculumErr maps a service error onto the right status code.
//
// A curriculum write can fail for three different reasons and the user needs to
// be told which: the row is gone (404, so the UI can drop it from the list), it
// still has children (409, so the UI explains what to remove first), or the code
// is already taken (409, so the UI can point at the field).
func respondCurriculumErr(w http.ResponseWriter, err error, kind, code string) {
	var children *curriculum.HasChildrenError
	switch {
	case curriculum.IsNotFound(err):
		httputil.RespondNotFound(w, "NOT_FOUND", "That "+kind+" no longer exists.")
	case errors.As(err, &children):
		httputil.RespondConflict(w, "HAS_DEPENDENTS", children.Error())
	case httputil.IsUniqueViolation(err):
		httputil.RespondConflict(w, "ALREADY_EXISTS", curriculumConflict(code, kind))
	default:
		httputil.RespondInternalError(w, err)
	}
}

// decodeBody reads a JSON body, rejecting unknown fields so a typo in the client
// ("kicdcode") fails loudly instead of silently storing an empty code.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return false
	}
	return true
}

// requireName is the one validation rule every curriculum item shares: an
// unnamed learning area is unusable in every dropdown downstream.
func requireName(w http.ResponseWriter, name string) bool {
	if strings.TrimSpace(name) == "" {
		httputil.RespondBadRequest(w, "NAME_REQUIRED", "A name is required.")
		return false
	}
	return true
}

func (h *Handler) updateLearningArea(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := curriculumID(w, r)
	if !ok {
		return
	}
	var la curriculum.LearningArea
	if !decodeBody(w, r, &la) {
		return
	}
	if !requireName(w, la.Name) {
		return
	}
	result, err := h.curriculumSvc.UpdateLearningArea(r.Context(), tenantID, id, &la)
	if err != nil {
		respondCurriculumErr(w, err, "learning area", la.KICDCode)
		return
	}
	httputil.RespondOK(w, result)
}

func (h *Handler) deleteLearningArea(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := curriculumID(w, r)
	if !ok {
		return
	}
	if err := h.curriculumSvc.DeleteLearningArea(r.Context(), tenantID, id); err != nil {
		respondCurriculumErr(w, err, "learning area", "")
		return
	}
	httputil.RespondNoContent(w)
}

func (h *Handler) updateStrand(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := curriculumID(w, r)
	if !ok {
		return
	}
	var st curriculum.Strand
	if !decodeBody(w, r, &st) {
		return
	}
	if !requireName(w, st.Name) {
		return
	}
	result, err := h.curriculumSvc.UpdateStrand(r.Context(), tenantID, id, &st)
	if err != nil {
		respondCurriculumErr(w, err, "strand", st.KICDCode)
		return
	}
	httputil.RespondOK(w, result)
}

func (h *Handler) deleteStrand(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := curriculumID(w, r)
	if !ok {
		return
	}
	if err := h.curriculumSvc.DeleteStrand(r.Context(), tenantID, id); err != nil {
		respondCurriculumErr(w, err, "strand", "")
		return
	}
	httputil.RespondNoContent(w)
}

func (h *Handler) updateSubStrand(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := curriculumID(w, r)
	if !ok {
		return
	}
	var ss curriculum.SubStrand
	if !decodeBody(w, r, &ss) {
		return
	}
	if !requireName(w, ss.Name) {
		return
	}
	result, err := h.curriculumSvc.UpdateSubStrand(r.Context(), tenantID, id, &ss)
	if err != nil {
		respondCurriculumErr(w, err, "sub-strand", ss.KICDCode)
		return
	}
	httputil.RespondOK(w, result)
}

func (h *Handler) deleteSubStrand(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := curriculumID(w, r)
	if !ok {
		return
	}
	if err := h.curriculumSvc.DeleteSubStrand(r.Context(), tenantID, id); err != nil {
		respondCurriculumErr(w, err, "sub-strand", "")
		return
	}
	httputil.RespondNoContent(w)
}

func (h *Handler) updateCoreCompetency(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := curriculumID(w, r)
	if !ok {
		return
	}
	var cc curriculum.CoreCompetency
	if !decodeBody(w, r, &cc) {
		return
	}
	if !requireName(w, cc.Name) {
		return
	}
	result, err := h.curriculumSvc.UpdateCoreCompetency(r.Context(), tenantID, id, &cc)
	if err != nil {
		respondCurriculumErr(w, err, "core competency", cc.KICDCode)
		return
	}
	httputil.RespondOK(w, result)
}

func (h *Handler) deleteCoreCompetency(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := curriculumID(w, r)
	if !ok {
		return
	}
	if err := h.curriculumSvc.DeleteCoreCompetency(r.Context(), tenantID, id); err != nil {
		respondCurriculumErr(w, err, "core competency", "")
		return
	}
	httputil.RespondNoContent(w)
}

func (h *Handler) updateValue(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := curriculumID(w, r)
	if !ok {
		return
	}
	var v curriculum.Value
	if !decodeBody(w, r, &v) {
		return
	}
	if !requireName(w, v.Name) {
		return
	}
	result, err := h.curriculumSvc.UpdateValue(r.Context(), tenantID, id, &v)
	if err != nil {
		respondCurriculumErr(w, err, "value", v.KICDCode)
		return
	}
	httputil.RespondOK(w, result)
}

func (h *Handler) deleteValue(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := curriculumID(w, r)
	if !ok {
		return
	}
	if err := h.curriculumSvc.DeleteValue(r.Context(), tenantID, id); err != nil {
		respondCurriculumErr(w, err, "value", "")
		return
	}
	httputil.RespondNoContent(w)
}
