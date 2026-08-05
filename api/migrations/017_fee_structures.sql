-- 017_fee_structures.sql
-- Finance: Fee structures (per-grade schedules) and items

-- Fee structure: a per-grade, per-term schedule template
CREATE TABLE IF NOT EXISTS fee_structures (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    grade VARCHAR(10) NOT NULL,
    term INT NOT NULL CHECK (term BETWEEN 1 AND 3),
    year INT NOT NULL,
    total_cents BIGINT NOT NULL DEFAULT 0 CHECK (total_cents >= 0),
    active BOOLEAN NOT NULL DEFAULT true,
    notes TEXT,
    created_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, grade, term, year)
);

CREATE INDEX IF NOT EXISTS idx_fee_structures_tenant ON fee_structures (tenant_id);
CREATE INDEX IF NOT EXISTS idx_fee_structures_grade ON fee_structures (tenant_id, grade, term, year);

CREATE TRIGGER set_fee_structures_updated_at
    BEFORE UPDATE ON fee_structures
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE fee_structures ENABLE ROW LEVEL SECURITY;

CREATE POLICY "fee_structures_select_policy" ON fee_structures
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "fee_structures_insert_policy" ON fee_structures
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "fee_structures_update_policy" ON fee_structures
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "fee_structures_delete_policy" ON fee_structures
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Individual line items on a fee structure (tuition, caution, transport, activity, boarding)
CREATE TABLE IF NOT EXISTS fee_structure_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    fee_structure_id UUID NOT NULL REFERENCES fee_structures(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents >= 0),
    item_type VARCHAR(20) NOT NULL DEFAULT 'other'
        CHECK (item_type IN ('tuition', 'caution', 'transport', 'activity', 'boarding', 'other')),
    is_optional BOOLEAN NOT NULL DEFAULT false,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (fee_structure_id, name)
);

CREATE INDEX IF NOT EXISTS idx_fee_structure_items_structure ON fee_structure_items (tenant_id, fee_structure_id);

ALTER TABLE fee_structure_items ENABLE ROW LEVEL SECURITY;

CREATE POLICY "fee_structure_items_select_policy" ON fee_structure_items
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "fee_structure_items_insert_policy" ON fee_structure_items
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "fee_structure_items_update_policy" ON fee_structure_items
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "fee_structure_items_delete_policy" ON fee_structure_items
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);