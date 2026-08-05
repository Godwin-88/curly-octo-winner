-- 002_staff_auth.sql
-- Staff table with roles

CREATE TYPE staff_role AS ENUM ('super_admin', 'principal', 'teacher', 'bursar', 'transport_manager', 'hr');

CREATE TABLE IF NOT EXISTS staff (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(15),
    role staff_role NOT NULL DEFAULT 'teacher',
    supabase_user_id UUID,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, email)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_staff_tenant_created ON staff (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_staff_email ON staff (email);
CREATE INDEX IF NOT EXISTS idx_staff_role ON staff (tenant_id, role);

-- Trigger
CREATE TRIGGER set_staff_updated_at
    BEFORE UPDATE ON staff
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- RLS
ALTER TABLE staff ENABLE ROW LEVEL SECURITY;

CREATE POLICY "staff_select_policy" ON staff
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "staff_insert_policy" ON staff
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "staff_update_policy" ON staff
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "staff_delete_policy" ON staff
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);