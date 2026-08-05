-- 015_routes.sql
-- Transport: Routes & Stops

CREATE TABLE IF NOT EXISTS routes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    vehicle_id UUID REFERENCES vehicles(id) ON DELETE SET NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_routes_tenant ON routes (tenant_id);
CREATE INDEX IF NOT EXISTS idx_routes_vehicle ON routes (tenant_id, vehicle_id);

CREATE TRIGGER set_routes_updated_at
    BEFORE UPDATE ON routes
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE routes ENABLE ROW LEVEL SECURITY;

CREATE POLICY "routes_select_policy" ON routes
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "routes_insert_policy" ON routes
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "routes_update_policy" ON routes
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "routes_delete_policy" ON routes
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Stops (ordered stations along a route)
CREATE TABLE IF NOT EXISTS stops (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    route_id UUID NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    sequence INT NOT NULL DEFAULT 0,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    landmark VARCHAR(150),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, route_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_stops_tenant ON stops (tenant_id);
CREATE INDEX IF NOT EXISTS idx_stops_route ON stops (tenant_id, route_id);

-- Route assignments (learners on a route)
CREATE TABLE IF NOT EXISTS route_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    route_id UUID NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    learner_id UUID NOT NULL REFERENCES learners(id) ON DELETE CASCADE,
    stop_id UUID NOT NULL REFERENCES stops(id) ON DELETE CASCADE,
    direction VARCHAR(10) NOT NULL DEFAULT 'to_school'
        CHECK (direction IN ('to_school', 'from_school', 'both')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, route_id, learner_id, direction)
);

CREATE INDEX IF NOT EXISTS idx_route_assignments_tenant ON route_assignments (tenant_id);
CREATE INDEX IF NOT EXISTS idx_route_assignments_route ON route_assignments (tenant_id, route_id);
CREATE INDEX IF NOT EXISTS idx_route_assignments_learner ON route_assignments (tenant_id, learner_id);

ALTER TABLE stops ENABLE ROW LEVEL SECURITY;
ALTER TABLE route_assignments ENABLE ROW LEVEL SECURITY;

CREATE POLICY "stops_select_policy" ON stops
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "stops_insert_policy" ON stops
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "stops_update_policy" ON stops
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "stops_delete_policy" ON stops
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "route_assignments_select_policy" ON route_assignments
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "route_assignments_insert_policy" ON route_assignments
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "route_assignments_update_policy" ON route_assignments
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "route_assignments_delete_policy" ON route_assignments
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);