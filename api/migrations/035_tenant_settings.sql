-- 035_tenant_settings.sql
-- Per-school configuration: the principal-facing Settings area.
--
-- 1. tenants  — adds the school profile fields a principal maintains
--               (contact details, address, M-Pesa paybill wiring).
-- 2. tenant_settings — one row per school: operational configuration
--               (term/year, attendance window, grading scale, feature
--               toggles, document footers).
-- 3. tenant_integrations — per-school provider credentials. Secrets are stored
--               in secrets_encrypted (AES-256-GCM, sealed by the API) and never
--               returned to the browser; only non-secret `config` is readable.
--               `use_platform_default` lets a school inherit the platform-level
--               environment credentials instead of its own.
--
-- Tenant isolation is enforced in the API (tenant_id comes from the verified
-- JWT, never the request body). RLS policies mirror the rest of the schema as
-- defence in depth.

-- ============================================================
-- 1. School profile
-- ============================================================
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS phone TEXT,
    ADD COLUMN IF NOT EXISTS email TEXT,
    ADD COLUMN IF NOT EXISTS address TEXT,
    ADD COLUMN IF NOT EXISTS county TEXT,
    ADD COLUMN IF NOT EXISTS mpesa_shortcode VARCHAR(20),
    ADD COLUMN IF NOT EXISTS mpesa_account_basis VARCHAR(20) NOT NULL DEFAULT 'phone',
    ADD COLUMN IF NOT EXISTS mpesa_callback_url TEXT;

-- ============================================================
-- 2. Operational settings (one row per tenant, created on demand)
-- ============================================================
CREATE TABLE IF NOT EXISTS tenant_settings (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,

    -- Academic
    current_term VARCHAR(50),
    current_academic_year VARCHAR(20),
    grading_scale JSONB NOT NULL DEFAULT '[{"min":80,"label":"A","points":4},{"min":70,"label":"B","points":3},{"min":60,"label":"C","points":2},{"min":50,"label":"D","points":1},{"min":0,"label":"E","points":0}]'::jsonb,

    -- Attendance
    attendance_time VARCHAR(5) NOT NULL DEFAULT '07:30',
    attendance_deadline VARCHAR(5) NOT NULL DEFAULT '09:00',

    -- Documents & receipts
    report_card_footer TEXT,
    receipt_footer TEXT,

    -- Feature toggles (a school switches off what it does not use)
    mpesa_enabled BOOLEAN NOT NULL DEFAULT false,
    sms_enabled BOOLEAN NOT NULL DEFAULT true,
    whatsapp_enabled BOOLEAN NOT NULL DEFAULT false,
    require_parent_consent BOOLEAN NOT NULL DEFAULT true,
    require_staff_approval_on_transfer BOOLEAN NOT NULL DEFAULT true,

    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID REFERENCES staff(id) ON DELETE SET NULL
);

CREATE OR REPLACE TRIGGER set_tenant_settings_updated_at
    BEFORE UPDATE ON tenant_settings
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;

CREATE POLICY "tenant_settings_select_policy" ON tenant_settings
    FOR SELECT USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
CREATE POLICY "tenant_settings_insert_policy" ON tenant_settings
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);
CREATE POLICY "tenant_settings_update_policy" ON tenant_settings
    FOR UPDATE USING (tenant_id = current_setting('app.tenant_id', true)::uuid);

-- ============================================================
-- 3. Integrations (credentials per school)
-- ============================================================
CREATE TABLE IF NOT EXISTS tenant_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider VARCHAR(40) NOT NULL,          -- mpesa | africastalking | whatsapp | backblaze | groq | upstash
    label VARCHAR(100),

    is_enabled BOOLEAN NOT NULL DEFAULT true,
    -- When true the school inherits the platform credentials from the API
    -- environment and the credentials below are ignored.
    use_platform_default BOOLEAN NOT NULL DEFAULT true,

    config JSONB NOT NULL DEFAULT '{}'::jsonb,   -- non-secret values, safe to read
    secrets_encrypted TEXT,                       -- "v1:" + base64(nonce|ciphertext)

    last_tested_at TIMESTAMPTZ,
    last_test_status VARCHAR(20),                 -- ok | failed | never
    last_test_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by UUID REFERENCES staff(id) ON DELETE SET NULL,

    UNIQUE (tenant_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_tenant_integrations_tenant
    ON tenant_integrations (tenant_id, provider);

CREATE OR REPLACE TRIGGER set_tenant_integrations_updated_at
    BEFORE UPDATE ON tenant_integrations
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE tenant_integrations ENABLE ROW LEVEL SECURITY;

CREATE POLICY "tenant_integrations_select_policy" ON tenant_integrations
    FOR SELECT USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
CREATE POLICY "tenant_integrations_insert_policy" ON tenant_integrations
    FOR INSERT WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);
CREATE POLICY "tenant_integrations_update_policy" ON tenant_integrations
    FOR UPDATE USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
CREATE POLICY "tenant_integrations_delete_policy" ON tenant_integrations
    FOR DELETE USING (tenant_id = current_setting('app.tenant_id', true)::uuid);
