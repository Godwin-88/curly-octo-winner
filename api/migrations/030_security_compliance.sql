-- 030_security_compliance.sql
-- EPIC 10: Digital Security & Compliance
-- RBAC, JWT refresh tokens, audit logs, Kenya Data Protection Act, parent consent

-- ============================================================
-- 1. Permissions & RBAC
-- ============================================================

-- Permission catalog (static seed data follows)
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(100) NOT NULL UNIQUE,        -- e.g. 'comms.send', 'finance.view', 'learners.manage'
    description TEXT,
    category VARCHAR(50),                     -- e.g. 'comms', 'finance', 'learners', 'admin'
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Role-to-permission mapping (role is text to avoid coupling to the staff_role enum)
CREATE TABLE IF NOT EXISTS role_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role VARCHAR(50) NOT NULL,                -- matches staff_role enum values as text
    permission_code VARCHAR(100) NOT NULL REFERENCES permissions(code) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (role, permission_code)
);

-- JWT refresh token / session store
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    staff_id UUID NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,  -- SHA-256 of the refresh token
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    replaced_by UUID,                         -- token rotation chain
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_staff ON refresh_tokens (staff_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_tenant ON refresh_tokens (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires ON refresh_tokens (expires_at);

ALTER TABLE refresh_tokens ENABLE ROW LEVEL SECURITY;

CREATE POLICY "refresh_tokens_select_policy" ON refresh_tokens
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- ============================================================
-- 2. Audit Log
-- ============================================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_staff_id UUID REFERENCES staff(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,             -- e.g. 'UPDATE', 'DELETE', 'CREATE', 'VIEW'
    entity_type VARCHAR(100) NOT NULL,        -- e.g. 'learner', 'payment', 'message'
    entity_id UUID,
    details JSONB DEFAULT '{}'::jsonb,        -- before/after snapshot or metadata
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs (tenant_id, entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs (tenant_id, actor_staff_id);

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY "audit_logs_select_policy" ON audit_logs
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "audit_logs_insert_policy" ON audit_logs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- ============================================================
-- 3. Kenya Data Protection Act — Data Processing Register
-- ============================================================

CREATE TABLE IF NOT EXISTS data_processing_register (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    activity VARCHAR(255) NOT NULL,           -- e.g. 'SMS fee reminders to parents'
    purpose TEXT NOT NULL,                    -- e.g. 'Fee collection communication'
    legal_basis VARCHAR(100) NOT NULL,        -- e.g. 'Consent', 'Contract', 'Legal obligation'
    data_subjects TEXT NOT NULL,              -- e.g. 'Guardians, Learners'
    categories_of_data TEXT,                  -- e.g. 'Name, Phone, M-Pesa transaction details'
    retention_period VARCHAR(100),            -- e.g. '7 years (statutory)'
    transfer_to_third_parties BOOLEAN NOT NULL DEFAULT false,
    third_parties TEXT,                       -- e.g. 'Africa''s Talking, Safaricom Daraja'
    security_measures TEXT,                   -- e.g. 'TLS in transit, AES-256 at rest, RLS'
    registered_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_dpr_tenant ON data_processing_register (tenant_id, created_at DESC);

ALTER TABLE data_processing_register ENABLE ROW LEVEL SECURITY;

CREATE POLICY "dpr_select_policy" ON data_processing_register
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "dpr_insert_policy" ON data_processing_register
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "dpr_update_policy" ON data_processing_register
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "dpr_delete_policy" ON data_processing_register
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE TRIGGER set_dpr_updated_at
    BEFORE UPDATE ON data_processing_register
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- ============================================================
-- 4. Kenya Data Protection Act — Consent Management
-- ============================================================

CREATE TYPE consent_type AS ENUM ('whatsapp_opt_in', 'sms', 'data_processing', 'marketing', 'transport_opt_in');

CREATE TABLE IF NOT EXISTS consent_agreements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    guardian_id UUID REFERENCES guardians(id) ON DELETE CASCADE,
    consent_type consent_type NOT NULL,
    granted BOOLEAN NOT NULL DEFAULT false,
    granted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    ip_address INET,
    source VARCHAR(50) DEFAULT 'portal',      -- 'portal', 'sms', 'whatsapp', 'admin'
    consent_version VARCHAR(20),              -- e.g. 'v1.0'
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (guardian_id, consent_type)
);

CREATE INDEX IF NOT EXISTS idx_consent_tenant ON consent_agreements (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_consent_guardian ON consent_agreements (guardian_id, consent_type);

ALTER TABLE consent_agreements ENABLE ROW LEVEL SECURITY;

CREATE POLICY "consent_select_policy" ON consent_agreements
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "consent_insert_policy" ON consent_agreements
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "consent_update_policy" ON consent_agreements
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "consent_delete_policy" ON consent_agreements
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- ============================================================
-- 5. Data Subject Rights — Erasure (Right to be Forgotten)
-- ============================================================

CREATE TABLE IF NOT EXISTS erasure_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subject_type VARCHAR(20) NOT NULL,        -- 'guardian' | 'learner'
    subject_id UUID NOT NULL,                 -- guardians.id or learners.id
    requested_by VARCHAR(255) NOT NULL,       -- requester name / contact
    request_type VARCHAR(30) NOT NULL DEFAULT 'erasure',  -- erasure | access | rectification | restriction
    status VARCHAR(30) NOT NULL DEFAULT 'pending',        -- pending | in_progress | completed | denied
    details TEXT,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_erasure_tenant ON erasure_requests (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_erasure_subject ON erasure_requests (tenant_id, subject_id);

ALTER TABLE erasure_requests ENABLE ROW LEVEL SECURITY;

CREATE POLICY "erasure_select_policy" ON erasure_requests
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "erasure_insert_policy" ON erasure_requests
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "erasure_update_policy" ON erasure_requests
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- ============================================================
-- Seed: Default permission catalog (deny-by-default RBAC)
-- ============================================================

INSERT INTO permissions (code, description, category) VALUES
    ('comms.view', 'View communications', 'comms'),
    ('comms.send', 'Send SMS/WhatsApp campaigns', 'comms'),
    ('comms.manage', 'Manage templates & inbox', 'comms'),
    ('academic.view', 'View curriculum & assessments', 'academic'),
    ('academic.manage', 'Manage curriculum & assessments', 'academic'),
    ('learners.view', 'View learner records', 'learners'),
    ('learners.manage', 'Manage learner enrollment & records', 'learners'),
    ('transport.view', 'View transport', 'transport'),
    ('transport.manage', 'Manage vehicles, routes & trips', 'transport'),
    ('finance.view', 'View finance', 'finance'),
    ('finance.manage', 'Manage fees, invoices & payments', 'finance'),
    ('finance.collect', 'Collect payments (M-Pesa)', 'finance'),
    ('hr.view', 'View HR records', 'hr'),
    ('hr.manage', 'Manage staff & payroll', 'hr'),
    ('procurement.view', 'View procurement', 'procurement'),
    ('procurement.manage', 'Manage suppliers & POs', 'procurement'),
    ('intelligence.view', 'View analytics & AI', 'intelligence'),
    ('import.manage', 'Import NEMIS records', 'nemis'),
    ('security.manage', 'Manage RBAC, audit & compliance', 'security'),
    ('tenant.manage', 'Manage tenant settings (super admin)', 'tenant')
ON CONFLICT (code) DO NOTHING;

-- Default role-permission matrix:
-- Principal: full access
-- Bursar: finance + comms (billing) + view reports
-- Teacher: academic + learners (view) + comms (view)
-- Transport Manager: transport
-- HR: hr
-- Super Admin: everything (implied by security.manage + tenant.manage, granted explicitly below)
INSERT INTO role_permissions (role, permission_code)
SELECT 'principal', code FROM permissions
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role, permission_code) VALUES
    ('super_admin', 'security.manage'),
    ('super_admin', 'tenant.manage'),
    ('super_admin', 'comms.view'), ('super_admin', 'comms.send'), ('super_admin', 'comms.manage'),
    ('super_admin', 'academic.view'), ('super_admin', 'academic.manage'),
    ('super_admin', 'learners.view'), ('super_admin', 'learners.manage'),
    ('super_admin', 'transport.view'), ('super_admin', 'transport.manage'),
    ('super_admin', 'finance.view'), ('super_admin', 'finance.manage'), ('super_admin', 'finance.collect'),
    ('super_admin', 'hr.view'), ('super_admin', 'hr.manage'),
    ('super_admin', 'procurement.view'), ('super_admin', 'procurement.manage'),
    ('super_admin', 'intelligence.view'), ('super_admin', 'import.manage')
ON CONFLICT DO NOTHING;

-- Teacher
INSERT INTO role_permissions (role, permission_code) VALUES
    ('teacher', 'academic.view'), ('teacher', 'academic.manage'),
    ('teacher', 'learners.view'), ('teacher', 'comms.view'),
    ('teacher', 'intelligence.view')
ON CONFLICT DO NOTHING;

-- Bursar
INSERT INTO role_permissions (role, permission_code) VALUES
    ('bursar', 'finance.view'), ('bursar', 'finance.manage'), ('bursar', 'finance.collect'),
    ('bursar', 'comms.view'), ('bursar', 'comms.send'),
    ('bursar', 'learners.view'), ('bursar', 'intelligence.view')
ON CONFLICT DO NOTHING;

-- Transport Manager
INSERT INTO role_permissions (role, permission_code) VALUES
    ('transport_manager', 'transport.view'), ('transport_manager', 'transport.manage'),
    ('transport_manager', 'learners.view'), ('transport_manager', 'comms.view')
ON CONFLICT DO NOTHING;

-- HR
INSERT INTO role_permissions (role, permission_code) VALUES
    ('hr', 'hr.view'), ('hr', 'hr.manage'),
    ('hr', 'learners.view'), ('hr', 'comms.view')
ON CONFLICT DO NOTHING;

-- ============================================================
-- Seed: Sample consent & data protection records (dev)
-- ============================================================

-- Sample data processing register entries for the default tenant
INSERT INTO data_processing_register (
    tenant_id, activity, purpose, legal_basis, data_subjects, categories_of_data,
    retention_period, transfer_to_third_parties, third_parties, security_measures
)
SELECT id, 'SMS and WhatsApp fee reminders', 'Fee collection communication to guardians',
       'Consent / Contract', 'Guardians, Learners', 'Name, Phone, Balance',
       '7 years', true, 'Africa''s Talking, Meta WhatsApp', 'TLS in transit, AES-256 at rest, RLS'
FROM tenants WHERE slug = 'jua-kali'
ON CONFLICT DO NOTHING;

INSERT INTO data_processing_register (
    tenant_id, activity, purpose, legal_basis, data_subjects, categories_of_data,
    retention_period, transfer_to_third_parties, third_parties, security_measures
)
SELECT id, 'M-Pesa payment processing', 'Process school fees via STK push',
       'Contract', 'Guardians', 'Name, Phone, M-Pesa transaction reference',
       '7 years', true, 'Safaricom Daraja', 'TLS in transit, PCI-DSS controls'
FROM tenants WHERE slug = 'jua-kali'
ON CONFLICT DO NOTHING;
