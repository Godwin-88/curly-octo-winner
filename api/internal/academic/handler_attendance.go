package academic

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/shule360/api/internal/academic/attendance"
	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/apperr"
	"github.com/shule360/api/pkg/httputil"
)

// markAttendanceBulk saves a whole register in one transaction, then runs the
// absence alert once for the day instead of once per learner.
func (h *Handler) markAttendanceBulk(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}

	var req attendance.BulkMarkRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if len(req.Marks) == 0 {
		httputil.RespondBadRequest(w, "NO_MARKS", "Mark at least one learner before saving.")
		return
	}

	// Record who marked the register rather than trusting the body: the client
	// controls every other field of this request, so a marked_by it supplied
	// would be an audit trail anyone could forge.
	if staffID, ok := middleware.GetStaffID(r); ok {
		for i := range req.Marks {
			req.Marks[i].MarkedBy = staffID
		}
	}

	result, err := h.attendanceSvc.MarkBulk(r.Context(), tenantID, req)
	if err != nil {
		respondAttendanceErr(w, err)
		return
	}

	// The register is saved whatever happens to the alerts; what was done
	// about them is reported beside it rather than swallowed.
	if req.Notify {
		staffID, _ := middleware.GetStaffID(r)
		alerts, err := h.absenceAlertSvc.AlertAbsences(r.Context(), tenantID, staffID, req.Date.Time)
		var refusal *apperr.ValidationError
		switch {
		case err == nil:
			result.Alerts = alerts
		case errors.As(err, &refusal):
			result.AlertError = refusal.Message
		default:
			slog.Error("absence alerts failed", "tenant_id", tenantID, "date", result.Date, "error", err)
			result.AlertError = "The register was saved, but the parents could not be texted. Try again from the register."
		}
	}

	httputil.RespondOK(w, result)
}

func respondAttendanceErr(w http.ResponseWriter, err error) {
	if errors.Is(err, attendance.ErrInvalidStatus) {
		httputil.RespondBadRequest(w, "INVALID_STATUS", err.Error())
		return
	}
	apperr.Respond(w, err)
}

// attendanceSummary powers the "today at a glance" panel.
func (h *Handler) attendanceSummary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	date, ok := queryDate(w, r)
	if !ok {
		return
	}

	rows, err := h.attendanceSvc.ListSummariesByDate(r.Context(), tenantID, date)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}

	// Derived here rather than in SQL so the panel and any future consumer
	// cannot disagree about what "present today" counts.
	summary := map[string]int{
		"present": 0, "absent": 0, "late": 0, "excused": 0, "marked": 0,
	}
	for _, row := range rows {
		summary[string(row.Status)]++
		summary["marked"]++
	}
	httputil.RespondOK(w, summary)
}

// listChronicAbsenteeism lists learners below the attendance threshold.
func (h *Handler) listChronicAbsenteeism(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	threshold, term, year := chronicParams(r)

	rows, err := h.attendanceSvc.ChronicAbsenteeism(r.Context(), tenantID, threshold, term, year)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, rows)
}

// termParams reads optional term/year query parameters, falling back to the
// current school term. Out-of-range values are ignored rather than rejected: a
// stale bookmark should show the current term, not an error page.
func termParams(r *http.Request) (int, int) {
	term := currentTerm()
	year := currentYear()
	if v := r.URL.Query().Get("term"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 1 && parsed <= 3 {
			term = parsed
		}
	}
	if v := r.URL.Query().Get("year"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 2000 && parsed < 3000 {
			year = parsed
		}
	}
	return term, year
}

func chronicParams(r *http.Request) (float64, int, int) {
	threshold := 75.0
	if v := r.URL.Query().Get("threshold"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed > 0 && parsed <= 100 {
			threshold = parsed
		}
	}
	term, year := termParams(r)
	return threshold, term, year
}

// currentTerm guesses the school term from the calendar.
//
// Kenyan schools run Term 1 January-April, Term 2 May-August and Term 3
// September-December.
func currentTerm() int {
	switch time.Now().Month() {
	case time.January, time.February, time.March, time.April:
		return 1
	case time.May, time.June, time.July, time.August:
		return 2
	case time.September, time.October, time.November, time.December:
		return 3
	default:
		return 1
	}
}

func currentYear() int { return time.Now().Year() }

// queryDate reads a required YYYY-MM-DD date parameter.
func queryDate(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		httputil.RespondBadRequest(w, "MISSING_PARAM", "date parameter is required (YYYY-MM-DD)")
		return time.Time{}, false
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_DATE", "date must be in YYYY-MM-DD format")
		return time.Time{}, false
	}
	return date, true
}
