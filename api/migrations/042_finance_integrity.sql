-- 042_finance_integrity.sql
-- Finance: what a school needs before it can trust the figures.
--
--   * A payment can never disappear: deleting an invoice no longer deletes its
--     payments (the foreign key refuses instead), invoices are voided with a
--     reason, and a reversal records who, when and why.
--   * A learner is billed once per term.
--   * Invoice and receipt numbers run in sequence per school and year.
--   * Money paid straight to the paybill lands in mpesa_inbox and is allocated
--     to invoices from there, so nothing received is unaccounted for.

-- Per-school running numbers (invoices, receipts).
CREATE TABLE IF NOT EXISTS finance_counters (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kind VARCHAR(20) NOT NULL CHECK (kind IN ('invoice', 'receipt')),
    year INT NOT NULL,
    last_value BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, kind, year)
);

ALTER TABLE finance_counters ENABLE ROW LEVEL SECURITY;

CREATE POLICY "finance_counters_tenant_policy" ON finance_counters
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- Invoices: voided, never deleted.
ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS voided_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS voided_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS void_reason TEXT;

-- One live invoice per learner per term. If this fails, the school already has
-- a learner billed twice for a term: void one of the two and run it again.
CREATE UNIQUE INDEX IF NOT EXISTS uq_invoices_learner_term
    ON invoices (tenant_id, learner_id, term, year)
    WHERE status <> 'void';

-- Payments.
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS receipt_number VARCHAR(32),
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(64),
    ADD COLUMN IF NOT EXISTS attempted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS failure_code VARCHAR(40),
    ADD COLUMN IF NOT EXISTS reversed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reversed_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS reversal_reason TEXT,
    ADD COLUMN IF NOT EXISTS inbox_id UUID;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_receipt_number
    ON payments (tenant_id, receipt_number)
    WHERE receipt_number IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_idempotency
    ON payments (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_payments_pending_mpesa
    ON payments (created_at)
    WHERE channel = 'mpesa' AND status = 'pending';

-- A payment outlives its invoice: refuse the delete instead of cascading.
-- NO ACTION rather than RESTRICT, so removing a whole school still works.
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_invoice_id_fkey;
ALTER TABLE payments
    ADD CONSTRAINT payments_invoice_id_fkey
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE NO ACTION;

-- Money received on the school's paybill outside an STK request (a parent
-- paying from the M-Pesa menu). One row per M-Pesa transaction.
CREATE TABLE IF NOT EXISTS mpesa_inbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    trans_id VARCHAR(32) NOT NULL,
    trans_time TIMESTAMPTZ NOT NULL DEFAULT now(),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    allocated_cents BIGINT NOT NULL DEFAULT 0 CHECK (allocated_cents >= 0),
    bill_ref VARCHAR(64),
    payer_name VARCHAR(255),
    payer_phone VARCHAR(80),
    short_code VARCHAR(20),
    learner_id UUID REFERENCES learners(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'unmatched'
        CHECK (status IN ('unmatched', 'part_allocated', 'allocated')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (allocated_cents <= amount_cents),
    -- An M-Pesa transaction code is unique across Safaricom: a repeated
    -- confirmation can never be recorded twice.
    UNIQUE (trans_id)
);

CREATE INDEX IF NOT EXISTS idx_mpesa_inbox_tenant ON mpesa_inbox (tenant_id, status, trans_time DESC);

CREATE TRIGGER set_mpesa_inbox_updated_at
    BEFORE UPDATE ON mpesa_inbox
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE mpesa_inbox ENABLE ROW LEVEL SECURITY;

CREATE POLICY "mpesa_inbox_tenant_policy" ON mpesa_inbox
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE payments
    ADD CONSTRAINT payments_inbox_id_fkey
    FOREIGN KEY (inbox_id) REFERENCES mpesa_inbox(id) ON DELETE NO ACTION;
