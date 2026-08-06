-- 027_purchase_orders.sql
-- Purchase Orders & Goods Receipt Notes (EPIC 8)

-- Purchase orders: generated from approved requisitions, sent to supplier
CREATE TABLE IF NOT EXISTS purchase_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    po_number VARCHAR(30) NOT NULL,
    requisition_id UUID REFERENCES purchase_requisitions(id) ON DELETE SET NULL,
    supplier_id UUID NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT,
    order_date DATE NOT NULL DEFAULT CURRENT_DATE,
    expected_delivery DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'sent', 'partially_received', 'received', 'cancelled')),
    total_amount_cents BIGINT NOT NULL DEFAULT 0,
    notes TEXT,
    created_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, po_number)
);

CREATE INDEX IF NOT EXISTS idx_purchase_orders_tenant_status ON purchase_orders (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_purchase_orders_supplier ON purchase_orders (supplier_id);

CREATE TRIGGER set_purchase_orders_updated_at
    BEFORE UPDATE ON purchase_orders
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE purchase_orders ENABLE ROW LEVEL SECURITY;

CREATE POLICY "purchase_orders_select_policy" ON purchase_orders
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "purchase_orders_insert_policy" ON purchase_orders
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "purchase_orders_update_policy" ON purchase_orders
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "purchase_orders_delete_policy" ON purchase_orders
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Purchase order line items
CREATE TABLE IF NOT EXISTS purchase_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    item_name VARCHAR(255) NOT NULL,
    description TEXT,
    quantity INT NOT NULL CHECK (quantity > 0),
    unit VARCHAR(30),
    unit_cost_cents BIGINT NOT NULL DEFAULT 0,
    total_cost_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_po_items_po ON purchase_order_items (purchase_order_id);

ALTER TABLE purchase_order_items ENABLE ROW LEVEL SECURITY;

CREATE POLICY "po_items_select_policy" ON purchase_order_items
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "po_items_insert_policy" ON purchase_order_items
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "po_items_update_policy" ON purchase_order_items
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "po_items_delete_policy" ON purchase_order_items
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Goods Receipt Notes: receiving officer confirms delivery; quantity verified
CREATE TABLE IF NOT EXISTS goods_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    grn_number VARCHAR(30) NOT NULL,
    purchase_order_id UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    supplier_id UUID NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    received_date DATE NOT NULL DEFAULT CURRENT_DATE,
    received_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'received'
        CHECK (status IN ('received', 'partial', 'rejected')),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, grn_number)
);

CREATE INDEX IF NOT EXISTS idx_goods_receipts_tenant ON goods_receipts (tenant_id, purchase_order_id);

CREATE TRIGGER set_goods_receipts_updated_at
    BEFORE UPDATE ON goods_receipts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE goods_receipts ENABLE ROW LEVEL SECURITY;

CREATE POLICY "goods_receipts_select_policy" ON goods_receipts
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "goods_receipts_insert_policy" ON goods_receipts
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "goods_receipts_update_policy" ON goods_receipts
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "goods_receipts_delete_policy" ON goods_receipts
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- GRN line items: quantity verified against PO
CREATE TABLE IF NOT EXISTS goods_receipt_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    goods_receipt_id UUID NOT NULL REFERENCES goods_receipts(id) ON DELETE CASCADE,
    po_item_id UUID REFERENCES purchase_order_items(id) ON DELETE SET NULL,
    item_name VARCHAR(255) NOT NULL,
    quantity_received INT NOT NULL CHECK (quantity_received > 0),
    quantity_rejected INT NOT NULL DEFAULT 0,
    unit VARCHAR(30),
    unit_cost_cents BIGINT NOT NULL DEFAULT 0,
    total_cost_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_grn_items_grn ON goods_receipt_items (goods_receipt_id);

ALTER TABLE goods_receipt_items ENABLE ROW LEVEL SECURITY;

CREATE POLICY "grn_items_select_policy" ON goods_receipt_items
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "grn_items_insert_policy" ON goods_receipt_items
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "grn_items_update_policy" ON goods_receipt_items
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "grn_items_delete_policy" ON goods_receipt_items
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);