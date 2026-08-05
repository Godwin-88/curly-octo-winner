-- 018_invoices.sql
-- Finance: Invoices, invoice items, and discounts/waivers

-- Invoice: a learner's fee bill for a term/year, snapshot of their fee structure
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    learner_id UUID NOT NULL REFERENCES learners(id) ON DELETE CASCADE,
    fee_structure_id UUID REFERENCES fee_structures(id) ON DELETE SET NULL,
    invoice_number VARCHAR(32) NOT NULL,
    term INT NOT NULL CHECK (term BETWEEN 1 AND 3),
    year INT NOT NULL,
    issue_date DATE NOT NULL DEFAULT CURRENT_DATE,
    due_date DATE,
    total_cents BIGINT NOT NULL CHECK (total_cents >= 0),
    discount_cents BIGINT NOT NULL DEFAULT 0 CHECK (discount_cents >= 0),
    paid_cents BIGINT NOT NULL DEFAULT 0 CHECK (paid_cents >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'unpaid'
        CHECK (status IN ('draft', 'unpaid', 'partially_paid', 'paid', 'overdue', 'void')),
    notes TEXT,
    created_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, invoice_number)
);

CREATE INDEX IF NOT EXISTS idx_invoices_tenant ON invoices (tenant_id);
CREATE INDEX IF NOT EXISTS idx_invoices_learner ON invoices (tenant_id, learner_id, term, year);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices (tenant_id, status, due_date);

CREATE TRIGGER set_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;

CREATE POLICY "invoices_select_policy" ON invoices
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "invoices_insert_policy" ON invoices
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "invoices_update_policy" ON invoices
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "invoices_delete_policy" ON invoices
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Snapshot line items on an invoice (copied from the fee structure at issue time)
CREATE TABLE IF NOT EXISTS invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents >= 0),
    item_type VARCHAR(20) NOT NULL DEFAULT 'other'
        CHECK (item_type IN ('tuition', 'caution', 'transport', 'activity', 'boarding', 'other')),
    is_optional BOOLEAN NOT NULL DEFAULT false,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice ON invoice_items (tenant_id, invoice_id);

ALTER TABLE invoice_items ENABLE ROW LEVEL SECURITY;

CREATE POLICY "invoice_items_select_policy" ON invoice_items
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "invoice_items_insert_policy" ON invoice_items
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "invoice_items_update_policy" ON invoice_items
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "invoice_items_delete_policy" ON invoice_items
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Discounts / waivers applied to a learner's invoice (partial or full, sibling discounts)
CREATE TABLE IF NOT EXISTS discounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    amount_cents BIGINT NOT NULL CHECK (amount_cents >= 0),
    discount_type VARCHAR(20) NOT NULL DEFAULT 'other'
        CHECK (discount_type IN ('scholarship', 'sibling', 'waiver', 'other')),
    reason TEXT,
    approved_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_discounts_invoice ON discounts (tenant_id, invoice_id);

ALTER TABLE discounts ENABLE ROW LEVEL SECURITY;

CREATE POLICY "discounts_select_policy" ON discounts
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "discounts_insert_policy" ON discounts
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "discounts_update_policy" ON discounts
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "discounts_delete_policy" ON discounts
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);