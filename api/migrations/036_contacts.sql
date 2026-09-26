-- 036_contacts.sql
-- Contact book for the Communications area: the people a school can actually
-- message, independent of whether they have a learner account.
--
-- Motivation: guardians are a *learner* concept — a school cannot message a
-- prospective parent, a sponsor, a vendor contact, or a staff member's next of
-- kin. The SMS/WhatsApp audience builder previously only resolved guardians,
-- so any other recipient was unreachable. `contacts` is that address book.
--
-- Design notes:
--  - `phone` is stored in E.164 form (+2547XXXXXXXX) and is UNIQUE per tenant.
--    The API normalises local formats (0712345678, 254712345678) on write, so
--    a school cannot end up with two "different" rows for the same person.
--  - `tags` powers audience segments ("Grade 4 parents", "Alumni 2024").
--  - `is_opted_out` is honoured by the audience builder, so consent state lives
--    with the contact instead of being re-derived per campaign.
--  - `guardian_id` is optional and links a contact to an existing guardian, so
--    the two books can be reconciled without deleting history.
--  - Deletes are soft (is_active = false): a contact that received a message
--    must keep pointing at a real delivery log.
--
-- Tenant isolation is enforced in the API (tenant_id comes from the verified
-- JWT, never the request body). RLS policies mirror the rest of the schema as
-- defence in depth.

CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(15) NOT NULL,
    email VARCHAR(255),

    -- Free-text so a school is not boxed into our list: parent, guardian,
    -- staff, sponsor, vendor, prospective...
    relationship VARCHAR(50),

    -- Optional grouping used by the audience builder, e.g. "Grade 4" or
    -- "North". Kept as a plain string to match how grades are stored today.
    grade_stream VARCHAR(100),

    tags TEXT[] NOT NULL DEFAULT '{}',
    notes TEXT,

    -- manual  = added through the UI
    -- import  = added through bulk import
    -- guardian= created from an existing guardian record
    source VARCHAR(20) NOT NULL DEFAULT 'manual'
        CHECK (source IN ('manual', 'import', 'guardian')),

    guardian_id UUID REFERENCES guardians(id) ON DELETE SET NULL,

    is_active BOOLEAN NOT NULL DEFAULT true,
    is_opted_out BOOLEAN NOT NULL DEFAULT false,

    created_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- One contact per phone number per school. Deduplication happens here as a
    -- backstop; the API also reports duplicates per row during bulk import.
    CONSTRAINT contacts_tenant_phone_unique UNIQUE (tenant_id, phone)
);

-- Listing is always "this tenant's active contacts, by name".
CREATE INDEX IF NOT EXISTS idx_contacts_tenant_name ON contacts (tenant_id, full_name);
CREATE INDEX IF NOT EXISTS idx_contacts_tenant_active ON contacts (tenant_id, is_active);
-- Audience builder filters by tag membership.
CREATE INDEX IF NOT EXISTS idx_contacts_tags ON contacts USING GIN (tags);

CREATE TRIGGER set_contacts_updated_at
    BEFORE UPDATE ON contacts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE contacts ENABLE ROW LEVEL SECURITY;

CREATE POLICY "contacts_select_policy" ON contacts
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "contacts_insert_policy" ON contacts
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "contacts_update_policy" ON contacts
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "contacts_delete_policy" ON contacts
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);