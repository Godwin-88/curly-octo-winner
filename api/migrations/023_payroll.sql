-- 023_payroll.sql
-- Payroll & Benefits (EPIC 7: Manage Human Resources)

-- Payroll runs: one per staff per month
CREATE TABLE IF NOT EXISTS payroll_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    staff_id UUID NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
    month INT NOT NULL CHECK (month BETWEEN 1 AND 12),
    year INT NOT NULL,
    basic_salary_cents BIGINT NOT NULL DEFAULT 0,
    allowances_cents BIGINT NOT NULL DEFAULT 0,
    gross_cents BIGINT NOT NULL DEFAULT 0,
    paye_cents BIGINT NOT NULL DEFAULT 0,
    nhif_cents BIGINT NOT NULL DEFAULT 0,
    nssf_cents BIGINT NOT NULL DEFAULT 0,
    other_deductions_cents BIGINT NOT NULL DEFAULT 0,
    net_cents BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'approved', 'paid')),
    paid_at TIMESTAMPTZ,
    created_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, staff_id, month, year)
);

CREATE INDEX IF NOT EXISTS idx_payroll_runs_staff ON payroll_runs (staff_id);
CREATE INDEX IF NOT EXISTS idx_payroll_runs_tenant_month ON payroll_runs (tenant_id, month, year);

CREATE TRIGGER set_payroll_runs_updated_at
    BEFORE UPDATE ON payroll_runs
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE payroll_runs ENABLE ROW LEVEL SECURITY;

CREATE POLICY "payroll_runs_select_policy" ON payroll_runs
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "payroll_runs_insert_policy" ON payroll_runs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "payroll_runs_update_policy" ON payroll_runs
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "payroll_runs_delete_policy" ON payroll_runs
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Payroll line items: breakdown of earnings and deductions per run
CREATE TABLE IF NOT EXISTS payroll_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    payroll_run_id UUID NOT NULL REFERENCES payroll_runs(id) ON DELETE CASCADE,
    item_type VARCHAR(20) NOT NULL CHECK (item_type IN ('earning', 'deduction')),
    name VARCHAR(100) NOT NULL,
    amount_cents BIGINT NOT NULL DEFAULT 0,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_payroll_items_run ON payroll_items (payroll_run_id);
CREATE INDEX IF NOT EXISTS idx_payroll_items_tenant ON payroll_items (tenant_id);

ALTER TABLE payroll_items ENABLE ROW LEVEL SECURITY;

CREATE POLICY "payroll_items_select_policy" ON payroll_items
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "payroll_items_insert_policy" ON payroll_items
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "payroll_items_update_policy" ON payroll_items
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "payroll_items_delete_policy" ON payroll_items
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);