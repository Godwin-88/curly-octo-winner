package contacts

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler exposes the contact book over HTTP.
//
// Tenant identity always comes from the verified JWT (middleware.GetTenantID),
// never from the request body, so one school can never read or write another
// school's contacts. The actor id is recorded for the audit trail.
type Handler struct {
	service *Service
}

// NewHandler creates a contacts HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Mount registers the contact routes.
//
//	GET    /contacts              list + search + filter
//	POST   /contacts              create one
//	POST   /contacts/import       bulk create from pasted CSV or JSON rows
//	GET    /contacts/summary      counts for the dashboard tiles
//	GET    /contacts/import/help  the columns the importer understands
//	GET    /contacts/{id}         read one
//	PATCH  /contacts/{id}         edit
//	DELETE /contacts/{id}         archive (soft delete)
//	POST   /contacts/{id}/restore undo an archive
func (h *Handler) Mount(r chi.Router) {
	r.Route("/contacts", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Post("/import", h.bulkImport)
		r.Get("/summary", h.summary)
		r.Get("/import/help", h.importHelp)
		r.Get("/{id}", h.get)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Post("/{id}/restore", h.restore)
	})
}

// contactRequest is the single-contact payload shared by create and update.
type contactRequest struct {
	FullName     string   `json:"full_name"`
	Phone        string   `json:"phone"`
	Email        string   `json:"email"`
	Relationship string   `json:"relationship"`
	GradeStream  string   `json:"grade_stream"`
	Tags         []string `json:"tags"`
	Notes        string   `json:"notes"`
	IsOptedOut   bool     `json:"is_opted_out"`
	GuardianID   *string  `json:"guardian_id"`
}

// toInput validates and normalizes the request. The bulk path applies the same
// rules, so a number accepted in one place is accepted in the other.
func (req contactRequest) toInput() (ContactInput, error) {
	name := strings.TrimSpace(req.FullName)
	if name == "" {
		return ContactInput{}, fmt.Errorf("full name is required")
	}
	if len(name) > 255 {
		return ContactInput{}, fmt.Errorf("full name must be 255 characters or fewer")
	}

	phone, err := NormalizePhone(req.Phone)
	if err != nil {
		return ContactInput{}, err
	}
	if err := ValidateEmail(req.Email); err != nil {
		return ContactInput{}, err
	}

	tags, err := NormalizeTags(req.Tags)
	if err != nil {
		return ContactInput{}, err
	}

	rel := strings.TrimSpace(req.Relationship)
	if len(rel) > 50 {
		return ContactInput{}, fmt.Errorf("relationship must be 50 characters or fewer")
	}
	grade := strings.TrimSpace(req.GradeStream)
	if len(grade) > 100 {
		return ContactInput{}, fmt.Errorf("class/grade must be 100 characters or fewer")
	}
	notes := strings.TrimSpace(req.Notes)
	if len(notes) > 2000 {
		return ContactInput{}, fmt.Errorf("notes must be 2000 characters or fewer")
	}

	in := ContactInput{
		FullName:     name,
		Phone:        phone,
		Email:        strings.TrimSpace(req.Email),
		Relationship: rel,
		GradeStream:  grade,
		Tags:         tags,
		Notes:        notes,
		IsOptedOut:   req.IsOptedOut,
	}

	if req.GuardianID != nil && strings.TrimSpace(*req.GuardianID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*req.GuardianID))
		if err != nil {
			return ContactInput{}, fmt.Errorf("guardian_id is not a valid id")
		}
		in.GuardianID = &id
	}
	return in, nil
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

// list handles GET /contacts
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	q := r.URL.Query()
	items, total, err := h.service.List(r.Context(), tenantID,
		q.Get("search"), q.Get("tag"), q.Get("status"),
		intQuery(q.Get("limit"), 100), intQuery(q.Get("offset"), 0))
	if err != nil {
		// An unknown status filter is a client mistake, not a server fault.
		if strings.Contains(err.Error(), "unknown status filter") {
			httputil.RespondBadRequest(w, "INVALID_FILTER", err.Error())
			return
		}
		httputil.RespondInternalError(w, err)
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count")
	httputil.RespondOK(w, items)
}

// summary handles GET /contacts/summary
func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	sum, err := h.service.Summary(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, sum)
}

