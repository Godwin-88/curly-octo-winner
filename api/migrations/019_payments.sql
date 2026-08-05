-- 019_payments.sql
-- Finance: Payments (M-Pesa, bank, cash, cheque) and ledger view

-- Payment: a single payment against an invoice
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    channel VARCHAR(20) NOT NULL
        CHECK (channel IN ('mpesa', 'bank', 'cash', 'cheque')),
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'completed', 'failed', 'reversed')),
    reference VARCHAR(64),
    paid_by VARCHAR(255),
    phone VARCHAR(20),
    paid_at TIMESTAMPTZ,
    received_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    notes TEXT,
    -- M-Pesa Daraja fields
    checkout_request_id VARCHAR(64),
    merchant_request_id VARCHAR(64),
    mpesa_receipt VARCHAR(32),
    mpesa_result_code VARCHAR(8),
    mpesa_result_desc TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payments_tenant ON payments (tenant_id);
CREATE INDEX IF NOT EXISTS idx_payments_invoice ON payments (tenant_id, invoice_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments (tenant_id, status, paid_at);
CREATE INDEX IF NOT EXISTS idx_payments_mpesa ON payments (tenant_id, checkout_request_id);

CREATE TRIGGER set_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE payments ENABLE ROW LEVEL SECURITY;

CREATE POLICY "payments_select_policy" ON payments
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "payments_insert_policy" ON payments
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "payments_update_policy" ON payments
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "payments_delete_policy" ON payments
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Ledger view: per-invoice running totals for reporting
CREATE OR REPLACE VIEW payment_ledger AS
SELECT
    p.id AS payment_id,
    p.tenant_id,
    p.invoice_id,
    i.invoice_number,
    i.learner_id,
    l.full_name AS learner_name,
    l.grade,
    l.stream,
    i.term,
    i.year,
    p.channel,
    p.status,
    p.amount_cents,
    p.reference,
    p.phone,
    p.paid_at,
    p.created_at
FROM payments p
JOIN invoices i ON i.id = p.invoice_id
JOIN learners l ON l.id = i.learner_id;