-- 009_curriculum.sql
-- CBC Curriculum Structure (EPIC 2.1)

CREATE TABLE IF NOT EXISTS learning_areas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    kicd_code VARCHAR(20) NOT NULL,
    grade_level VARCHAR(20) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, kicd_code)
);

CREATE INDEX IF NOT EXISTS idx_learning_areas_tenant_grade ON learning_areas (tenant_id, grade_level);

CREATE TRIGGER set_learning_areas_updated_at
    BEFORE UPDATE ON learning_areas
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE learning_areas ENABLE ROW LEVEL SECURITY;

CREATE POLICY "learning_areas_select_policy" ON learning_areas
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learning_areas_insert_policy" ON learning_areas
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learning_areas_update_policy" ON learning_areas
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learning_areas_delete_policy" ON learning_areas
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Strands
CREATE TABLE IF NOT EXISTS strands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    learning_area_id UUID NOT NULL REFERENCES learning_areas(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    kicd_code VARCHAR(20) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, learning_area_id, kicd_code)
);

CREATE INDEX IF NOT EXISTS idx_strands_learning_area ON strands (learning_area_id);
CREATE INDEX IF NOT EXISTS idx_strands_tenant ON strands (tenant_id);

CREATE TRIGGER set_strands_updated_at
    BEFORE UPDATE ON strands
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE strands ENABLE ROW LEVEL SECURITY;

CREATE POLICY "strands_select_policy" ON strands
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "strands_insert_policy" ON strands
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "strands_update_policy" ON strands
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "strands_delete_policy" ON strands
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Sub-strands
CREATE TABLE IF NOT EXISTS sub_strands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    strand_id UUID NOT NULL REFERENCES strands(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    kicd_code VARCHAR(20) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, strand_id, kicd_code)
);

CREATE INDEX IF NOT EXISTS idx_sub_strands_strand ON sub_strands (strand_id);
CREATE INDEX IF NOT EXISTS idx_sub_strands_tenant ON sub_strands (tenant_id);

CREATE TRIGGER set_sub_strands_updated_at
    BEFORE UPDATE ON sub_strands
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE sub_strands ENABLE ROW LEVEL SECURITY;

CREATE POLICY "sub_strands_select_policy" ON sub_strands
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "sub_strands_insert_policy" ON sub_strands
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "sub_strands_update_policy" ON sub_strands
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "sub_strands_delete_policy" ON sub_strands
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Learning Outcomes (SLOs)
CREATE TABLE IF NOT EXISTS learning_outcomes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sub_strand_id UUID NOT NULL REFERENCES sub_strands(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_learning_outcomes_sub_strand ON learning_outcomes (sub_strand_id);
CREATE INDEX IF NOT EXISTS idx_learning_outcomes_tenant ON learning_outcomes (tenant_id);

CREATE TRIGGER set_learning_outcomes_updated_at
    BEFORE UPDATE ON learning_outcomes
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE learning_outcomes ENABLE ROW LEVEL SECURITY;

CREATE POLICY "learning_outcomes_select_policy" ON learning_outcomes
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learning_outcomes_insert_policy" ON learning_outcomes
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learning_outcomes_update_policy" ON learning_outcomes
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learning_outcomes_delete_policy" ON learning_outcomes
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Core Competencies
CREATE TABLE IF NOT EXISTS core_competencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    kicd_code VARCHAR(20) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, kicd_code)
);

CREATE INDEX IF NOT EXISTS idx_core_competencies_tenant ON core_competencies (tenant_id);

CREATE TRIGGER set_core_competencies_updated_at
    BEFORE UPDATE ON core_competencies
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE core_competencies ENABLE ROW LEVEL SECURITY;

CREATE POLICY "core_competencies_select_policy" ON core_competencies
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "core_competencies_insert_policy" ON core_competencies
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "core_competencies_update_policy" ON core_competencies
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "core_competencies_delete_policy" ON core_competencies
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Values
CREATE TABLE IF NOT EXISTS values (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    kicd_code VARCHAR(20) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, kicd_code)
);

CREATE INDEX IF NOT EXISTS idx_values_tenant ON values (tenant_id);

CREATE TRIGGER set_values_updated_at
    BEFORE UPDATE ON values
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE values ENABLE ROW LEVEL SECURITY;

CREATE POLICY "values_select_policy" ON values
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "values_insert_policy" ON values
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "values_update_policy" ON values
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "values_delete_policy" ON values
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
