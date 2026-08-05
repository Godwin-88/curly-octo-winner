-- 016_trips.sql
-- Transport: Trips (live runs) & tracking check-ins

CREATE TABLE IF NOT EXISTS trips (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    route_id UUID NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    vehicle_id UUID REFERENCES vehicles(id) ON DELETE SET NULL,
    direction VARCHAR(10) NOT NULL DEFAULT 'to_school'
        CHECK (direction IN ('to_school', 'from_school')),
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled'
        CHECK (status IN ('scheduled', 'in_progress', 'completed', 'cancelled')),
    scheduled_departure TIMESTAMPTZ NOT NULL,
    actual_departure TIMESTAMPTZ,
    actual_arrival TIMESTAMPTZ,
    boarded_count INT NOT NULL DEFAULT 0,
    created_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trips_tenant ON trips (tenant_id);
CREATE INDEX IF NOT EXISTS idx_trips_route ON trips (tenant_id, route_id);
CREATE INDEX IF NOT EXISTS idx_trips_status ON trips (tenant_id, status, scheduled_departure);
CREATE INDEX IF NOT EXISTS idx_trips_date ON trips (tenant_id, scheduled_departure);

CREATE TRIGGER set_trips_updated_at
    BEFORE UPDATE ON trips
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE trips ENABLE ROW LEVEL SECURITY;

CREATE POLICY "trips_select_policy" ON trips
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "trips_insert_policy" ON trips
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "trips_update_policy" ON trips
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "trips_delete_policy" ON trips
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Known positions reported during a trip (GPS pings)
CREATE TABLE IF NOT EXISTS trip_positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    speed_kmh REAL,
    heading_deg REAL,
    odometer_km REAL,
    reported_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trip_positions_trip ON trip_positions (tenant_id, trip_id, reported_at);

ALTER TABLE trip_positions ENABLE ROW LEVEL SECURITY;

CREATE POLICY "trip_positions_select_policy" ON trip_positions
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "trip_positions_insert_policy" ON trip_positions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "trip_positions_update_policy" ON trip_positions
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "trip_positions_delete_policy" ON trip_positions
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Boarding / alighting check-ins for learners on a trip
CREATE TABLE IF NOT EXISTS trip_checkins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    learner_id UUID NOT NULL REFERENCES learners(id) ON DELETE CASCADE,
    stop_id UUID REFERENCES stops(id) ON DELETE SET NULL,
    action VARCHAR(10) NOT NULL CHECK (action IN ('boarded', 'alighted')),
    checked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sms_notified BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (trip_id, learner_id, action)
);

CREATE INDEX IF NOT EXISTS idx_trip_checkins_trip ON trip_checkins (tenant_id, trip_id);
CREATE INDEX IF NOT EXISTS idx_trip_checkins_learner ON trip_checkins (tenant_id, learner_id);

ALTER TABLE trip_checkins ENABLE ROW LEVEL SECURITY;

CREATE POLICY "trip_checkins_select_policy" ON trip_checkins
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "trip_checkins_insert_policy" ON trip_checkins
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "trip_checkins_update_policy" ON trip_checkins
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "trip_checkins_delete_policy" ON trip_checkins
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Latest position per trip view (for the live tracking dashboard)
CREATE OR REPLACE VIEW trip_latest_position AS
SELECT DISTINCT ON (trip_id)
    tp.trip_id,
    tp.tenant_id,
    tp.latitude,
    tp.longitude,
    tp.speed_kmh,
    tp.heading_deg,
    tp.reported_at
FROM trip_positions tp
ORDER BY tp.trip_id, tp.reported_at DESC;