-- 013_learner_progression.sql
-- Learner Progression & Transfers (EPIC 3)

-- Learner progression history (grade promotions, retentions)
CREATE TABLE IF NOT EXISTS learner_progressions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    learner_id UUID NOT NULL REFERENCES learners(id) ON DELETE CASCADE,
    from_grade VARCHAR(10) NOT NULL,
    to_grade VARCHAR(10) NOT NULL,
    action VARCHAR(20) NOT NULL CHECK (action IN ('promote', 'retain', 'transfer_out', 'transfer_in')),
    term INT CHECK (term IN (1, 2, 3)),
    year INT NOT NULL,
    approved_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_learner_progressions_learner ON learner_progressions (learner_id);
CREATE INDEX IF NOT EXISTS idx_learner_progressions_tenant ON learner_progressions (tenant_id, year);

CREATE TRIGGER set_learner_progressions_updated_at
    BEFORE UPDATE ON learner_progressions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE learner_progressions ENABLE ROW LEVEL SECURITY;

CREATE POLICY "learner_progressions_select_policy" ON learner_progressions
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_progressions_insert_policy" ON learner_progressions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_progressions_update_policy" ON learner_progressions
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_progressions_delete_policy" ON learner_progressions
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);