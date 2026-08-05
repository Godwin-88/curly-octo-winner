-- 010_assessments.sql
-- Formative Assessment (EPIC 2.3)

CREATE TABLE IF NOT EXISTS assessments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    learner_id UUID NOT NULL REFERENCES learners(id) ON DELETE CASCADE,
    sub_strand_id UUID NOT NULL REFERENCES sub_strands(id) ON DELETE CASCADE,
    rubric_level INT NOT NULL CHECK (rubric_level BETWEEN 1 AND 4),
    note TEXT,
    evidence_urls TEXT[] NOT NULL DEFAULT '{}',
    teacher_id UUID NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
    term INT NOT NULL CHECK (term IN (1, 2, 3)),
    year INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_assessments_learner ON assessments (learner_id);
CREATE INDEX IF NOT EXISTS idx_assessments_sub_strand ON assessments (sub_strand_id);
CREATE INDEX IF NOT EXISTS idx_assessments_tenant_learner ON assessments (tenant_id, learner_id);
CREATE INDEX IF NOT EXISTS idx_assessments_tenant_term ON assessments (tenant_id, term, year);
CREATE INDEX IF NOT EXISTS idx_assessments_teacher ON assessments (tenant_id, teacher_id);

CREATE TRIGGER set_assessments_updated_at
    BEFORE UPDATE ON assessments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE assessments ENABLE ROW LEVEL SECURITY;

CREATE POLICY "assessments_select_policy" ON assessments
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "assessments_insert_policy" ON assessments
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "assessments_update_policy" ON assessments
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "assessments_delete_policy" ON assessments
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Assessment summary view for report cards
CREATE OR REPLACE VIEW assessment_summary AS
SELECT
    a.tenant_id,
    a.learner_id,
    l.full_name AS learner_name,
    l.grade,
    l.stream,
    a.sub_strand_id,
    s.name AS sub_strand_name,
    s.kicd_code AS sub_strand_code,
    str.name AS strand_name,
    la.name AS learning_area,
    a.rubric_level,
    CASE a.rubric_level
        WHEN 1 THEN 'Below Expectation'
        WHEN 2 THEN 'Approaching Expectation'
        WHEN 3 THEN 'Meeting Expectation'
        WHEN 4 THEN 'Exceeding Expectation'
    END AS rubric_label,
    a.note,
    a.term,
    a.year,
    a.created_at
FROM assessments a
JOIN learners l ON l.id = a.learner_id AND l.tenant_id = a.tenant_id
JOIN sub_strands s ON s.id = a.sub_strand_id AND s.tenant_id = a.tenant_id
JOIN strands str ON str.id = s.strand_id AND str.tenant_id = a.tenant_id
JOIN learning_areas la ON la.id = str.learning_area_id AND la.tenant_id = a.tenant_id;

CREATE INDEX IF NOT EXISTS idx_assessment_summary_learner ON assessment_summary (tenant_id, learner_id, term, year);