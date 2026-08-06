-- 028_supplier_payments.sql
-- Supplier Payments (EPIC 8: Manage Supplier & Procurement Operations)

-- Supplier payments: three-way match (PO → GRN → Invoice) before payment authorisation
CREATE TABLE IF NOT EXISTS supplier_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    payment_no VARCHAR(30) NOT NULL,
    supplier_id UUID NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT,
    purchase_order_id UUID REFERENCES purchase_orders(id) ON DELETE SET NULL,
    goods_receipt_id UUID REFERENCES goods_receipts(id) ON DELETE SET NULL,
    invoice_number VARCHAR(50),
    invoice_date DATE,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    payment_method VARCHAR(20) NOT NULL DEFAULT 'bank'
        CHECK (payment_method IN ('bank', 'mpesa', 'cash', 'cheque')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'authorised', 'paid', 'cancelled')),
    authorised_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    authorised_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    reference VARCHAR(100),
    notes TEXT,
    created_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, payment_no)
);

CREATE INDEX IF NOT EXISTS idx_supplier_payments_tenant_status ON supplier_payments (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_supplier_payments_supplier ON supplier_payments (supplier_id);

CREATE TRIGGER set_supplier_payments_updated_at
    BEFORE UPDATE ON supplier_payments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE supplier_payments ENABLE ROW LEVEL SECURITY;

CREATE POLICY "supplier_payments_select_policy" ON supplier_payments
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "supplier_payments_insert_policy" ON supplier_payments
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "supplier_payments_update_policy" ON supplier_payments
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "supplier_payments_delete_policy" ON supplier_payments
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);