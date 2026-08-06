-- 025_suppliers.sql
-- Supplier Registry (EPIC 8: Manage Supplier & Procurement Operations)

-- Suppliers: KYC registry for vendor onboarding
CREATE TABLE IF NOT EXISTS suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    business_registration VARCHAR(100),
    kra_pin VARCHAR(20),
    category VARCHAR(50) DEFAULT 'general'
        CHECK (category IN ('textbooks', 'stationery', 'furniture', 'ict', 'uniforms', 'food', 'lab', 'construction', 'transport', 'general')),
    contact_person VARCHAR(255),
    phone VARCHAR(20),
    email VARCHAR(255),
    whatsapp_phone VARCHAR(20),
    physical_address TEXT,
    bank_branch VARCHAR(255),
    bank_account_name VARCHAR(255),
    bank_account_number VARCHAR(50),
    bank_swift_code VARCHAR(20),
    notes TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_suppliers_tenant ON suppliers (tenant_id, is_active);
CREATE INDEX IF NOT EXISTS idx_suppliers_category ON suppliers (tenant_id, category);

CREATE TRIGGER set_suppliers_updated_at
    BEFORE UPDATE ON suppliers
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE suppliers ENABLE ROW LEVEL SECURITY;

CREATE POLICY "suppliers_select_policy" ON suppliers
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "suppliers_insert_policy" ON suppliers
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "suppliers_update_policy" ON suppliers
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "suppliers_delete_policy" ON suppliers
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);