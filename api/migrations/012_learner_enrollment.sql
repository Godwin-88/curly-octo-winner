-- 012_learner_enrollment.sql
-- Learner Enrollment & Documents (EPIC 3)

-- Extend learners table with enrollment fields
ALTER TABLE learners
    ADD COLUMN IF NOT EXISTS birth_cert_no VARCHAR(30),
    ADD COLUMN IF NOT EXISTS entry_level VARCHAR(20),
    ADD COLUMN IF NOT EXISTS special_needs BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS admission_date DATE;

CREATE INDEX IF NOT EXISTS idx_learners_active ON learners (tenant_id, is_active);
CREATE INDEX IF NOT EXISTS idx_learners_name ON learners (tenant_id, full_name);

-- Learner documents (birth cert, report cards, medical, etc.)
CREATE TABLE IF NOT EXISTS learner_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    learner_id UUID NOT NULL REFERENCES learners(id) ON DELETE CASCADE,
    doc_type VARCHAR(30) NOT NULL CHECK (doc_type IN ('birth_cert', 'report_card', 'medical', 'transfer', 'other')),
    file_name VARCHAR(255) NOT NULL,
    file_url TEXT NOT NULL,
    mime_type VARCHAR(100),
    file_size BIGINT,
    uploaded_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_learner_documents_learner ON learner_documents (learner_id);
CREATE INDEX IF NOT EXISTS idx_learner_documents_tenant ON learner_documents (tenant_id, learner_id);

CREATE TRIGGER set_learner_documents_updated_at
    BEFORE UPDATE ON learner_documents
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE learner_documents ENABLE ROW LEVEL SECURITY;

CREATE POLICY "learner_documents_select_policy" ON learner_documents
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_documents_insert_policy" ON learner_documents
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_documents_update_policy" ON learner_documents
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_documents_delete_policy" ON learner_documents
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);