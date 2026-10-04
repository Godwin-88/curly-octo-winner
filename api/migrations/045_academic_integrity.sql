-- 045_academic_integrity.sql
-- Report cards that are published and then left alone, absence alerts that
-- are real messages, and Academic as a module a school has or does not have.

-- A published report card says who published it and when. Publishing is the
-- only way a card becomes 'final'; a final card is not edited, regenerated or
-- deleted until a principal reopens it.
ALTER TABLE report_cards
    ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS published_by UUID REFERENCES staff(id) ON DELETE SET NULL;

-- Cards that were already final were published when they were generated.
UPDATE report_cards
SET published_at = generated_at, published_by = generated_by
WHERE status = 'final' AND published_at IS NULL;

-- The SMS that told a parent about an absence is a message like any other, so
-- its delivery can be looked up. sms_notified is true only when one exists.
ALTER TABLE attendance
    ADD COLUMN IF NOT EXISTS alert_message_id UUID REFERENCES messages(id) ON DELETE SET NULL;

-- Nothing was ever sent for the marks recorded so far that claim otherwise:
-- the flag was copied from the request.
UPDATE attendance SET sms_notified = false WHERE sms_notified = true AND alert_message_id IS NULL;

-- report_card_pdfs recorded files that were never stored anywhere (the address
-- was a placeholder). A report card's PDF is now produced when it is asked
-- for. The table is left in place and no longer written to.

-- Academic becomes a module. A school with an explicit list of modules had
-- Academic until now, so it keeps it.
UPDATE tenants
SET modules = array_append(modules, 'academic')
WHERE modules IS NOT NULL AND NOT ('academic' = ANY(modules));

-- The curriculum's core competencies and values are the same in every school.
-- New schools get them when they are registered; schools that have none yet
-- get them here. A school that has entered its own is left alone.
INSERT INTO core_competencies (tenant_id, name)
SELECT t.id, n
FROM tenants t
CROSS JOIN unnest(ARRAY[
    'Communication and Collaboration', 'Critical Thinking and Problem Solving',
    'Creativity and Imagination', 'Citizenship', 'Digital Literacy',
    'Learning to Learn', 'Self-Efficacy']) AS n
WHERE NOT EXISTS (SELECT 1 FROM core_competencies c WHERE c.tenant_id = t.id);

INSERT INTO values (tenant_id, name)
SELECT t.id, n
FROM tenants t
CROSS JOIN unnest(ARRAY[
    'Love', 'Responsibility', 'Respect', 'Unity', 'Peace', 'Patriotism',
    'Social Justice', 'Integrity']) AS n
WHERE NOT EXISTS (SELECT 1 FROM values v WHERE v.tenant_id = t.id);
