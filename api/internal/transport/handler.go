package transport

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/shule360/api/internal/middleware"
	"github.com/shule360/api/pkg/httputil"
)

// Handler contains the HTTP handlers for the transport API.
type Handler struct {
	service *Service
}

// NewHandler creates a new transport handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Mount registers all transport routes under the provided router.
func (h *Handler) Mount(r chi.Router) {
	r.Route("/vehicles", func(r chi.Router) {
		r.Get("/", h.listVehicles)
		r.Post("/", h.createVehicle)
		r.Get("/{id}", h.getVehicle)
		r.Patch("/{id}", h.updateVehicle)
		r.Delete("/{id}", h.deleteVehicle)
	})

	r.Route("/routes", func(r chi.Router) {
		r.Get("/", h.listRoutes)
		r.Post("/", h.createRoute)
		r.Get("/{id}", h.getRoute)
		r.Patch("/{id}", h.updateRoute)
		r.Delete("/{id}", h.deleteRoute)

		r.Get("/{id}/stops", h.listStops)
		r.Post("/{id}/stops", h.createStop)

		r.Get("/{id}/assignments", h.listAssignments)
		r.Post("/{id}/assignments", h.assignLearner)
	})

	r.Delete("/stops/{stopId}", h.deleteStop)
	r.Delete("/assignments/{assignmentId}", h.removeAssignment)

	r.Route("/trips", func(r chi.Router) {
		r.Get("/", h.listTrips)
		r.Post("/", h.createTrip)
		r.Get("/{id}", h.getTrip)
		r.Patch("/{id}", h.updateTrip)
		r.Delete("/{id}", h.deleteTrip)

		r.Post("/{id}/start", h.startTrip)
		r.Post("/{id}/complete", h.completeTrip)
		r.Post("/{id}/cancel", h.cancelTrip)

		r.Get("/{id}/positions", h.listPositions)
		r.Post("/{id}/positions", h.reportPosition)

		r.Get("/{id}/checkins", h.listCheckins)
		r.Post("/{id}/checkins", h.checkIn)
	})
}

// --- Vehicle handlers ---

func (h *Handler) listVehicles(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	status := r.URL.Query().Get("status")
	vehicles, err := h.service.ListVehicles(r.Context(), tenantID, status)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, vehicles)
}

func (h *Handler) createVehicle(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	if req.Registration == "" || req.Make == "" {
		httputil.RespondBadRequest(w, "VALIDATION_FAILED", "registration and make are required")
		return
	}
	vehicle, err := h.service.CreateVehicle(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, vehicle)
}

func (h *Handler) getVehicle(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid vehicle ID")
		return
	}
	vehicle, err := h.service.GetVehicle(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, vehicle)
}

func (h *Handler) updateVehicle(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid vehicle ID")
		return
	}
	var req UpdateVehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	vehicle, err := h.service.UpdateVehicle(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, vehicle)
}

func (h *Handler) deleteVehicle(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid vehicle ID")
		return
	}
	if err := h.service.DeleteVehicle(r.Context(), tenantID, id); err != nil {
		httputil.RespondBadRequest(w, "DELETE_FAILED", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}

// --- Route handlers ---

func (h *Handler) listRoutes(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	routes, err := h.service.ListRoutes(r.Context(), tenantID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, routes)
}

func (h *Handler) createRoute(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	if req.Name == "" {
		httputil.RespondBadRequest(w, "VALIDATION_FAILED", "name is required")
		return
	}
	route, err := h.service.CreateRoute(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, route)
}

func (h *Handler) getRoute(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid route ID")
		return
	}
	route, err := h.service.GetRoute(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, route)
}

func (h *Handler) updateRoute(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid route ID")
		return
	}
	var req UpdateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	route, err := h.service.UpdateRoute(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, route)
}

func (h *Handler) deleteRoute(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid route ID")
		return
	}
	if err := h.service.DeleteRoute(r.Context(), tenantID, id); err != nil {
		httputil.RespondBadRequest(w, "DELETE_FAILED", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}

// --- Stop handlers ---

// listStops reuses getRoute and returns its stops.
func (h *Handler) listStops(w http.ResponseWriter, r *http.Request) {
	route, ok := h.getRouteForStops(w, r)
	if !ok {
		return
	}
	httputil.RespondOK(w, route.Stops)
}

func (h *Handler) createStop(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	routeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid route ID")
		return
	}
	var req StopInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	if req.Name == "" {
		httputil.RespondBadRequest(w, "VALIDATION_FAILED", "name is required")
		return
	}
	stop, err := h.service.CreateStop(r.Context(), tenantID, routeID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, stop)
}

func (h *Handler) deleteStop(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "stopId"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid stop ID")
		return
	}
	if err := h.service.DeleteStop(r.Context(), tenantID, id); err != nil {
		httputil.RespondBadRequest(w, "DELETE_FAILED", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}

// getRouteForStops is a helper that parses the route ID and fetches the route.
func (h *Handler) getRouteForStops(w http.ResponseWriter, r *http.Request) (*Route, bool) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return nil, false
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid route ID")
		return nil, false
	}
	route, err := h.service.GetRoute(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return nil, false
	}
	return route, true
}

