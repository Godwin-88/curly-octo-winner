// HTTP surface for the learner import staging area.
//
// Tenant identity always comes from the verified JWT (middleware.GetTenantID),
// never from the request body, so one school can never stage or read another
// school's roster by guessing a tenant id.
package learnerimport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler serves the staging endpoints.
type Handler struct {
	service *Service
}

// NewHandler builds a Handler over a Service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Mount registers the routes. It sits under the same role group as the rest of
// the learner routes, because a roster is learner data.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/learner-imports", func(r chi.Router) {
		r.Get("/", h.listBatches)
		r.Post("/", h.createBatch)
		// Registered before /{id} so "help" is not captured as an id.
		r.Get("/help", h.importHelp)

		r.Get("/{batchId}/rows", h.listRows)
		r.Get("/{batchId}", h.getBatch)
		r.Delete("/{batchId}", h.deleteBatch)

		r.Patch("/rows/{rowId}", h.updateRow)
		r.Post("/rows/{rowId}/promote", h.promoteRow)
		r.Delete("/rows/{rowId}", h.deleteRow)
	})
}

type createBatchRequest struct {
	// CSV is pasted or uploaded file text. Headers are matched loosely; see
	// ParseCSV.
	CSV string `json:"csv"`
	// Filename is only a label for the batch list.
	Filename string `json:"filename"`
}

// createBatch stages an upload. Every row is stored, complete or not; the
// response carries the counts so the UI can say "41 staged, 12 ready" without
// a second request.
func (h *Handler) createBatch(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	actorID, _ := middleware.GetStaffID(r)

	var req createBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	if req.CSV == "" {
		httputil.RespondBadRequest(w, "INVALID_IMPORT", "paste a CSV or upload a file first")
		return
	}

	rows, err := ParseCSV(req.CSV)
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_IMPORT", err.Error())
		return
	}

	batch, err := h.service.CreateBatch(r.Context(), tenantID, actorID, req.Filename, rows)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondCreated(w, batch)
}

// listBatches returns the school's uploads, newest first.
func (h *Handler) listBatches(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	q := r.URL.Query()
	batches, total, err := h.service.ListBatches(r.Context(), tenantID,
		intQuery(q.Get("limit"), 50), intQuery(q.Get("offset"), 0))
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, map[string]any{
		"items": batches,
		"total": total,
	})
}

func (h *Handler) getBatch(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	batchID, ok := pathUUID(w, r, "batchId")
	if !ok {
		return
	}
	batch, err := h.service.GetBatch(r.Context(), tenantID, batchID)
	if err != nil {
		respondErr(w, err)
		return
	}
	httputil.RespondOK(w, batch)
}

// listRows returns a batch's staged rows. `status` and `tag` narrow the list;
// both are optional.
func (h *Handler) listRows(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	batchID, ok := pathUUID(w, r, "batchId")
	if !ok {
		return
	}
	q := r.URL.Query()
	rows, total, err := h.service.ListRows(r.Context(), tenantID, batchID,
		q.Get("status"), q.Get("tag"))
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, map[string]any{
		"items": rows,
		"total": total,
	})
}

// updateRow applies a partial edit to one staged row. This is how a school
// completes the half-finished rows: edit the field, and the row becomes
// promotable on its own.
func (h *Handler) updateRow(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	rowID, ok := pathUUID(w, r, "rowId")
	if !ok {
		return
	}
	var patch RowPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	row, err := h.service.UpdateRow(r.Context(), tenantID, rowID, patch)
	if err != nil {
		respondErr(w, err)
		return
	}
	httputil.RespondOK(w, row)
}

// promoteRow turns one sufficient row into a real learner.
func (h *Handler) promoteRow(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	rowID, ok := pathUUID(w, r, "rowId")
	if !ok {
		return
	}
	result, err := h.service.Promote(r.Context(), tenantID, rowID)
	if err != nil {
		respondErr(w, err)
		return
	}
	httputil.RespondCreated(w, result)
}

func (h *Handler) deleteRow(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	rowID, ok := pathUUID(w, r, "rowId")
	if !ok {
		return
	}
	if err := h.service.DeleteRow(r.Context(), tenantID, rowID); err != nil {
		respondErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteBatch discards a whole upload and its rows.
func (h *Handler) deleteBatch(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	batchID, ok := pathUUID(w, r, "batchId")
	if !ok {
		return
	}
	if err := h.service.DeleteBatch(r.Context(), tenantID, batchID); err != nil {
		respondErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// importHelp describes the expected columns, so the user does not have to
// guess what to put in the spreadsheet.
func (h *Handler) importHelp(w http.ResponseWriter, r *http.Request) {
	httputil.RespondOK(w, map[string]any{
		"columns": []map[string]any{
			{"name": "Parent Name", "aliases": []string{"Parent", "Guardian"}, "required": false},
			{"name": "Parent Phone", "aliases": []string{"Phone", "Contact"}, "required": false},
			{"name": "Student Name", "aliases": []string{"Learner", "Name"}, "required": true},
			{"name": "Student Number", "aliases": []string{"Number", "Admission Number", "UPI"}, "required": true},
			{"name": "Grade", "aliases": []string{"Class", "Standard"}, "required": true},
			{"name": "Stream", "aliases": nil, "required": false},
			{"name": "Tags", "aliases": []string{"Labels"}, "required": false,
				"note": "Separate several with ; or ,"},
		},
		"notes": []string{
			"Every column may be left blank -- a row is staged either way.",
			"A row becomes a learner once it has a student name, student number and grade.",
			"Headings are matched loosely, so 'Parent Name' and 'parent_name' both work.",
			"Extra columns such as House or Remarks are ignored.",
		},
	})
}

// pathUUID reads a UUID path parameter, answering 400 itself on failure.
func pathUUID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", fmt.Sprintf("Invalid %s", name))
		return uuid.Nil, false
	}
	return id, true
}

// respondErr maps a service error onto the right status, keeping internal
// details out of the response body.
//
// The distinction matters to the client: a 409 tells the UI "this conflicts
// with something that already exists, point the user at the field", while a
// 500 would render as a generic failure and leave them unable to act.
func respondErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.RespondNotFound(w, "NOT_FOUND", "That import row or file no longer exists.")
	case errors.Is(err, ErrNotSufficient):
		// The message names the missing fields, which is what the user needs.
		httputil.RespondBadRequest(w, "NOT_SUFFICIENT", err.Error())
	case errors.Is(err, ErrDuplicateNumber):
		httputil.RespondConflict(w, "DUPLICATE_NUMBER", err.Error())
	case errors.Is(err, ErrAlreadyImported):
		httputil.RespondConflict(w, "ALREADY_IMPORTED", err.Error())
	case errors.Is(err, ErrBatchHasLearners):
		httputil.RespondConflict(w, "BATCH_HAS_LEARNERS", err.Error())
	default:
		httputil.RespondInternalError(w, err)
	}
}

func intQuery(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
