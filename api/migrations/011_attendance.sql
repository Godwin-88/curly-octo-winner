-- 011_attendance.sql
-- Attendance Management

CREATE TABLE IF NOT EXISTS attendance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    learner_id UUID NOT NULL REFERENCES learners(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('present', 'absent', 'late', 'excused')),
    marked_by UUID REFERENCES staff(id) ON DELETE SET NULL,
    reason TEXT,
    sms_notified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, learner_id, date)
);

CREATE INDEX IF NOT EXISTS idx_attendance_learner ON attendance (learner_id);
CREATE INDEX IF NOT EXISTS idx_attendance_date ON attendance (tenant_id, date);
CREATE INDEX IF NOT EXISTS idx_attendance_status ON attendance (tenant_id, status);

CREATE TRIGGER set_attendance_updated_at
    BEFORE UPDATE ON attendance
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE attendance ENABLE ROW LEVEL SECURITY;

CREATE POLICY "attendance_select_policy" ON attendance
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "attendance_insert_policy" ON attendance
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "attendance_update_policy" ON attendance
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "attendance_delete_policy" ON attendance
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Attendance summary view
CREATE OR REPLACE VIEW attendance_summary AS
SELECT
    a.tenant_id,
    a.learner_id,
    l.full_name AS learner_name,
    l.grade,
    l.stream,
    a.date,
    a.status,
    a.reason,
    a.sms_notified,
    a.created_at
FROM attendance a
JOIN learners l ON l.id = a.learner_id AND l.tenant_id = a.tenant_id;