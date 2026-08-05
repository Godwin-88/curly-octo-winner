package transport

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ListTrips returns trips filtered by status/date, joined with route/vehicle and latest position.
func (s *Service) ListTrips(ctx context.Context, tenantID uuid.UUID, status string, onDate *time.Time) ([]Trip, error) {
	query := `SELECT t.id, t.tenant_id, t.route_id, r.name, t.vehicle_id, v.registration,
		t.direction, t.status, t.scheduled_departure, t.actual_departure, t.actual_arrival,
		t.boarded_count, t.created_by, t.notes, t.created_at, t.updated_at,
		tlp.latitude, tlp.longitude, tlp.reported_at
		FROM trips t
		LEFT JOIN routes r ON r.id = t.route_id AND r.tenant_id = t.tenant_id
		LEFT JOIN vehicles v ON v.id = t.vehicle_id AND v.tenant_id = t.tenant_id
		LEFT JOIN trip_latest_position tlp ON tlp.trip_id = t.id AND tlp.tenant_id = t.tenant_id
		WHERE t.tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2
	if status != "" {
		query += ` AND t.status = $2`
		args = append(args, status)
		argIdx = 3
	}
	if onDate != nil {
		query += ` AND t.scheduled_departure::date = $` + strconv.Itoa(argIdx)
		args = append(args, onDate.Format("2006-01-02"))
	}
	query += ` ORDER BY t.scheduled_departure DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trips []Trip
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		trips = append(trips, *t)
	}
	return trips, rows.Err()
}

