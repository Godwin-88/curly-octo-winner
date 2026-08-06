-- 020_report_cards.sql
-- CBC Report Card Generator (EPIC 6)

-- Report cards: one per learner per term/year
CREATE TABLE IF NOT EXISTS report_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    learner_id UUID NOT NULL REFERENCES learners(id) ON DELETE CASCADE,
    term INT NOT NULL CHECK (term IN (1, 2, 3)),
    year INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'final')),
    overall_rating INT CHECK (overall_rating BETWEEN 1 AND 4),
    core_competency_remarks JSONB NOT NULL DEFAULT '{}',
    teacher_comments JSONB NOT NULL DEFAULT '{}',
    attendance_summary JSONB NOT NULL DEFAULT '{}',
    generated_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, learner_id, term, year)
);

CREATE INDEX IF NOT EXISTS idx_report_cards_learner ON report_cards (learner_id);
CREATE INDEX IF NOT EXISTS idx_report_cards_tenant_term ON report_cards (tenant_id, term, year);

CREATE TRIGGER set_report_cards_updated_at
    BEFORE UPDATE ON report_cards
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE report_cards ENABLE ROW LEVEL SECURITY;

CREATE POLICY "report_cards_select_policy" ON report_cards
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "report_cards_insert_policy" ON report_cards
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "report_cards_update_policy" ON report_cards
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "report_cards_delete_policy" ON report_cards
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Report card line items: per learning area / strand ratings and comments
CREATE TABLE IF NOT EXISTS report_card_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    report_card_id UUID NOT NULL REFERENCES report_cards(id) ON DELETE CASCADE,
    learning_area_id UUID REFERENCES learning_areas(id) ON DELETE SET NULL,
    strand_id UUID REFERENCES strands(id) ON DELETE SET NULL,
    sub_strand_id UUID REFERENCES sub_strands(id) ON DELETE SET NULL,
    rubric_level INT CHECK (rubric_level BETWEEN 1 AND 4),
    comment TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (report_card_id, sub_strand_id)
);

CREATE INDEX IF NOT EXISTS idx_report_card_items_report ON report_card_items (report_card_id);
CREATE INDEX IF NOT EXISTS idx_report_card_items_tenant ON report_card_items (tenant_id);

ALTER TABLE report_card_items ENABLE ROW LEVEL SECURITY;

CREATE POLICY "report_card_items_select_policy" ON report_card_items
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "report_card_items_insert_policy" ON report_card_items
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "report_card_items_update_policy" ON report_card_items
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "report_card_items_delete_policy" ON report_card_items
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);