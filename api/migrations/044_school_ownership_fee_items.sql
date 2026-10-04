-- 044_school_ownership_fee_items.sql
-- Public and private schools charge different things, and what a school
-- charges is the school's decision, not a list built into the software.
--
--   * tenants.ownership records whether a school is public or private.
--   * fee_categories is each school's own list of what it charges for. A fee
--     structure is built from it, and the school adds to it freely.
--   * The fixed list of item kinds on fee and invoice items is dropped.

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS ownership VARCHAR(10) NOT NULL DEFAULT 'private'
        CHECK (ownership IN ('public', 'private'));

CREATE TABLE IF NOT EXISTS fee_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    -- Billed only to the learners who take it up (transport, boarding, lunch).
    is_optional BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_fee_categories_name ON fee_categories (tenant_id, lower(name));

CREATE TRIGGER set_fee_categories_updated_at
    BEFORE UPDATE ON fee_categories
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE fee_categories ENABLE ROW LEVEL SECURITY;

CREATE POLICY "fee_categories_tenant_policy" ON fee_categories
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- What a school charges is free text now.
ALTER TABLE fee_structure_items DROP CONSTRAINT IF EXISTS fee_structure_items_item_type_check;
ALTER TABLE invoice_items DROP CONSTRAINT IF EXISTS invoice_items_item_type_check;
ALTER TABLE fee_structure_items ALTER COLUMN item_type TYPE VARCHAR(100);
ALTER TABLE invoice_items ALTER COLUMN item_type TYPE VARCHAR(100);

-- Existing schools start with what they already charge, so nothing they have
-- set up disappears from the list, plus the usual private-school items.
INSERT INTO fee_categories (tenant_id, name, is_optional, sort_order)
SELECT DISTINCT ON (i.tenant_id, lower(i.name)) i.tenant_id, i.name, i.is_optional, 0
FROM fee_structure_items i
ORDER BY i.tenant_id, lower(i.name), i.created_at
ON CONFLICT DO NOTHING;

INSERT INTO fee_categories (tenant_id, name, is_optional, sort_order)
SELECT t.id, d.name, d.is_optional, d.sort_order
FROM tenants t
CROSS JOIN (VALUES
    ('Tuition', false, 1),
    ('Activity fee', false, 2),
    ('Lunch', true, 3),
    ('Transport', true, 4),
    ('Boarding', true, 5),
    ('Caution money', false, 6)
) AS d(name, is_optional, sort_order)
ON CONFLICT DO NOTHING;
