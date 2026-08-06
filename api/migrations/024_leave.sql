-- 024_leave.sql
-- Leave Management & Staff Attendance (EPIC 7: Manage Human Resources)

-- Leave requests: application → approval workflow
CREATE TABLE IF NOT EXISTS leave_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    staff_id UUID NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
    leave_type VARCHAR(30) NOT NULL
        CHECK (leave_type IN ('annual', 'sick', 'maternity', 'paternity', 'compassionate', 'study', 'unpaid', 'other')),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    days INT NOT NULL CHECK (days > 0),
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'denied', 'cancelled')),
    approved_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    approved_at TIMESTAMPTZ,
    denial_reason TEXT,
    substitute_id UUID REFERENCES staff(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date)
);

CREATE INDEX IF NOT EXISTS idx_leave_requests_staff ON leave_requests (staff_id);
CREATE INDEX IF NOT EXISTS idx_leave_requests_tenant_status ON leave_requests (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_leave_requests_dates ON leave_requests (tenant_id, start_date, end_date);

CREATE TRIGGER set_leave_requests_updated_at
    BEFORE UPDATE ON leave_requests
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE leave_requests ENABLE ROW LEVEL SECURITY;

CREATE POLICY "leave_requests_select_policy" ON leave_requests
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "leave_requests_insert_policy" ON leave_requests
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "leave_requests_update_policy" ON leave_requests
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "leave_requests_delete_policy" ON leave_requests
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Staff attendance: daily clock-in/out records
CREATE TABLE IF NOT EXISTS staff_attendance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    staff_id UUID NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    clock_in TIMESTAMPTZ,
    clock_out TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'present'
        CHECK (status IN ('present', 'absent', 'late', 'half_day', 'leave', 'holiday')),
    notes TEXT,
    marked_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, staff_id, date)
);

CREATE INDEX IF NOT EXISTS idx_staff_attendance_staff ON staff_attendance (staff_id);
CREATE INDEX IF NOT EXISTS idx_staff_attendance_tenant_date ON staff_attendance (tenant_id, date);

CREATE TRIGGER set_staff_attendance_updated_at
    BEFORE UPDATE ON staff_attendance
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE staff_attendance ENABLE ROW LEVEL SECURITY;

CREATE POLICY "staff_attendance_select_policy" ON staff_attendance
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "staff_attendance_insert_policy" ON staff_attendance
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "staff_attendance_update_policy" ON staff_attendance
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "staff_attendance_delete_policy" ON staff_attendance
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- TSC-aligned performance appraisals
CREATE TABLE IF NOT EXISTS staff_appraisals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    staff_id UUID NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
    year INT NOT NULL,
    term INT CHECK (term IN (1, 2, 3)),
    appraiser_id UUID REFERENCES staff(id) ON DELETE SET NULL,
    scores JSONB NOT NULL DEFAULT '{}',
    overall_score NUMERIC(5,2),
    rating VARCHAR(20),
    comments TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'approved')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, staff_id, year, term)
);

CREATE INDEX IF NOT EXISTS idx_staff_appraisals_staff ON staff_appraisals (staff_id);
CREATE INDEX IF NOT EXISTS idx_staff_appraisals_tenant ON staff_appraisals (tenant_id, year, term);

CREATE TRIGGER set_staff_appraisals_updated_at
    BEFORE UPDATE ON staff_appraisals
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE staff_appraisals ENABLE ROW LEVEL SECURITY;

CREATE POLICY "staff_appraisals_select_policy" ON staff_appraisals
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "staff_appraisals_insert_policy" ON staff_appraisals
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "staff_appraisals_update_policy" ON staff_appraisals
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "staff_appraisals_delete_policy" ON staff_appraisals
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);