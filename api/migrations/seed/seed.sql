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