func scanTrip(row pgx.Row) (*Trip, error) {
	var t Trip
	err := row.Scan(
		&t.ID, &t.TenantID, &t.RouteID, &t.RouteName, &t.VehicleID, &t.VehicleReg,
		&t.Direction, &t.Status, &t.ScheduledDeparture, &t.ActualDeparture, &t.ActualArrival,
		&t.BoardedCount, &t.CreatedBy, &t.Notes, &t.CreatedAt, &t.UpdatedAt,
		&t.LastLatitude, &t.LastLongitude, &t.LastReported,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetTrip returns a single trip with its latest position.
func (s *Service) GetTrip(ctx context.Context, tenantID, id uuid.UUID) (*Trip, error) {
	query := `SELECT t.id, t.tenant_id, t.route_id, r.name, t.vehicle_id, v.registration,
		t.direction, t.status, t.scheduled_departure, t.actual_departure, t.actual_arrival,
		t.boarded_count, t.created_by, t.notes, t.created_at, t.updated_at,
		tlp.latitude, tlp.longitude, tlp.reported_at
		FROM trips t
		LEFT JOIN routes r ON r.id = t.route_id AND r.tenant_id = t.tenant_id
		LEFT JOIN vehicles v ON v.id = t.vehicle_id AND v.tenant_id = t.tenant_id
		LEFT JOIN trip_latest_position tlp ON tlp.trip_id = t.id AND tlp.tenant_id = t.tenant_id
		WHERE t.tenant_id = $1 AND t.id = $2`
	return scanTrip(s.pool.QueryRow(ctx, query, tenantID, id))
}

// CreateTrip schedules a new trip.
func (s *Service) CreateTrip(ctx context.Context, tenantID uuid.UUID, req CreateTripRequest) (*Trip, error) {
	query := `INSERT INTO trips (tenant_id, route_id, vehicle_id, direction, scheduled_departure, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, route_id, ''::text, vehicle_id, NULL::text,
		direction, status, scheduled_departure, actual_departure, actual_arrival,
		boarded_count, created_by, notes, created_at, updated_at,
		NULL::float8, NULL::float8, NULL::timestamptz`
	return scanTrip(s.pool.QueryRow(ctx, query,
		tenantID, req.RouteID, req.VehicleID, req.Direction, req.ScheduledDeparture, req.Notes))
}

// UpdateTrip partially updates a trip.
func (s *Service) UpdateTrip(ctx context.Context, tenantID, id uuid.UUID, req UpdateTripRequest) (*Trip, error) {
	query := `UPDATE trips SET
		route_id = COALESCE($3, route_id),
		vehicle_id = COALESCE($4, vehicle_id),
		direction = COALESCE($5, direction),
		scheduled_departure = COALESCE($6, scheduled_departure),
		notes = COALESCE($7, notes)
		WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, route_id, ''::text, vehicle_id, NULL::text,
		direction, status, scheduled_departure, actual_departure, actual_arrival,
		boarded_count, created_by, notes, created_at, updated_at,
		NULL::float8, NULL::float8, NULL::timestamptz`
	return scanTrip(s.pool.QueryRow(ctx, query,
		tenantID, id, req.RouteID, req.VehicleID, req.Direction, req.ScheduledDeparture, req.Notes))
}

// StartTrip marks a trip as in_progress and records actual departure.
func (s *Service) StartTrip(ctx context.Context, tenantID, id uuid.UUID) (*Trip, error) {
	query := `UPDATE trips SET status = 'in_progress', actual_departure = COALESCE(actual_departure, now())
		WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, route_id, ''::text, vehicle_id, NULL::text,
		direction, status, scheduled_departure, actual_departure, actual_arrival,
		boarded_count, created_by, notes, created_at, updated_at,
		NULL::float8, NULL::float8, NULL::timestamptz`
	return scanTrip(s.pool.QueryRow(ctx, query, tenantID, id))
}

// CompleteTrip marks a trip as completed and records actual arrival.
func (s *Service) CompleteTrip(ctx context.Context, tenantID, id uuid.UUID) (*Trip, error) {
	query := `UPDATE trips SET status = 'completed', actual_arrival = now()
		WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, route_id, ''::text, vehicle_id, NULL::text,
		direction, status, scheduled_departure, actual_departure, actual_arrival,
		boarded_count, created_by, notes, created_at, updated_at,
		NULL::float8, NULL::float8, NULL::timestamptz`
	return scanTrip(s.pool.QueryRow(ctx, query, tenantID, id))
}

// CancelTrip marks a trip as cancelled.
func (s *Service) CancelTrip(ctx context.Context, tenantID, id uuid.UUID) (*Trip, error) {
	query := `UPDATE trips SET status = 'cancelled'
		WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, route_id, ''::text, vehicle_id, NULL::text,
		direction, status, scheduled_departure, actual_departure, actual_arrival,
		boarded_count, created_by, notes, created_at, updated_at,
		NULL::float8, NULL::float8, NULL::timestamptz`
	return scanTrip(s.pool.QueryRow(ctx, query, tenantID, id))
}

// DeleteTrip removes a trip.
func (s *Service) DeleteTrip(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM trips WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("trip not found")
	}
	return nil
}

// --- Tracking & check-ins ---

// ReportPosition records a GPS ping for a trip.
func (s *Service) ReportPosition(ctx context.Context, tenantID, tripID uuid.UUID, req ReportPositionRequest) (*TripPosition, error) {
	var p TripPosition
	err := s.pool.QueryRow(ctx, `INSERT INTO trip_positions (tenant_id, trip_id, latitude, longitude, speed_kmh, heading_deg, odometer_km)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, trip_id, latitude, longitude, speed_kmh, heading_deg, odometer_km, reported_at`,
		tenantID, tripID, req.Latitude, req.Longitude, req.SpeedKMH, req.HeadingDeg, req.OdometerKM).
		Scan(&p.ID, &p.TripID, &p.Latitude, &p.Longitude, &p.SpeedKMH, &p.HeadingDeg, &p.OdometerKM, &p.ReportedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListPositions returns the position history for a trip.
func (s *Service) ListPositions(ctx context.Context, tenantID, tripID uuid.UUID, limit int) ([]TripPosition, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `SELECT id, trip_id, latitude, longitude, speed_kmh, heading_deg, odometer_km, reported_at
		FROM trip_positions WHERE tenant_id = $1 AND trip_id = $2
		ORDER BY reported_at DESC LIMIT $3`, tenantID, tripID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []TripPosition
	for rows.Next() {
		var p TripPosition
		if err := rows.Scan(&p.ID, &p.TripID, &p.Latitude, &p.Longitude, &p.SpeedKMH, &p.HeadingDeg, &p.OdometerKM, &p.ReportedAt); err != nil {
			return nil, err
		}
		positions = append(positions, p)
	}
	return positions, rows.Err()
}

// CheckIn records a boarded/alighted event for a learner.
func (s *Service) CheckIn(ctx context.Context, tenantID, tripID uuid.UUID, req CreateCheckinRequest) (*TripCheckin, error) {
	var c TripCheckin
	err := s.pool.QueryRow(ctx, `INSERT INTO trip_checkins (tenant_id, trip_id, learner_id, stop_id, action)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (trip_id, learner_id, action) DO NOTHING
		RETURNING id, trip_id, learner_id, stop_id, action, checked_at, sms_notified`,
		tenantID, tripID, req.LearnerID, req.StopID, req.Action).
		Scan(&c.ID, &c.TripID, &c.LearnerID, &c.StopID, &c.Action, &c.CheckedAt, &c.SMSNotified)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("learner already checked in for this action on this trip")
		}
		return nil, err
	}
	// Update boarded count for boarded events.
	if req.Action == "boarded" {
		_, _ = s.pool.Exec(ctx, `UPDATE trips SET boarded_count = boarded_count + 1 WHERE tenant_id = $1 AND id = $2`, tenantID, tripID)
	}
	return &c, nil
}

// ListCheckins returns check-ins for a trip.
func (s *Service) ListCheckins(ctx context.Context, tenantID, tripID uuid.UUID) ([]TripCheckin, error) {
	rows, err := s.pool.Query(ctx, `SELECT tc.id, tc.trip_id, tc.learner_id, l.full_name, tc.stop_id, st.name, tc.action, tc.checked_at, tc.sms_notified
		FROM trip_checkins tc
		JOIN learners l ON l.id = tc.learner_id AND l.tenant_id = tc.tenant_id
		LEFT JOIN stops st ON st.id = tc.stop_id AND st.tenant_id = tc.tenant_id
		WHERE tc.tenant_id = $1 AND tc.trip_id = $2
		ORDER BY tc.checked_at`, tenantID, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checkins []TripCheckin
	for rows.Next() {
		var c TripCheckin
		if err := rows.Scan(&c.ID, &c.TripID, &c.LearnerID, &c.LearnerName, &c.StopID, &c.StopName, &c.Action, &c.CheckedAt, &c.SMSNotified); err != nil {
			return nil, err
		}
		checkins = append(checkins, c)
	}
	return checkins, rows.Err()
}