// importHelp handles GET /contacts/import/help — the columns the importer
// understands, so the UI shows a real hint instead of a guess.
func (h *Handler) importHelp(w http.ResponseWriter, r *http.Request) {
	httputil.RespondOK(w, map[string]any{
		"columns": []map[string]string{
			{"field": "full_name", "label": "Name", "required": "true", "example": "Jane Doe"},
			{"field": "phone", "label": "Phone", "required": "true", "example": "0712345678"},
			{"field": "email", "label": "Email", "required": "false", "example": "jane@example.com"},
			{"field": "relationship", "label": "Relationship", "required": "false", "example": "parent"},
			{"field": "grade_stream", "label": "Class / Grade", "required": "false", "example": "Grade 4 North"},
			{"field": "tags", "label": "Tags (comma separated)", "required": "false", "example": "grade 4, boarding"},
			{"field": "notes", "label": "Notes", "required": "false", "example": "Prefers WhatsApp"},
			{"field": "is_opted_out", "label": "Opted out (yes/no)", "required": "false", "example": "no"},
		},
		"accepted_phone_formats": []string{
			"0712345678", "+254712345678", "254712345678", "712345678", "0712 345 678",
		},
		"max_rows": maxBulkRows,
	})
}

// create handles POST /contacts
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	actorID, _ := middleware.GetStaffID(r)

	var req contactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	in, err := req.toInput()
	if err != nil {
		httputil.RespondBadRequest(w, "VALIDATION_FAILED", err.Error())
		return
	}

	contact, err := h.service.Create(r.Context(), tenantID, actorID, in)
	switch {
	case errors.Is(err, ErrAlreadyExists):
		// 409 lets the UI say "already saved as X — update it instead?",
		// which beats a generic failure for the single most common mistake.
		httputil.RespondConflict(w, "DUPLICATE_CONTACT", err.Error())
	case err != nil:
		httputil.RespondInternalError(w, err)
	default:
		httputil.RespondCreated(w, contact)
	}
}

// get handles GET /contacts/{id}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	contact, err := h.service.Get(r.Context(), tenantID, id)
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
	case err != nil:
		httputil.RespondInternalError(w, err)
	default:
		httputil.RespondOK(w, contact)
	}
}

// update handles PATCH /contacts/{id}
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	var req contactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	in, err := req.toInput()
	if err != nil {
		httputil.RespondBadRequest(w, "VALIDATION_FAILED", err.Error())
		return
	}

	contact, err := h.service.Update(r.Context(), tenantID, id, in)
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
	case errors.Is(err, ErrAlreadyExists):
		httputil.RespondConflict(w, "DUPLICATE_CONTACT", "another contact already uses this phone number")
	case err != nil:
		httputil.RespondInternalError(w, err)
	default:
		httputil.RespondOK(w, contact)
	}
}

// delete handles DELETE /contacts/{id} — an archive, not a hard delete.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), tenantID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
			return
		}
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// restore handles POST /contacts/{id}/restore — undo an archive.
func (h *Handler) restore(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	if err := h.service.Restore(r.Context(), tenantID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.RespondNotFound(w, "NOT_FOUND", "contact is not archived")
			return
		}
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondNoContent(w)
}

// pathID parses the {id} URL parameter, answering 400 when it is not a UUID.
func pathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid contact id")
		return uuid.Nil, false
	}
	return id, true
}

// importRequest accepts either pasted spreadsheet text (csv) or structured
// rows (rows), so the UI can offer "paste from Excel" and a programmatic path
// from the same endpoint.
type importRequest struct {
	CSV                 string    `json:"csv"`
	Rows                []BulkRow `json:"rows"`
	DefaultRelationship string    `json:"default_relationship"`
	UpdateExisting      bool      `json:"update_existing"`
	DryRun              bool      `json:"dry_run"`
}

// bulkImport handles POST /contacts/import.
//
// Partial success is the contract: valid rows are saved and invalid ones come
// back with the line number that failed, because a bursar pasting 300 rows
// must not have the whole import rejected over one typo.
func (h *Handler) bulkImport(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	actorID, _ := middleware.GetStaffID(r)

	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}

	rows := req.Rows
	if strings.TrimSpace(req.CSV) != "" {
		parsed, err := ParseCSV(req.CSV)
		if err != nil {
			httputil.RespondBadRequest(w, "INVALID_IMPORT", err.Error())
			return
		}
		rows = parsed
	}
	if len(rows) == 0 {
		httputil.RespondBadRequest(w, "INVALID_IMPORT", "nothing to import: paste rows or send a rows array")
		return
	}
	if len(rows) > maxBulkRows {
		httputil.RespondBadRequest(w, "INVALID_IMPORT",
			fmt.Sprintf("that is %d rows; the limit is %d per import", len(rows), maxBulkRows))
		return
	}

	// A dry run validates everything and saves nothing, so the UI can show
	// "12 ready, 2 need fixing, 1 already saved" before the user commits.
	if req.DryRun {
		preview, err := h.dryRun(r, tenantID, rows, req.DefaultRelationship)
		if err != nil {
			httputil.RespondInternalError(w, err)
			return
		}
		httputil.RespondOK(w, preview)
		return
	}

	result, err := h.service.Import(r.Context(), tenantID, actorID, rows, req.UpdateExisting, req.DefaultRelationship)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, result)
}

