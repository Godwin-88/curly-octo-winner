-- 022_staff_profiles.sql
-- Staff Records & Profiles (EPIC 7: Manage Human Resources)

-- Extend staff with HR profile fields (TSC number, qualifications, subjects, employment)
ALTER TABLE staff
    ADD COLUMN IF NOT EXISTS tsc_number VARCHAR(20),
    ADD COLUMN IF NOT EXISTS national_id VARCHAR(20),
    ADD COLUMN IF NOT EXISTS kra_pin VARCHAR(20),
    ADD COLUMN IF NOT EXISTS date_of_birth DATE,
    ADD COLUMN IF NOT EXISTS gender VARCHAR(10),
    ADD COLUMN IF NOT EXISTS department VARCHAR(100),
    ADD COLUMN IF NOT EXISTS job_title VARCHAR(100),
    ADD COLUMN IF NOT EXISTS employment_type VARCHAR(20) DEFAULT 'permanent'
        CHECK (employment_type IN ('permanent', 'contract', 'temporary', 'intern', 'volunteer')),
    ADD COLUMN IF NOT EXISTS hire_date DATE,
    ADD COLUMN IF NOT EXISTS qualifications JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS subjects JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS employment_history JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS emergency_contact JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS bank_details JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS photo_url TEXT;

CREATE INDEX IF NOT EXISTS idx_staff_tsc ON staff (tenant_id, tsc_number);
CREATE INDEX IF NOT EXISTS idx_staff_department ON staff (tenant_id, department);
CREATE INDEX IF NOT EXISTS idx_staff_employment_type ON staff (tenant_id, employment_type);

-- Staff documents (contracts, certificates, appraisal forms)
CREATE TABLE IF NOT EXISTS staff_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    staff_id UUID NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
    doc_type VARCHAR(50) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_url TEXT NOT NULL,
    mime_type VARCHAR(100),
    file_size BIGINT,
    uploaded_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, staff_id, doc_type, file_name)
);

CREATE INDEX IF NOT EXISTS idx_staff_documents_staff ON staff_documents (staff_id);
CREATE INDEX IF NOT EXISTS idx_staff_documents_tenant ON staff_documents (tenant_id);

CREATE TRIGGER set_staff_documents_updated_at
    BEFORE UPDATE ON staff_documents
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE staff_documents ENABLE ROW LEVEL SECURITY;

CREATE POLICY "staff_documents_select_policy" ON staff_documents
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "staff_documents_insert_policy" ON staff_documents
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "staff_documents_update_policy" ON staff_documents
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "staff_documents_delete_policy" ON staff_documents
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);