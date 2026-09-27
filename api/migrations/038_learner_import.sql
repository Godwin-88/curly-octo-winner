-- 038_learner_import.sql
-- Staging area for importing a learner roster from a CSV.
--
-- Motivation: a school's roster spreadsheet is rarely clean. It has a parent
-- column, a learner column, a register number, and half the rows missing a
-- grade. Rejecting the whole file over one blank cell is the wrong trade -- the
-- school still wants the 298 good rows and wants to finish the rest later. So
-- an import lands in a staging table where every field is nullable, the school
-- edits rows in place, and a row becomes a real learner only once it holds
-- enough to be one.
--
-- Design notes:
--  - Two tables, not one. A batch records which file a row came from, who
--    uploaded it and how it is going; rows are the editable working set.
--  - Every data column is nullable ON PURPOSE. Nothing here is rejected for
--    being incomplete -- that is the whole point of staging.
--  - is_sufficient and missing are STORED generated columns, so the "can this
--    become a learner yet" rule lives in exactly one place (the database)
--    instead of being re-derived in the API, the UI and any future report. A
--    row is sufficient once it has a learner name, a learner number and a
--    grade -- the three fields learners itself requires, since upi, full_name
--    and grade are all NOT NULL there.
--  - A stage row is NOT a learner. It only gains learner_id / guardian_id on
--    promotion, and both are ON DELETE SET NULL so deleting a real learner
--    leaves the staging history readable instead of cascading it away.
--  - tags matches the contacts table (TEXT[] + GIN) so a school can group an
--    import ("2025 intake", "needs UPI") the way it already groups contacts.
--  - The learner number is deliberately NOT unique here. Two rows may share
--    one (a row per parent), and a number that already exists in learners is
--    caught at promotion time with a clear message rather than blocked at
--    paste time.
--
-- Tenant isolation is enforced in the API (tenant_id comes from the verified
-- JWT, never the request body). RLS policies mirror the rest of the schema as
-- defence in depth.

CREATE TABLE IF NOT EXISTS learner_import_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Shown back to the user so several uploads can be told apart.
    filename TEXT,

    uploaded_by UUID REFERENCES staff(id) ON DELETE SET NULL,

    -- Recomputed from the rows on read; cached here so a batch list does not
    -- need a correlated subquery per row.
    total_rows INT NOT NULL DEFAULT 0,
    ready_rows INT NOT NULL DEFAULT 0,
    imported_rows INT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE TABLE IF NOT EXISTS learner_import_rows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    batch_id UUID NOT NULL REFERENCES learner_import_batches(id) ON DELETE CASCADE,

    -- The 1-based line in the source file, kept so an error can point the user
    -- back at the exact row in Excel.
    row_number INT NOT NULL,

    -- ---- the columns a school actually has in its spreadsheet ---------------
    -- All nullable: a row may arrive half-finished and be completed in the UI.
    parent_name TEXT,
    parent_phone TEXT,
    student_name TEXT,
    -- Becomes learners.upi on promotion. Named student_number because that is
    -- what it is called in the school's own spreadsheet, not in our schema.
    student_number TEXT,
    grade TEXT,
    stream TEXT,

    -- Same shape as contacts.tags: free-form grouping used for filtering.
    tags TEXT[] NOT NULL DEFAULT '{}',
    notes TEXT,

    -- draft    = staged, editable, not yet a learner
    -- imported = promoted into learners/guardians
    -- rejected = a human marked it as not wanted (e.g. a withdrawn learner)
    status VARCHAR(20) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'imported', 'rejected')),

    -- Why promotion could not happen, in words a user can act on.
    problem TEXT,

    -- Filled in on promotion. SET NULL, not CASCADE: the staging row records
    -- what was imported and should outlive the learner it produced.
    learner_id UUID REFERENCES learners(id) ON DELETE SET NULL,
    guardian_id UUID REFERENCES guardians(id) ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- The sufficiency rule, stated once. Must stay in step with `missing` and
    -- with MissingFields() in the service -- a test asserts the three agree.
    is_sufficient BOOLEAN GENERATED ALWAYS AS (
        btrim(COALESCE(student_name, '')) <> ''
        AND btrim(COALESCE(student_number, '')) <> ''
        AND btrim(COALESCE(grade, '')) <> ''
    ) STORED,

    -- Which required fields are absent, so the UI can say what to fix rather
    -- than only that the row is invalid.
    missing TEXT[] GENERATED ALWAYS AS (
        ARRAY_REMOVE(ARRAY[
            CASE WHEN btrim(COALESCE(student_name, '')) = ''    THEN 'student_name' END,
            CASE WHEN btrim(COALESCE(student_number, '')) = '' THEN 'student_number' END,
            CASE WHEN btrim(COALESCE(grade, '')) = ''          THEN 'grade' END
        ], NULL)
    ) STORED,

    -- Row numbers are unique within a file, which is what makes "line 12 of that
    -- upload" a stable way to refer to a row.
    UNIQUE (batch_id, row_number)
);

-- The staging screen lists a batch newest-first.
CREATE INDEX IF NOT EXISTS idx_learner_import_batches_tenant
    ON learner_import_batches (tenant_id, created_at DESC);
-- The working list: a batch's rows, in file order.
CREATE INDEX IF NOT EXISTS idx_learner_import_rows_batch
    ON learner_import_rows (batch_id, row_number);
-- Ready-to-promote filter.
CREATE INDEX IF NOT EXISTS idx_learner_import_rows_status
    ON learner_import_rows (tenant_id, status, is_sufficient);
-- Tag filtering, matching contacts.
CREATE INDEX IF NOT EXISTS idx_learner_import_rows_tags
    ON learner_import_rows USING GIN (tags);

CREATE TRIGGER set_learner_import_batches_updated_at
    BEFORE UPDATE ON learner_import_batches
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_learner_import_rows_updated_at
    BEFORE UPDATE ON learner_import_rows
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE learner_import_batches ENABLE ROW LEVEL SECURITY;
ALTER TABLE learner_import_rows ENABLE ROW LEVEL SECURITY;

CREATE POLICY "learner_import_batches_select_policy" ON learner_import_batches
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_import_batches_insert_policy" ON learner_import_batches
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_import_batches_update_policy" ON learner_import_batches
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_import_batches_delete_policy" ON learner_import_batches
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_import_rows_select_policy" ON learner_import_rows
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_import_rows_insert_policy" ON learner_import_rows
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_import_rows_update_policy" ON learner_import_rows
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "learner_import_rows_delete_policy" ON learner_import_rows
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);
