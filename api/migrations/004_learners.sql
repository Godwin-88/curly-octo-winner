-- 004_learners.sql
-- Learners table

CREATE TABLE IF NOT EXISTS learners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    upi VARCHAR(20) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    date_of_birth DATE,
    grade VARCHAR(10) NOT NULL,
    stream VARCHAR(50),
    photo_url TEXT,
    guardian_ids UUID[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, upi)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_learners_tenant_created ON learners (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_learners_grade_stream ON learners (tenant_id, grade, stream);
CREATE INDEX IF NOT EXISTS idx_learners_upi ON learners (tenant_id, upi);

-- Trigger
CREATE TRIGGER set_learners_updated_at
    BEFORE UPDATE ON learners
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- RLS
ALTER TABLE learners ENABLE ROW LEVEL SECURITY;

CREATE POLICY "learners_select_policy" ON learners
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learners_insert_policy" ON learners
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learners_update_policy" ON learners
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learners_delete_policy" ON learners
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);