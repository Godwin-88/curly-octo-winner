-- 003_guardians.sql
-- Guardians (parents) table

CREATE TABLE IF NOT EXISTS guardians (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    phone_primary VARCHAR(15) NOT NULL,
    phone_wa VARCHAR(15),
    wa_opted_in BOOLEAN NOT NULL DEFAULT false,
    wa_opted_in_at TIMESTAMPTZ,
    email VARCHAR(255),
    is_sms_opted_out BOOLEAN NOT NULL DEFAULT false,
    is_transport_enrolled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_guardians_tenant_created ON guardians (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_guardians_phone ON guardians (tenant_id, phone_primary);
CREATE INDEX IF NOT EXISTS idx_guardians_wa_optin ON guardians (tenant_id, wa_opted_in);

-- Trigger
CREATE TRIGGER set_guardians_updated_at
    BEFORE UPDATE ON guardians
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- RLS
ALTER TABLE guardians ENABLE ROW LEVEL SECURITY;

CREATE POLICY "guardians_select_policy" ON guardians
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "guardians_insert_policy" ON guardians
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "guardians_update_policy" ON guardians
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "guardians_delete_policy" ON guardians
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);