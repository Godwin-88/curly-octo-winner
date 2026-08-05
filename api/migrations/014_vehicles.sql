-- 014_vehicles.sql
-- Transport: Vehicles (Kenya number plates)

CREATE TABLE IF NOT EXISTS vehicles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    registration VARCHAR(20) NOT NULL,
    make VARCHAR(50) NOT NULL,
    model VARCHAR(50) NOT NULL,
    capacity INT NOT NULL DEFAULT 14 CHECK (capacity > 0),
    year INT,
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'maintenance', 'retired')),
    insurance_expiry DATE,
    inspection_expiry DATE,
    driver_id UUID REFERENCES staff(id) ON DELETE SET NULL,
    driver_name VARCHAR(100),
    driver_phone VARCHAR(20),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, registration)
);

CREATE INDEX IF NOT EXISTS idx_vehicles_tenant ON vehicles (tenant_id);
CREATE INDEX IF NOT EXISTS idx_vehicles_status ON vehicles (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_vehicles_registration ON vehicles (tenant_id, registration);

CREATE TRIGGER set_vehicles_updated_at
    BEFORE UPDATE ON vehicles
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE vehicles ENABLE ROW LEVEL SECURITY;

CREATE POLICY "vehicles_select_policy" ON vehicles
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "vehicles_insert_policy" ON vehicles
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "vehicles_update_policy" ON vehicles
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "vehicles_delete_policy" ON vehicles
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);