// --- Assignment handlers ---

func (h *Handler) listAssignments(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	routeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid route ID")
		return
	}
	assignments, err := h.service.ListAssignments(r.Context(), tenantID, routeID)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, assignments)
}

func (h *Handler) assignLearner(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	routeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid route ID")
		return
	}
	var req CreateAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	assignment, err := h.service.AssignLearner(r.Context(), tenantID, routeID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "ASSIGN_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, assignment)
}

func (h *Handler) removeAssignment(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "assignmentId"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid assignment ID")
		return
	}
	if err := h.service.RemoveAssignment(r.Context(), tenantID, id); err != nil {
		httputil.RespondBadRequest(w, "DELETE_FAILED", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}

// --- Trip handlers ---

func (h *Handler) listTrips(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	status := r.URL.Query().Get("status")
	var onDate *time.Time
	if ds := r.URL.Query().Get("on_date"); ds != "" {
		d, err := time.Parse("2006-01-02", ds)
		if err == nil {
			onDate = &d
		}
	}
	trips, err := h.service.ListTrips(r.Context(), tenantID, status, onDate)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, trips)
}

func (h *Handler) createTrip(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	var req CreateTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	if req.RouteID == uuid.Nil || req.ScheduledDeparture.IsZero() {
		httputil.RespondBadRequest(w, "VALIDATION_FAILED", "route_id and scheduled_departure are required")
		return
	}
	if req.Direction == "" {
		req.Direction = "to_school"
	}
	trip, err := h.service.CreateTrip(r.Context(), tenantID, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CREATE_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, trip)
}

func (h *Handler) getTrip(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid trip ID")
		return
	}
	trip, err := h.service.GetTrip(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondNotFound(w, "NOT_FOUND", err.Error())
		return
	}
	httputil.RespondOK(w, trip)
}

func (h *Handler) updateTrip(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid trip ID")
		return
	}
	var req UpdateTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	trip, err := h.service.UpdateTrip(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "UPDATE_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, trip)
}

func (h *Handler) deleteTrip(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid trip ID")
		return
	}
	if err := h.service.DeleteTrip(r.Context(), tenantID, id); err != nil {
		httputil.RespondBadRequest(w, "DELETE_FAILED", err.Error())
		return
	}
	httputil.RespondNoContent(w)
}

func (h *Handler) startTrip(w http.ResponseWriter, r *http.Request) {
	h.transitionTrip(w, r, "start")
}

func (h *Handler) completeTrip(w http.ResponseWriter, r *http.Request) {
	h.transitionTrip(w, r, "complete")
}

func (h *Handler) cancelTrip(w http.ResponseWriter, r *http.Request) {
	h.transitionTrip(w, r, "cancel")
}

func (h *Handler) transitionTrip(w http.ResponseWriter, r *http.Request, action string) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid trip ID")
		return
	}
	var trip interface{}
	switch action {
	case "start":
		trip, err = h.service.StartTrip(r.Context(), tenantID, id)
	case "complete":
		trip, err = h.service.CompleteTrip(r.Context(), tenantID, id)
	case "cancel":
		trip, err = h.service.CancelTrip(r.Context(), tenantID, id)
	}
	if err != nil {
		httputil.RespondBadRequest(w, "TRANSITION_FAILED", err.Error())
		return
	}
	httputil.RespondOK(w, trip)
}

// --- Position handlers ---

func (h *Handler) listPositions(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid trip ID")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	positions, err := h.service.ListPositions(r.Context(), tenantID, id, limit)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, positions)
}

func (h *Handler) reportPosition(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid trip ID")
		return
	}
	var req ReportPositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	if req.Latitude == 0 && req.Longitude == 0 {
		httputil.RespondBadRequest(w, "VALIDATION_FAILED", "latitude and longitude are required")
		return
	}
	position, err := h.service.ReportPosition(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "REPORT_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, position)
}

// --- Check-in handlers ---

func (h *Handler) listCheckins(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid trip ID")
		return
	}
	checkins, err := h.service.ListCheckins(r.Context(), tenantID, id)
	if err != nil {
		httputil.RespondInternalError(w, err)
		return
	}
	httputil.RespondOK(w, checkins)
}

func (h *Handler) checkIn(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r)
	if !ok {
		httputil.RespondUnauthorized(w, "UNAUTHORIZED", "Tenant ID not found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondBadRequest(w, "INVALID_ID", "Invalid trip ID")
		return
	}
	var req CreateCheckinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondBadRequest(w, "INVALID_REQUEST", "Invalid request body: "+err.Error())
		return
	}
	if req.LearnerID == uuid.Nil || (req.Action != "boarded" && req.Action != "alighted") {
		httputil.RespondBadRequest(w, "VALIDATION_FAILED", "learner_id required and action must be boarded or alighted")
		return
	}
	checkin, err := h.service.CheckIn(r.Context(), tenantID, id, req)
	if err != nil {
		httputil.RespondBadRequest(w, "CHECKIN_FAILED", err.Error())
		return
	}
	httputil.RespondCreated(w, checkin)
}