// Preview is the dry-run report: one entry per row, valid or not.
type Preview struct {
	Line     int      `json:"line"`
	Name     string   `json:"name,omitempty"`
	Phone    string   `json:"phone,omitempty"`
	PhoneRaw string   `json:"phone_raw,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Valid    bool     `json:"valid"`
	Message  string   `json:"message,omitempty"`
	// Duplicate marks a phone that is already saved in this school, so the UI
	// can say what an import will actually do before it does it.
	Duplicate bool   `json:"duplicate"`
	Existing  string `json:"existing_name,omitempty"`
}

type PreviewResult struct {
	Total      int       `json:"total"`
	Valid      int       `json:"valid"`
	Invalid    int       `json:"invalid"`
	Duplicates int       `json:"duplicates"`
	Rows       []Preview `json:"rows"`
}

// previewRows validates rows without touching the database.
//
// The three counters are a partition of the rows: every row lands in exactly
// one of ready / already saved / needs fixing. Getting this wrong is how an
// import dialog ends up claiming "1 + 1 + 1" for a two-row paste.
func previewRows(rows []BulkRow, defaultRelationship string) PreviewResult {
	out := PreviewResult{Total: len(rows), Rows: make([]Preview, 0, len(rows))}
	seen := make(map[string]int, len(rows))

	for _, row := range rows {
		p := Preview{Line: row.Line, Name: strings.TrimSpace(row.FullName), PhoneRaw: row.Phone}

		in, err := validateRow(row, defaultRelationship)
		if err != nil {
			p.Message = err.Error()
			out.Rows = append(out.Rows, p)
			continue
		}

		p.Phone = in.Phone
		p.Tags = in.Tags
		p.Valid = true

		if firstLine, dup := seen[in.Phone]; dup {
			p.Duplicate = true
			p.Message = fmt.Sprintf("duplicate of row %d in this file", firstLine)
		}
		seen[in.Phone] = row.Line

		out.Rows = append(out.Rows, p)
	}

	out.recompute()
	return out
}

// recompute derives the ready / duplicate / invalid counters from the rows so
// they always sum to the number of rows. Call it again after adding
// book-level duplicate marks.
func (r *PreviewResult) recompute() {
	r.Valid, r.Invalid, r.Duplicates = 0, 0, 0
	for _, p := range r.Rows {
		switch {
		case !p.Valid:
			r.Invalid++
		case p.Duplicate:
			r.Duplicates++
		default:
			r.Valid++
		}
	}
}

// dryRun runs previewRows and then marks the rows whose phone is already saved
// in this school, so the import dialog can say whether a row will be created
// or merged instead of finding out after committing.
func (h *Handler) dryRun(r *http.Request, tenantID uuid.UUID, rows []BulkRow, defaultRelationship string) (*PreviewResult, error) {
	result := previewRows(rows, defaultRelationship)

	phones := make([]string, 0, len(rows))
	for _, p := range result.Rows {
		if p.Phone != "" {
			phones = append(phones, p.Phone)
		}
	}
	if len(phones) == 0 {
		return &result, nil
	}

	existing, err := h.service.NamesByPhone(r.Context(), tenantID, phones)
	if err != nil {
		return nil, err
	}

	for i := range result.Rows {
		p := &result.Rows[i]
		if !p.Valid || p.Phone == "" {
			continue
		}
		if name, ok := existing[p.Phone]; ok {
			p.Duplicate = true
			p.Existing = name
			if p.Message == "" {
				// Already in the school's book: with update_existing this is a
				// merge, otherwise the row is reported as skipped.
				p.Message = "already saved as " + name
			}
		}
	}

	// Book-level duplicates change the partition, so derive the counters again.
	result.recompute()
	return &result, nil
}
