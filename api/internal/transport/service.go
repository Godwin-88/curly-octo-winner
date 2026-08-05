package transport

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles transport domain operations.
type Service struct {
	pool *pgxpool.Pool
}

// NewService creates a transport service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// --- Vehicle operations ---

const vehicleColumns = `id, tenant_id, registration, make, model, capacity, year, status,
	insurance_expiry, inspection_expiry, driver_id, driver_name, driver_phone, notes, created_at, updated_at`

func scanVehicle(row pgx.Row) (*Vehicle, error) {
	var v Vehicle
	err := row.Scan(
		&v.ID, &v.TenantID, &v.Registration, &v.Make, &v.Model, &v.Capacity, &v.Year, &v.Status,
		&v.InsuranceExpiry, &v.InspectionExpiry, &v.DriverID, &v.DriverName, &v.DriverPhone, &v.Notes,
		&v.CreatedAt, &v.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVehicles returns vehicles optionally filtered by status.
func (s *Service) ListVehicles(ctx context.Context, tenantID uuid.UUID, status string) ([]Vehicle, error) {
	query := fmt.Sprintf(`SELECT %s FROM vehicles WHERE tenant_id = $1`, vehicleColumns)
	args := []any{tenantID}
	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicles []Vehicle
	for rows.Next() {
		v, err := scanVehicle(rows)
		if err != nil {
			return nil, err
		}
		vehicles = append(vehicles, *v)
	}
	return vehicles, rows.Err()
}

// GetVehicle returns a vehicle by ID.
func (s *Service) GetVehicle(ctx context.Context, tenantID, id uuid.UUID) (*Vehicle, error) {
	query := fmt.Sprintf(`SELECT %s FROM vehicles WHERE tenant_id = $1 AND id = $2`, vehicleColumns)
	return scanVehicle(s.pool.QueryRow(ctx, query, tenantID, id))
}

// CreateVehicle inserts a new vehicle.
func (s *Service) CreateVehicle(ctx context.Context, tenantID uuid.UUID, req CreateVehicleRequest) (*Vehicle, error) {
	status := req.Status
	if status == "" {
		status = "active"
	}
	query := fmt.Sprintf(`INSERT INTO vehicles (tenant_id, registration, make, model, capacity, year, status,
		insurance_expiry, inspection_expiry, driver_id, driver_name, driver_phone, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING %s`, vehicleColumns)
	return scanVehicle(s.pool.QueryRow(ctx, query,
		tenantID, req.Registration, req.Make, req.Model, req.Capacity, req.Year, status,
		req.InsuranceExpiry, req.InspectionExpiry, req.DriverID, req.DriverName, req.DriverPhone, req.Notes,
	))
}

// UpdateVehicle partially updates a vehicle.
func (s *Service) UpdateVehicle(ctx context.Context, tenantID, id uuid.UUID, req UpdateVehicleRequest) (*Vehicle, error) {
	// Ensure the vehicle exists.
	if _, err := s.GetVehicle(ctx, tenantID, id); err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`UPDATE vehicles SET
		registration = COALESCE($3, registration),
		make = COALESCE($4, make),
		model = COALESCE($5, model),
		capacity = COALESCE($6, capacity),
		year = COALESCE($7, year),
		status = COALESCE($8, status),
		insurance_expiry = COALESCE($9, insurance_expiry),
		inspection_expiry = COALESCE($10, inspection_expiry),
		driver_id = COALESCE($11, driver_id),
		driver_name = COALESCE($12, driver_name),
		driver_phone = COALESCE($13, driver_phone),
		notes = COALESCE($14, notes)
		WHERE tenant_id = $1 AND id = $2
		RETURNING %s`, vehicleColumns)
	return scanVehicle(s.pool.QueryRow(ctx, query,
		tenantID, id,
		req.Registration, req.Make, req.Model, req.Capacity, req.Year, req.Status,
		req.InsuranceExpiry, req.InspectionExpiry, req.DriverID, req.DriverName, req.DriverPhone, req.Notes,
	))
}

// DeleteVehicle removes a vehicle.
func (s *Service) DeleteVehicle(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM vehicles WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("vehicle not found")
	}
	return nil
}

// --- Route operations ---

