-- seed.sql
-- Demo data for development: 1 tenant, 5 staff, 20 guardians, 50 learners
-- Requires: migrations 001-008 applied first

-- Insert demo tenant
INSERT INTO tenants (id, name, slug, subscription_tier, at_sender_id)
VALUES ('a0000000-0000-0000-0000-000000000001', 'Jua Kali Primary School', 'jua-kali', 'free', 'SHULE360')
ON CONFLICT (slug) DO NOTHING;

-- Staff records
-- Note: These staff are created without Supabase Auth users.
-- Use Supabase Admin API to create users with temp passwords in production.
INSERT INTO staff (id, tenant_id, full_name, email, phone, role) VALUES
    ('b0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'John Kamau', 'principal@juakali.sch.ke', '+254712345601', 'principal'),
    ('b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'Jane Wanjiku', 'bursar@juakali.sch.ke', '+254712345602', 'bursar'),
    ('b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'Peter Otieno', 'teacher1@juakali.sch.ke', '+254712345603', 'teacher'),
    ('b0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', 'Mary Akinyi', 'teacher2@juakali.sch.ke', '+254712345604', 'teacher'),
    ('b0000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001', 'David Mwangi', 'transport@juakali.sch.ke', '+254712345605', 'transport_manager')
ON CONFLICT (tenant_id, email) DO NOTHING;

-- Guardians (20)
INSERT INTO guardians (id, tenant_id, full_name, phone_primary, phone_wa, wa_opted_in, is_transport_enrolled) VALUES
    ('c0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'Grace Muthoni', '+254712345101', '+254712345101', true, true),
    ('c0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'Samuel Kiprop', '+254712345102', '+254712345102', true, false),
    ('c0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'Esther Wambui', '+254712345103', '+254712345103', true, true),
    ('c0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', 'Joseph Ochieng', '+254712345104', NULL, false, false),
    ('c0000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001', 'Faith Nyambura', '+254712345105', '+254712345105', true, true),
    ('c0000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000001', 'Hellen Chebet', '+254712345106', NULL, false, false),
    ('c0000000-0000-0000-0000-000000000007', 'a0000000-0000-0000-0000-000000000001', 'James Kariuki', '+254712345107', '+254712345107', true, true),
    ('c0000000-0000-0000-0000-000000000008', 'a0000000-0000-0000-0000-000000000001', 'Sarah Jerono', '+254712345108', '+254712345108', true, false),
    ('c0000000-0000-0000-0000-000000000009', 'a0000000-0000-0000-0000-000000000001', 'Paul Njoroge', '+254712345109', NULL, false, true),
    ('c0000000-0000-0000-0000-000000000010', 'a0000000-0000-0000-0000-000000000001', 'Nancy Wanjala', '+254712345110', '+254712345110', true, false),
    ('c0000000-0000-0000-0000-000000000011', 'a0000000-0000-0000-0000-000000000001', 'Patrick Kibet', '+254712345111', '+254712345111', true, true),
    ('c0000000-0000-0000-0000-000000000012', 'a0000000-0000-0000-0000-000000000001', 'Diana Akinyi', '+254712345112', NULL, false, false),
    ('c0000000-0000-0000-0000-000000000013', 'a0000000-0000-0000-0000-000000000001', 'George Maina', '+254712345113', '+254712345113', true, true),
    ('c0000000-0000-0000-0000-000000000014', 'a0000000-0000-0000-0000-000000000001', 'Ruth Wanjiru', '+254712345114', '+254712345114', true, false),
    ('c0000000-0000-0000-0000-000000000015', 'a0000000-0000-0000-0000-000000000001', 'Tom Odhiambo', '+254712345115', NULL, false, true),
    ('c0000000-0000-0000-0000-000000000016', 'a0000000-0000-0000-0000-000000000001', 'Joyce Awino', '+254712345116', '+254712345116', true, false),
    ('c0000000-0000-0000-0000-000000000017', 'a0000000-0000-0000-0000-000000000001', 'Eliud Ngetich', '+254712345117', '+254712345117', true, true),
    ('c0000000-0000-0000-0000-000000000018', 'a0000000-0000-0000-0000-000000000001', 'Monicah Waithera', '+254712345118', NULL, false, false),
    ('c0000000-0000-0000-0000-000000000019', 'a0000000-0000-0000-0000-000000000001', 'Daniel Kiprono', '+254712345119', '+254712345119', true, true),
    ('c0000000-0000-0000-0000-000000000020', 'a0000000-0000-0000-0000-000000000001', 'Catherine Mwikali', '+254712345120', '+254712345120', true, false)
ON CONFLICT DO NOTHING;

-- Learners (50) - Grades 4, 5, 6 with 2 streams each
DO $$
DECLARE
    i INT;
    grade TEXT;
    stream TEXT;
    learner_id UUID;
    guardian_id UUID;
    guardian_arr UUID[];
BEGIN
    FOR i IN 1..50 LOOP
        -- Assign grade and stream
        IF i <= 16 THEN
            grade := 'Grade 4';
            IF i <= 8 THEN stream := 'North'; ELSE stream := 'South'; END IF;
        ELSIF i <= 32 THEN
            grade := 'Grade 5';
            IF i <= 24 THEN stream := 'North'; ELSE stream := 'South'; END IF;
        ELSE
            grade := 'Grade 6';
            IF i <= 42 THEN stream := 'North'; ELSE stream := 'South'; END IF;
        END IF;

        -- Assign guardian (cyclically, each guardian gets 2-3 learners)
        guardian_id := ('c0000000-0000-0000-0000-0000000000' || LPAD(((i - 1) % 20 + 1)::text, 3, '0'))::uuid;
        guardian_arr := ARRAY[guardian_id];

        INSERT INTO learners (id, tenant_id, upi, full_name, date_of_birth, grade, stream, guardian_ids)
        VALUES (
            ('d0000000-0000-0000-0000-0000' || LPAD(i::text, 8, '0'))::uuid,
            'a0000000-0000-0000-0000-000000000001',
            'TEST' || LPAD(i::text, 8, '0'),
            'Learner ' || i,
            '2016-01-01'::date + (i || ' days')::interval,
            grade,
            stream,
            guardian_arr
        )
        ON CONFLICT (tenant_id, upi) DO NOTHING;
    END LOOP;
END $$;

-- Transport (EPIC 4): 2 vehicles, 2 routes with stops, assignments, and a sample trip
-- Requires: migrations 014-016 applied first

-- Vehicles
INSERT INTO vehicles (id, tenant_id, registration, make, model, capacity, year, status, insurance_expiry, inspection_expiry, driver_id, driver_name, driver_phone) VALUES
    ('e0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'KDE 123A', 'Toyota', 'HiAce', 14, 2019, 'active', '2025-12-31', '2025-09-30', 'b0000000-0000-0000-0000-000000000005', 'David Mwangi', '+254712345605'),
    ('e0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'KDX 456B', 'Nissan', 'Civilian', 26, 2021, 'active', '2025-11-30', '2025-10-15', NULL, 'Samuel Mutua', '+254712345606')
ON CONFLICT (tenant_id, registration) DO NOTHING;

-- Routes
INSERT INTO routes (id, tenant_id, name, description, vehicle_id, active) VALUES
    ('f0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'Route A - Kasarani', 'Kasarani area morning/evening pickup', 'e0000000-0000-0000-0000-000000000001', true),
    ('f0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'Route B - Ruiru', 'Ruiru bypass corridor', 'e0000000-0000-0000-0000-000000000002', true)
ON CONFLICT (tenant_id, name) DO NOTHING;

-- Stops for Route A
INSERT INTO stops (id, tenant_id, route_id, name, sequence, latitude, longitude, landmark) VALUES
    ('f1000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000001', 'Kasarani Stage', 1, -1.2197, 36.8953, 'Kasarani Stadium'),
    ('f1000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000001', 'Mwiki', 2, -1.2078, 36.9156, 'Mwiki Shopping Centre'),
    ('f1000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000001', 'Clay City', 3, -1.2300, 36.9061, 'Clay City Mall')
ON CONFLICT DO NOTHING;

-- Stops for Route B
INSERT INTO stops (id, tenant_id, route_id, name, sequence, latitude, longitude, landmark) VALUES
    ('f1000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000002', 'Ruiru Town', 1, -1.1470, 36.9691, 'Ruiru Catholic'),
    ('f1000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000002', 'Juja Road', 2, -1.1758, 36.9433, 'Juja Road Junction'),
    ('f1000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000002', 'Githurai 44', 3, -1.2545, 36.8946, 'Githurai 44 Stage')
ON CONFLICT DO NOTHING;

-- Assign a few learners to Route A (to_school and from_school)
INSERT INTO route_assignments (tenant_id, route_id, learner_id, stop_id, direction) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000001', 'f1000000-0000-0000-0000-000000000001', 'both'),
    ('a0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', 'f1000000-0000-0000-0000-000000000002', 'to_school'),
    ('a0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000003', 'f1000000-0000-0000-0000-000000000005', 'both')
ON CONFLICT DO NOTHING;

-- Sample scheduled trip for Route A (tomorrow morning)
INSERT INTO trips (id, tenant_id, route_id, vehicle_id, direction, status, scheduled_departure, created_by, notes)
SELECT 'f2000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001',
       'f0000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000001',
       'to_school', 'scheduled', (CURRENT_DATE + 1) + INTERVAL '6:30' HOUR, 'b0000000-0000-0000-0000-000000000005',
       'Morning pickup Route A'
ON CONFLICT DO NOTHING;

-- Finance (EPIC 5): fee structures, invoices, and sample payments
-- Requires: migrations 017-019 applied first

-- Fee structure for Grade 4, Term 1
INSERT INTO fee_structures (id, tenant_id, name, grade, term, year, total_cents, active, notes, created_by) VALUES
    ('a1000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'Grade 4 Term 1 Fees', 'Grade 4', 1, 2026, 1750000, true, 'Tuition, caution, transport and activity', 'b0000000-0000-0000-0000-000000000002')
ON CONFLICT (tenant_id, grade, term, year) DO NOTHING;

INSERT INTO fee_structure_items (id, tenant_id, fee_structure_id, name, amount_cents, item_type, is_optional, sort_order) VALUES
    ('a1100000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000001', 'Tuition', 1200000, 'tuition', false, 1),
    ('a1100000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000001', 'Caution', 100000, 'caution', false, 2),
    ('a1100000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000001', 'Transport', 300000, 'transport', false, 3),
    ('a1100000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000001', 'Activities', 150000, 'activity', true, 4)
ON CONFLICT (fee_structure_id, name) DO NOTHING;

-- Fee structure for Grade 5, Term 1
INSERT INTO fee_structures (id, tenant_id, name, grade, term, year, total_cents, active, notes, created_by) VALUES
    ('a1000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'Grade 5 Term 1 Fees', 'Grade 5', 1, 2026, 1900000, true, 'Tuition, caution, transport and activity', 'b0000000-0000-0000-0000-000000000002')
ON CONFLICT (tenant_id, grade, term, year) DO NOTHING;

INSERT INTO fee_structure_items (id, tenant_id, fee_structure_id, name, amount_cents, item_type, is_optional, sort_order) VALUES
    ('a1100000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000002', 'Tuition', 1300000, 'tuition', false, 1),
    ('a1100000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000002', 'Caution', 100000, 'caution', false, 2),
    ('a1100000-0000-0000-0000-000000000007', 'a0000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000002', 'Transport', 300000, 'transport', false, 3),
    ('a1100000-0000-0000-0000-000000000008', 'a0000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000002', 'Activities', 200000, 'activity', true, 4)
ON CONFLICT (fee_structure_id, name) DO NOTHING;

-- Sample invoices for a few Grade 4 learners (Term 1 2026)
INSERT INTO invoices (id, tenant_id, learner_id, fee_structure_id, invoice_number, term, year, issue_date, due_date, total_cents, discount_cents, paid_cents, status, created_by)
SELECT 'a2000000-0000-0000-0000-00000000000' || lp, 'a0000000-0000-0000-0000-000000000001',
       ('d0000000-0000-0000-0000-0000' || LPAD(lp::text, 8, '0'))::uuid,
       'a1000000-0000-0000-0000-000000000001',
       'INV-2026-1-' || LPAD(lp::text, 6, '0'),
       1, 2026, CURRENT_DATE, (CURRENT_DATE + 30),
       1750000, 0, 0, 'unpaid', 'b0000000-0000-0000-0000-000000000002'
FROM generate_series(1, 6) AS lp
ON CONFLICT (tenant_id, invoice_number) DO NOTHING;

-- Copy fee structure items into the invoices' snapshot items
INSERT INTO invoice_items (tenant_id, invoice_id, name, amount_cents, item_type, is_optional, sort_order)
SELECT fsi.tenant_id, i.id, fsi.name, fsi.amount_cents, fsi.item_type, fsi.is_optional, fsi.sort_order
FROM fee_structure_items fsi
JOIN invoices i ON i.fee_structure_id = fsi.fee_structure_id
WHERE fsi.fee_structure_id = 'a1000000-0000-0000-0000-000000000001'
  AND NOT EXISTS (
      SELECT 1 FROM invoice_items ii
      WHERE ii.invoice_id = i.id AND ii.name = fsi.name
  );

-- A sample sibling discount on invoice 2 (learner 2 has a sibling in Grade 5)
INSERT INTO discounts (id, tenant_id, invoice_id, amount_cents, discount_type, reason, approved_by)
SELECT 'a3000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', i.id, 50000, 'sibling', 'Sibling discount (Grade 5 sibling)', 'b0000000-0000-0000-0000-000000000002'
FROM invoices i
WHERE i.invoice_number = 'INV-2026-1-000002'
ON CONFLICT DO NOTHING;

-- A sample completed M-Pesa payment on invoice 1
INSERT INTO payments (id, tenant_id, invoice_id, amount_cents, channel, status, reference, paid_by, phone, paid_at, mpesa_receipt, received_by)
SELECT 'a4000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', i.id, 1000000, 'mpesa', 'completed', 'STK', 'Grace Muthoni', '+254712345101', now() - INTERVAL '2 days', 'SFA1234567', 'b0000000-0000-0000-0000-000000000002'
FROM invoices i
WHERE i.invoice_number = 'INV-2026-1-000001'
ON CONFLICT DO NOTHING;

-- A sample cash payment on invoice 3
INSERT INTO payments (id, tenant_id, invoice_id, amount_cents, channel, status, reference, paid_by, paid_at, received_by)
SELECT 'a4000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', i.id, 500000, 'cash', 'completed', 'REC-1001', 'Cashier', now() - INTERVAL '1 day', 'b0000000-0000-0000-0000-000000000002'
FROM invoices i
WHERE i.invoice_number = 'INV-2026-1-000003'
ON CONFLICT DO NOTHING;

-- Reconcile invoice ledger fields (paid_cents, discount_cents, status) for seeded invoices
UPDATE invoices i SET
    discount_cents = COALESCE((SELECT SUM(amount_cents) FROM discounts WHERE invoice_id = i.id), 0),
    paid_cents = COALESCE((SELECT SUM(amount_cents) FROM payments WHERE invoice_id = i.id AND status = 'completed'), 0),
    status = CASE
        WHEN COALESCE((SELECT SUM(amount_cents) FROM payments WHERE invoice_id = i.id AND status = 'completed'), 0) >= i.total_cents - COALESCE((SELECT SUM(amount_cents) FROM discounts WHERE invoice_id = i.id), 0)
            THEN 'paid'
        WHEN COALESCE((SELECT SUM(amount_cents) FROM payments WHERE invoice_id = i.id AND status = 'completed'), 0) > 0
            THEN 'partially_paid'
        WHEN i.due_date IS NOT NULL AND i.due_date < CURRENT_DATE THEN 'overdue'
        ELSE 'unpaid'
    END
WHERE i.tenant_id = 'a0000000-0000-0000-0000-000000000001'
  AND i.invoice_number LIKE 'INV-2026-1-%';

-- Reports & Analytics (EPIC 6): assessments, attendance, and a sample report card
-- Requires: migrations 009-011, 020-021 applied first

-- Seed formative assessments for a few learners across sub-strands (Term 1 2026)
-- Uses the first few sub-strands from the curriculum and learners 1-6
INSERT INTO assessments (tenant_id, learner_id, sub_strand_id, rubric_level, note, teacher_id, term, year)
SELECT
    'a0000000-0000-0000-0000-000000000001',
    ('d0000000-0000-0000-0000-0000' || LPAD(lp::text, 8, '0'))::uuid,
    ss.id,
    (1 + ((lp + row_number() OVER (ORDER BY ss.id)) % 4))::int,
    'Formative observation for ' || ss.name,
    'b0000000-0000-0000-0000-000000000003',
    1, 2026
FROM generate_series(1, 6) AS lp
CROSS JOIN (
    SELECT id FROM sub_strands
    WHERE tenant_id = 'a0000000-0000-0000-0000-000000000001'
    ORDER BY id
    LIMIT 4
) ss
ON CONFLICT DO NOTHING;

-- Seed attendance for learners 1-6 (Term 1 2026: Jan-Apr)
INSERT INTO attendance (tenant_id, learner_id, date, status, marked_by)
SELECT
    'a0000000-0000-0000-0000-000000000001',
    ('d0000000-0000-0000-0000-0000' || LPAD(lp::text, 8, '0'))::uuid,
    d::date,
    CASE
        WHEN (lp + extract(day from d)::int) % 10 = 0 THEN 'absent'
        WHEN (lp + extract(day from d)::int) % 7 = 0 THEN 'late'
        ELSE 'present'
    END,
    'b0000000-0000-0000-0000-000000000003'
FROM generate_series(1, 6) AS lp
CROSS JOIN generate_series('2026-01-05'::date, '2026-04-30'::date, '1 day'::interval) AS d
ON CONFLICT (tenant_id, learner_id, date) DO NOTHING;

-- Generate a sample report card for learner 1 (Term 1 2026)
INSERT INTO report_cards (tenant_id, learner_id, term, year, status, overall_rating,
    core_competency_remarks, teacher_comments, attendance_summary, generated_by)
SELECT
    'a0000000-0000-0000-0000-000000000001',
    'd0000000-0000-0000-0000-000000000001',
    1, 2026, 'final', 3,
    '{"Communication and Collaboration": "Works well in groups and communicates clearly.", "Critical Thinking and Problem Solving": "Shows good reasoning in problem-solving tasks."}',
    '{"Mathematics": "Consistent effort and improvement.", "English": "Good reading comprehension."}',
    '{"total_days": 80, "present_days": 72, "absent_days": 4, "late_days": 3, "excused_days": 1, "attendance_rate": 90.0}',
    'b0000000-0000-0000-0000-000000000003'
ON CONFLICT (tenant_id, learner_id, term, year) DO NOTHING;

-- Populate report card items from the learner's assessments (learner 1, Term 1 2026)
INSERT INTO report_card_items (tenant_id, report_card_id, learning_area_id, strand_id, sub_strand_id, rubric_level, comment, sort_order)
SELECT
    a.tenant_id,
    rc.id,
    la.id,
    str.id,
    a.sub_strand_id,
    a.rubric_level,
    a.note,
    row_number() OVER (ORDER BY la.name, str.name, s.name)
FROM assessments a
JOIN report_cards rc ON rc.learner_id = a.learner_id AND rc.term = a.term AND rc.year = a.year
JOIN sub_strands s ON s.id = a.sub_strand_id
JOIN strands str ON str.id = s.strand_id
JOIN learning_areas la ON la.id = str.learning_area_id
WHERE a.tenant_id = 'a0000000-0000-0000-0000-000000000001'
  AND a.learner_id = 'd0000000-0000-0000-0000-000000000001'
  AND a.term = 1 AND a.year = 2026
  AND NOT EXISTS (
      SELECT 1 FROM report_card_items rci
      WHERE rci.report_card_id = rc.id AND rci.sub_strand_id = a.sub_strand_id
  );

-- Human Resources (EPIC 7): staff profiles, payroll, leave, attendance, appraisals
-- Requires: migrations 022-024 applied first

-- Enrich existing staff with HR profile fields
UPDATE staff SET
    tsc_number = 'TSC' || LPAD((row_number() OVER (ORDER BY id))::text, 6, '0'),
    national_id = 'ID' || LPAD((row_number() OVER (ORDER BY id))::text, 7, '0'),
    kra_pin = 'A00' || LPAD((row_number() OVER (ORDER BY id))::text, 5, '0') || 'K',
    department = CASE role
        WHEN 'principal' THEN 'Administration'
        WHEN 'bursar' THEN 'Finance'
        WHEN 'transport_manager' THEN 'Transport'
        ELSE 'Academic'
    END,
    job_title = CASE role
        WHEN 'principal' THEN 'Head Teacher'
        WHEN 'bursar' THEN 'School Bursar'
        WHEN 'transport_manager' THEN 'Transport Manager'
        ELSE 'Class Teacher'
    END,
    employment_type = 'permanent',
    hire_date = '2020-01-01'::date + (row_number() OVER (ORDER BY id) * 30 || ' days')::interval,
    qualifications = '[{"name": "Bachelor of Education", "institution": "Kenyatta University", "year": 2015}]'::jsonb,
    subjects = '["Mathematics", "English"]'::jsonb,
    emergency_contact = '{"name": "Next of Kin", "phone": "+254700000000"}'::jsonb,
    bank_details = '{"bank": "Equity Bank", "account": "1234567890", "branch": "Nairobi"}'::jsonb
WHERE tenant_id = 'a0000000-0000-0000-0000-000000000001';

-- Seed payroll runs for staff 1-5 (January 2026)
INSERT INTO payroll_runs (tenant_id, staff_id, month, year, basic_salary_cents, allowances_cents,
    gross_cents, paye_cents, nhif_cents, nssf_cents, other_deductions_cents, net_cents, status, created_by)
SELECT
    'a0000000-0000-0000-0000-000000000001',
    st.id,
    1, 2026,
    5000000, 1000000,
    6000000, 900000, 50000, 40000, 10000,
    6000000 - 900000 - 50000 - 40000 - 10000,
    'paid',
    'b0000000-0000-0000-0000-000000000002'
FROM staff st
WHERE st.tenant_id = 'a0000000-0000-0000-0000-000000000001'
  AND st.is_active = true
ON CONFLICT (tenant_id, staff_id, month, year) DO NOTHING;

-- Seed a pending leave request for teacher 1
INSERT INTO leave_requests (tenant_id, staff_id, leave_type, start_date, end_date, days, reason, status)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000003',
    'annual', '2026-08-10', '2026-08-14', 5,
    'Family vacation', 'pending'
) ON CONFLICT DO NOTHING;

-- Seed staff attendance for staff 1-5 (a few days in August 2026)
INSERT INTO staff_attendance (tenant_id, staff_id, date, status, marked_by)
SELECT
    'a0000000-0000-0000-0000-000000000001',
    st.id,
    d::date,
    CASE
        WHEN (row_number() OVER (ORDER BY st.id, d) % 9) = 0 THEN 'absent'
        WHEN (row_number() OVER (ORDER BY st.id, d) % 6) = 0 THEN 'late'
        ELSE 'present'
    END,
    'b0000000-0000-0000-0000-000000000002'
FROM staff st
CROSS JOIN generate_series('2026-08-03'::date, '2026-08-07'::date, '1 day'::interval) AS d
WHERE st.tenant_id = 'a0000000-0000-0000-0000-000000000001'
  AND st.is_active = true
ON CONFLICT (tenant_id, staff_id, date) DO NOTHING;

-- Seed a TSC appraisal for teacher 1 (Term 1 2026)
INSERT INTO staff_appraisals (tenant_id, staff_id, year, term, appraiser_id, scores, overall_score, rating, comments, status)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000003',
    2026, 1,
    'b0000000-0000-0000-0000-000000000001',
    '{"Professional Knowledge": 4, "Teaching Skills": 3, "Classroom Management": 4, "Student Assessment": 3, "Professional Development": 3}',
    3.4, 'Good',
    'Consistent and dedicated teacher. Shows strong classroom management.',
    'approved'
) ON CONFLICT (tenant_id, staff_id, year, term) DO NOTHING;

-- Seed suppliers (EPIC 8: Procurement)
INSERT INTO suppliers (id, tenant_id, name, business_registration, kra_pin, category, contact_person, phone, email, whatsapp_phone, physical_address, bank_branch, bank_account_name, bank_account_number, bank_swift_code, is_active) VALUES
    ('d0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'Kenya Textbooks Ltd', 'KTL-2024-001', 'P000123456A', 'textbooks', 'Alice Wanjiru', '+254700111001', 'sales@kenyatextbooks.co.ke', '+254700111001', 'Nairobi Industrial Area', 'Equity Bank Industrial Area', 'Kenya Textbooks Ltd', '0123456789', 'EQBLKENA', true),
    ('d0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'Stationery Plus', 'SP-2024-045', 'P000654321B', 'stationery', 'Brian Omondi', '+254700111002', 'orders@stationeryplus.co.ke', '+254700111002', 'Mombasa Road', 'KCB Mombasa Road', 'Stationery Plus', '9876543210', 'KCBLKENX', true),
    ('d0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'Furniture World Kenya', 'FWK-2023-112', 'P000987654C', 'furniture', 'Catherine Njeri', '+254700111003', 'info@furnitureworld.co.ke', NULL, 'Thika Road', 'Co-op Bank Thika Road', 'Furniture World Kenya', '4567890123', 'COOPKENA', true),
    ('d0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', 'ICT Solutions Ltd', 'ICT-2022-078', 'P000222333D', 'ict', 'Daniel Mutua', '+254700111004', 'sales@ictsolutions.co.ke', '+254700111004', 'Westlands', 'Stanbic Westlands', 'ICT Solutions Ltd', '7890123456', 'SBICKENX', true)
ON CONFLICT (id) DO NOTHING;

-- Seed a purchase requisition (pending)
INSERT INTO purchase_requisitions (id, tenant_id, requisition_no, title, department, requested_by, required_by, justification, status, total_estimate_cents)
VALUES (
    'e0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'REQ-A1B2C3D4',
    'Grade 4 Science Textbooks & Lab Supplies',
    'Science Department',
    'b0000000-0000-0000-0000-000000000003',
    '2026-09-01',
    'New KICD curriculum requires updated Grade 4 Science materials',
    'hod_approved',
    250000
) ON CONFLICT (tenant_id, requisition_no) DO NOTHING;

INSERT INTO requisition_items (tenant_id, requisition_id, item_name, description, quantity, unit, estimated_unit_cost_cents, estimated_total_cents) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000001', 'Grade 4 Science Textbook', 'KICD approved 2026 edition', 100, 'books', 1800, 180000),
    ('a0000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000001', 'Lab Test Tubes', 'Borosilicate glass, pack of 10', 50, 'packs', 1400, 70000)
ON CONFLICT DO NOTHING;

-- Seed a purchase order (sent)
INSERT INTO purchase_orders (id, tenant_id, po_number, requisition_id, supplier_id, order_date, expected_delivery, status, total_amount_cents, notes, created_by)
VALUES (
    'f0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'PO-5E6F7A8B',
    'e0000000-0000-0000-0000-000000000001',
    'd0000000-0000-0000-0000-000000000001',
    '2026-08-01',
    '2026-08-28',
    'sent',
    175000,
    'PO for Grade 4 Science textbooks',
    'b0000000-0000-0000-0000-000000000002'
) ON CONFLICT (tenant_id, po_number) DO NOTHING;

INSERT INTO purchase_order_items (tenant_id, purchase_order_id, item_name, description, quantity, unit, unit_cost_cents, total_cost_cents) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'f0000000-0000-0000-0000-000000000001', 'Grade 4 Science Textbook', 'KICD approved 2026 edition', 100, 'books', 1750, 175000)
ON CONFLICT DO NOTHING;

-- Seed a goods receipt (received)
INSERT INTO goods_receipts (id, tenant_id, grn_number, purchase_order_id, supplier_id, received_date, received_by, status, notes)
VALUES (
    'c0000000-0000-0000-0000-000000000021',
    'a0000000-0000-0000-0000-000000000001',
    'GRN-11223344',
    'f0000000-0000-0000-0000-000000000001',
    'd0000000-0000-0000-0000-000000000001',
    '2026-08-25',
    'b0000000-0000-0000-0000-000000000003',
    'received',
    'All 100 textbooks delivered in good condition'
) ON CONFLICT (tenant_id, grn_number) DO NOTHING;

INSERT INTO goods_receipt_items (tenant_id, goods_receipt_id, item_name, quantity_received, quantity_rejected, unit, unit_cost_cents, total_cost_cents) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000021', 'Grade 4 Science Textbook', 100, 0, 'books', 1750, 175000)
ON CONFLICT DO NOTHING;

-- Seed a supplier payment (authorised - three-way match complete)
INSERT INTO supplier_payments (id, tenant_id, payment_no, supplier_id, purchase_order_id, goods_receipt_id, invoice_number, invoice_date, amount_cents, payment_method, status, authorised_by, authorised_at, notes, created_by)
VALUES (
    'd0000000-0000-0000-0000-000000000011',
    'a0000000-0000-0000-0000-000000000001',
    'PAY-99887766',
    'd0000000-0000-0000-0000-000000000001',
    'f0000000-0000-0000-0000-000000000001',
    'c0000000-0000-0000-0000-000000000021',
    'INV-2026-045',
    '2026-08-25',
    175000,
    'bank',
    'authorised',
    'b0000000-0000-0000-0000-000000000002',
    now(),
    'Payment authorised after PO → GRN → Invoice three-way match',
    'b0000000-0000-0000-0000-000000000002'
) ON CONFLICT (tenant_id, payment_no) DO NOTHING;

-- Digital Intelligence (EPIC 8): FAQ knowledge base and message template embeddings
-- Requires: migration 029 applied first

-- FAQ entries for the parent query auto-response chatbot
INSERT INTO faq_entries (id, tenant_id, question, answer, category, keywords, is_active) VALUES
    ('a5000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'What is my child''s fee balance?', 'Your child''s current fee balance is available in the parent portal under Fees. For assistance, contact the bursar at +254712345602.', 'fees', ARRAY['fee', 'balance', 'fees', 'pay'], true),
    ('a5000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'How can I pay school fees?', 'Fees can be paid via M-Pesa Paybill, bank transfer, cash at the bursar''s office, or cheque. M-Pesa receipts are sent automatically via SMS.', 'fees', ARRAY['pay', 'mpesa', 'payment', 'fees'], true),
    ('a5000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'When will my child''s results be released?', 'CBC report cards are released at the end of each term. You will receive a notification via WhatsApp when your child''s report is ready.', 'results', ARRAY['results', 'report', 'report card', 'grades'], true),
    ('a5000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', 'What time does the school bus arrive at my stop?', 'Bus arrival times are sent via SMS and WhatsApp when the bus departs school. You can also track the live bus location in the parent portal.', 'transport', ARRAY['bus', 'transport', 'pickup', 'drop'], true),
    ('a5000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001', 'What is the school timetable?', 'The class timetable is available in the parent portal under Timetable. A copy is also sent to your WhatsApp at the start of each term.', 'timetable', ARRAY['timetable', 'schedule', 'classes'], true)
ON CONFLICT (id) DO NOTHING;

-- Message template embeddings for AI template suggestion
INSERT INTO message_template_embeddings (id, tenant_id, template_id, content, purpose, tone, language) VALUES
    ('a6000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'fee_reminder', 'Dear {{parent_name}}, this is a reminder that {{learner_name}} has an outstanding fee balance of KES {{fee_balance}}. Please settle before {{due_date}} to avoid disruption. Thank you.', 'fee reminder', 'formal', 'en'),
    ('a6000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'fee_receipt', 'Dear {{parent_name}}, we have received your payment of KES {{amount}} for {{learner_name}}. Receipt number {{receipt_no}}. Thank you for your prompt payment.', 'payment receipt', 'friendly', 'en'),
    ('a6000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'results_release', 'Dear {{parent_name}}, the CBC report card for {{learner_name}} (Term {{term}} {{year}}) is now available. Please check the parent portal or your WhatsApp for the full report.', 'results release', 'formal', 'en'),
    ('a6000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000001', 'bus_departure', 'Dear {{parent_name}}, the school bus {{bus_reg}} has departed school. ETA to {{stop_name}}: {{eta}}. Route: {{route_name}}.', 'bus departure alert', 'urgent', 'en'),
    ('a6000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000001', 'school_closure', 'Dear {{parent_name}}, school will close for {{holiday}} on {{date}}. Learners will be dismissed at {{time}}. Please arrange pickup accordingly.', 'school closure notice', 'formal', 'en'),
    ('a6000000-0000-0000-0000-000000000006', 'a0000000-0000-0000-0000-000000000001', 'ukumbusho_ada', 'Mpendwa {{parent_name}}, tunakukumbusha kwamba {{learner_name}} ana deni la KES {{fee_balance}}. Tafadhali lipa kabla ya {{due_date}}. Asante.', 'fee reminder', 'formal', 'sw')
ON CONFLICT (id) DO NOTHING;
