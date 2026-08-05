package transport

import (
	"time"

	"github.com/google/uuid"
)

// Vehicle represents a school vehicle.
type Vehicle struct {
	ID               uuid.UUID  `json:"id"`
	TenantID         uuid.UUID  `json:"tenant_id"`
	Registration     string     `json:"registration"`
	Make             string     `json:"make"`
	Model            string     `json:"model"`
	Capacity         int        `json:"capacity"`
	Year             *int       `json:"year,omitempty"`
	Status           string     `json:"status"`
	InsuranceExpiry  *time.Time `json:"insurance_expiry,omitempty"`
	InspectionExpiry *time.Time `json:"inspection_expiry,omitempty"`
	DriverID         *uuid.UUID `json:"driver_id,omitempty"`
	DriverName       *string    `json:"driver_name,omitempty"`
	DriverPhone      *string    `json:"driver_phone,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// Stop represents a route stop.
type Stop struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	RouteID   uuid.UUID `json:"route_id"`
	Name      string    `json:"name"`
	Sequence  int       `json:"sequence"`
	Latitude  *float64  `json:"latitude,omitempty"`
	Longitude *float64  `json:"longitude,omitempty"`
	Landmark  *string   `json:"landmark,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Assignment represents a learner assigned to a route.
type Assignment struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	RouteID   uuid.UUID `json:"route_id"`
	LearnerID uuid.UUID `json:"learner_id"`
	StopID    uuid.UUID `json:"stop_id"`
	Direction string    `json:"direction"`
	CreatedAt time.Time `json:"created_at"`
	// Joined fields for list responses
	LearnerName string `json:"learner_name,omitempty"`
	Grade       string `json:"grade,omitempty"`
	Stream      string `json:"stream,omitempty"`
	StopName    string `json:"stop_name,omitempty"`
}

// Trip represents a single journey on a route.
type Trip struct {
	ID                 uuid.UUID  `json:"id"`
	TenantID           uuid.UUID  `json:"tenant_id"`
	RouteID            uuid.UUID  `json:"route_id"`
	RouteName          string     `json:"route_name,omitempty"`
	VehicleID          *uuid.UUID `json:"vehicle_id,omitempty"`
	VehicleReg         *string    `json:"vehicle_registration,omitempty"`
	Direction          string     `json:"direction"`
	Status             string     `json:"status"`
	ScheduledDeparture time.Time  `json:"scheduled_departure"`
	ActualDeparture    *time.Time `json:"actual_departure,omitempty"`
	ActualArrival      *time.Time `json:"actual_arrival,omitempty"`
	BoardedCount       int        `json:"boarded_count"`
	CreatedBy          *uuid.UUID `json:"created_by,omitempty"`
	Notes              *string    `json:"notes,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	// Latest position (from trip_latest_position view)
	LastLatitude  *float64   `json:"last_latitude,omitempty"`
	LastLongitude *float64   `json:"last_longitude,omitempty"`
	LastReported  *time.Time `json:"last_reported,omitempty"`
}

// TripCheckin represents a boarded/alighted check-in.
type TripCheckin struct {
	ID          uuid.UUID  `json:"id"`
	TripID      uuid.UUID  `json:"trip_id"`
	LearnerID   uuid.UUID  `json:"learner_id"`
	LearnerName string     `json:"learner_name,omitempty"`
	StopID      *uuid.UUID `json:"stop_id,omitempty"`
	StopName    *string    `json:"stop_name,omitempty"`
	Action      string     `json:"action"`
	CheckedAt   time.Time  `json:"checked_at"`
	SMSNotified bool       `json:"sms_notified"`
}

// TripPosition represents a GPS ping.
type TripPosition struct {
	ID         uuid.UUID `json:"id"`
	TripID     uuid.UUID `json:"trip_id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	SpeedKMH   *float32  `json:"speed_kmh,omitempty"`
	HeadingDeg *float32  `json:"heading_deg,omitempty"`
	OdometerKM *float32  `json:"odometer_km,omitempty"`
	ReportedAt time.Time `json:"reported_at"`
}

type CreateVehicleRequest struct {
	Registration     string     `json:"registration"`
	Make             string     `json:"make"`
	Model            string     `json:"model"`
	Capacity         int        `json:"capacity"`
	Year             *int       `json:"year,omitempty"`
	Status           string     `json:"status"`
	InsuranceExpiry  *time.Time `json:"insurance_expiry,omitempty"`
	InspectionExpiry *time.Time `json:"inspection_expiry,omitempty"`
	DriverID         *uuid.UUID `json:"driver_id,omitempty"`
	DriverName       *string    `json:"driver_name,omitempty"`
	DriverPhone      *string    `json:"driver_phone,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
}

type UpdateVehicleRequest struct {
	Registration     *string    `json:"registration,omitempty"`
	Make             *string    `json:"make,omitempty"`
	Model            *string    `json:"model,omitempty"`
	Capacity         *int       `json:"capacity,omitempty"`
	Year             *int       `json:"year,omitempty"`
	Status           *string    `json:"status,omitempty"`
	InsuranceExpiry  *time.Time `json:"insurance_expiry,omitempty"`
	InspectionExpiry *time.Time `json:"inspection_expiry,omitempty"`
	DriverID         *uuid.UUID `json:"driver_id,omitempty"`
	DriverName       *string    `json:"driver_name,omitempty"`
	DriverPhone      *string    `json:"driver_phone,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
}

type CreateRouteRequest struct {
	Name        string      `json:"name"`
	Description *string     `json:"description,omitempty"`
	VehicleID   *uuid.UUID  `json:"vehicle_id,omitempty"`
	Active      *bool       `json:"active,omitempty"`
	Stops       []StopInput `json:"stops,omitempty"`
}

type UpdateRouteRequest struct {
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	VehicleID   *uuid.UUID `json:"vehicle_id,omitempty"`
	Active      *bool      `json:"active,omitempty"`
}

type StopInput struct {
	Name      string   `json:"name"`
	Sequence  int      `json:"sequence"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	Landmark  *string  `json:"landmark,omitempty"`
}

type CreateAssignmentRequest struct {
	LearnerID uuid.UUID `json:"learner_id"`
	StopID    uuid.UUID `json:"stop_id"`
	Direction string    `json:"direction"`
}

type CreateTripRequest struct {
	RouteID            uuid.UUID  `json:"route_id"`
	VehicleID          *uuid.UUID `json:"vehicle_id,omitempty"`
	Direction          string     `json:"direction"`
	ScheduledDeparture time.Time  `json:"scheduled_departure"`
	Notes              *string    `json:"notes,omitempty"`
}

type UpdateTripRequest struct {
	RouteID            *uuid.UUID `json:"route_id,omitempty"`
	VehicleID          *uuid.UUID `json:"vehicle_id,omitempty"`
	Direction          *string    `json:"direction,omitempty"`
	ScheduledDeparture *time.Time `json:"scheduled_departure,omitempty"`
	Notes              *string    `json:"notes,omitempty"`
}

type CreateCheckinRequest struct {
	LearnerID uuid.UUID  `json:"learner_id"`
	StopID    *uuid.UUID `json:"stop_id,omitempty"`
	Action    string     `json:"action"`
}

type ReportPositionRequest struct {
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	SpeedKMH   *float32 `json:"speed_kmh,omitempty"`
	HeadingDeg *float32 `json:"heading_deg,omitempty"`
	OdometerKM *float32 `json:"odometer_km,omitempty"`
}