type Route struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	VehicleID   *uuid.UUID `json:"vehicle_id,omitempty"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Stops       []Stop     `json:"stops,omitempty"`
}

const routeColumns = `id, tenant_id, name, description, vehicle_id, active, created_at, updated_at`

// ListRoutes returns routes with their stops.
func (s *Service) ListRoutes(ctx context.Context, tenantID uuid.UUID) ([]Route, error) {
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`SELECT %s FROM routes WHERE tenant_id = $1 ORDER BY name`, routeColumns), tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []Route
	for rows.Next() {
		var r Route
		var vehicleID *uuid.UUID
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.Description, &vehicleID, &r.Active, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.VehicleID = vehicleID
		routes = append(routes, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attach stops.
	stops, err := s.pool.Query(ctx, `SELECT id, tenant_id, route_id, name, sequence, latitude, longitude, landmark, created_at FROM stops WHERE tenant_id = $1 ORDER BY route_id, sequence`, tenantID)
	if err != nil {
		return nil, err
	}
	defer stops.Close()
	stopMap := make(map[uuid.UUID][]Stop)
	for stops.Next() {
		var st Stop
		if err := stops.Scan(&st.ID, &st.TenantID, &st.RouteID, &st.Name, &st.Sequence, &st.Latitude, &st.Longitude, &st.Landmark, &st.CreatedAt); err != nil {
			return nil, err
		}
		stopMap[st.RouteID] = append(stopMap[st.RouteID], st)
	}
	if err := stops.Err(); err != nil {
		return nil, err
	}
	for i := range routes {
		if ss, ok := stopMap[routes[i].ID]; ok {
			routes[i].Stops = ss
		}
	}
	return routes, nil
}

// GetRoute returns a single route with stops.
func (s *Service) GetRoute(ctx context.Context, tenantID, id uuid.UUID) (*Route, error) {
	var r Route
	var vehicleID *uuid.UUID
	err := s.pool.QueryRow(ctx, fmt.Sprintf(`SELECT %s FROM routes WHERE tenant_id = $1 AND id = $2`, routeColumns), tenantID, id).
		Scan(&r.ID, &r.TenantID, &r.Name, &r.Description, &vehicleID, &r.Active, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	r.VehicleID = vehicleID

	rows, err := s.pool.Query(ctx, `SELECT id, tenant_id, route_id, name, sequence, latitude, longitude, landmark, created_at FROM stops WHERE tenant_id = $1 AND route_id = $2 ORDER BY sequence`, tenantID, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stops []Stop
	for rows.Next() {
		var st Stop
		if err := rows.Scan(&st.ID, &st.TenantID, &st.RouteID, &st.Name, &st.Sequence, &st.Latitude, &st.Longitude, &st.Landmark, &st.CreatedAt); err != nil {
			return nil, err
		}
		stops = append(stops, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	r.Stops = stops
	return &r, nil
}

// CreateRoute inserts a route and its stops in a transaction.
func (s *Service) CreateRoute(ctx context.Context, tenantID uuid.UUID, req CreateRouteRequest) (*Route, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var r Route
	var vehicleID *uuid.UUID
	err = tx.QueryRow(ctx, fmt.Sprintf(`INSERT INTO routes (tenant_id, name, description, vehicle_id, active)
		VALUES ($1, $2, $3, $4, COALESCE($5, true))
		RETURNING %s`, routeColumns), tenantID, req.Name, req.Description, req.VehicleID, req.Active).
		Scan(&r.ID, &r.TenantID, &r.Name, &r.Description, &vehicleID, &r.Active, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	r.VehicleID = vehicleID

	// Insert stops if provided.
	var stops []Stop
	for _, si := range req.Stops {
		var st Stop
		err := tx.QueryRow(ctx, `INSERT INTO stops (tenant_id, route_id, name, sequence, latitude, longitude, landmark)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, tenant_id, route_id, name, sequence, latitude, longitude, landmark, created_at`,
			tenantID, r.ID, si.Name, si.Sequence, si.Latitude, si.Longitude, si.Landmark).
			Scan(&st.ID, &st.TenantID, &st.RouteID, &st.Name, &st.Sequence, &st.Latitude, &st.Longitude, &st.Landmark, &st.CreatedAt)
		if err != nil {
			return nil, err
		}
		stops = append(stops, st)
	}
	r.Stops = stops

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &r, nil
}

// UpdateRoute partially updates a route.
func (s *Service) UpdateRoute(ctx context.Context, tenantID, id uuid.UUID, req UpdateRouteRequest) (*Route, error) {
	if _, err := s.GetRoute(ctx, tenantID, id); err != nil {
		return nil, err
	}
	var r Route
	var vehicleID *uuid.UUID
	err := s.pool.QueryRow(ctx, fmt.Sprintf(`UPDATE routes SET
		name = COALESCE($3, name),
		description = COALESCE($4, description),
		vehicle_id = COALESCE($5, vehicle_id),
		active = COALESCE($6, active)
		WHERE tenant_id = $1 AND id = $2
		RETURNING %s`, routeColumns), tenantID, id, req.Name, req.Description, req.VehicleID, req.Active).
		Scan(&r.ID, &r.TenantID, &r.Name, &r.Description, &vehicleID, &r.Active, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	r.VehicleID = vehicleID
	return &r, nil
}

// DeleteRoute removes a route and cascades to stops/assignments.
func (s *Service) DeleteRoute(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM routes WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("route not found")
	}
	return nil
}

// --- Stop operations ---

// CreateStop adds a stop to an existing route.
func (s *Service) CreateStop(ctx context.Context, tenantID, routeID uuid.UUID, req StopInput) (*Stop, error) {
	var st Stop
	err := s.pool.QueryRow(ctx, `INSERT INTO stops (tenant_id, route_id, name, sequence, latitude, longitude, landmark)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, tenant_id, route_id, name, sequence, latitude, longitude, landmark, created_at`,
		tenantID, routeID, req.Name, req.Sequence, req.Latitude, req.Longitude, req.Landmark).
		Scan(&st.ID, &st.TenantID, &st.RouteID, &st.Name, &st.Sequence, &st.Latitude, &st.Longitude, &st.Landmark, &st.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// DeleteStop removes a stop.
func (s *Service) DeleteStop(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM stops WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("stop not found")
	}
	return nil
}

// --- Route assignments ---

// ListAssignments returns learners assigned to a route.
func (s *Service) ListAssignments(ctx context.Context, tenantID, routeID uuid.UUID) ([]Assignment, error) {
	rows, err := s.pool.Query(ctx, `SELECT ra.id, ra.tenant_id, ra.route_id, ra.learner_id, ra.stop_id, ra.direction, ra.created_at,
		l.full_name AS learner_name, l.grade, l.stream, st.name AS stop_name
		FROM route_assignments ra
		JOIN learners l ON l.id = ra.learner_id AND l.tenant_id = ra.tenant_id
		LEFT JOIN stops st ON st.id = ra.stop_id AND st.tenant_id = ra.tenant_id
		WHERE ra.tenant_id = $1 AND ra.route_id = $2
		ORDER BY l.full_name`, tenantID, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []Assignment
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.ID, &a.TenantID, &a.RouteID, &a.LearnerID, &a.StopID, &a.Direction, &a.CreatedAt,
			&a.LearnerName, &a.Grade, &a.Stream, &a.StopName); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}

// AssignLearner assigns a learner to a route.
func (s *Service) AssignLearner(ctx context.Context, tenantID, routeID uuid.UUID, req CreateAssignmentRequest) (*Assignment, error) {
	var a Assignment
	err := s.pool.QueryRow(ctx, `INSERT INTO route_assignments (tenant_id, route_id, learner_id, stop_id, direction)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, route_id, learner_id, direction) DO NOTHING
		RETURNING id, tenant_id, route_id, learner_id, stop_id, direction, created_at`,
		tenantID, routeID, req.LearnerID, req.StopID, req.Direction).
		Scan(&a.ID, &a.TenantID, &a.RouteID, &a.LearnerID, &a.StopID, &a.Direction, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("learner already assigned to this route/direction")
		}
		return nil, err
	}
	return &a, nil
}

// RemoveAssignment deletes a learner from a route.
func (s *Service) RemoveAssignment(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM route_assignments WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("assignment not found")
	}
	return nil
}
