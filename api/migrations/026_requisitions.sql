-- 026_requisitions.sql
-- Purchase Requisitions & Approval Workflow (EPIC 8)

-- Purchase requisitions: staff submits → HoD approves → Principal/Bursar approves
CREATE TABLE IF NOT EXISTS purchase_requisitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    requisition_no VARCHAR(30) NOT NULL,
    title VARCHAR(255) NOT NULL,
    department VARCHAR(100),
    requested_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    required_by DATE,
    justification TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'hod_approved', 'approved', 'rejected', 'cancelled', 'ordered')),
    hod_approved_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    hod_approved_at TIMESTAMPTZ,
    approved_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    rejection_reason TEXT,
    total_estimate_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, requisition_no)
);

CREATE INDEX IF NOT EXISTS idx_requisitions_tenant_status ON purchase_requisitions (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_requisitions_department ON purchase_requisitions (tenant_id, department);

CREATE TRIGGER set_purchase_requisitions_updated_at
    BEFORE UPDATE ON purchase_requisitions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE purchase_requisitions ENABLE ROW LEVEL SECURITY;

CREATE POLICY "requisitions_select_policy" ON purchase_requisitions
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "requisitions_insert_policy" ON purchase_requisitions
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "requisitions_update_policy" ON purchase_requisitions
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "requisitions_delete_policy" ON purchase_requisitions
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Requisition line items
CREATE TABLE IF NOT EXISTS requisition_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    requisition_id UUID NOT NULL REFERENCES purchase_requisitions(id) ON DELETE CASCADE,
    item_name VARCHAR(255) NOT NULL,
    description TEXT,
    quantity INT NOT NULL CHECK (quantity > 0),
    unit VARCHAR(30),
    estimated_unit_cost_cents BIGINT NOT NULL DEFAULT 0,
    estimated_total_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_requisition_items_req ON requisition_items (requisition_id);

ALTER TABLE requisition_items ENABLE ROW LEVEL SECURITY;

CREATE POLICY "requisition_items_select_policy" ON requisition_items
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "requisition_items_insert_policy" ON requisition_items
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "requisition_items_update_policy" ON requisition_items
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "requisition_items_delete_policy" ON requisition_items
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);