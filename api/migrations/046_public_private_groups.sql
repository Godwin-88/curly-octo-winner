-- 046_public_private_groups.sql
-- There are exactly two school groups: public schools and private schools.
--
-- Until now a group was a free-standing thing (a trust, a chain) that a school
-- might or might not belong to, and whether a school was public or private
-- was a separate fact. They are now the same fact: every school is in the
-- group for its kind, and changing its kind moves it.

ALTER TABLE school_groups
    ADD COLUMN IF NOT EXISTS ownership VARCHAR(10) CHECK (ownership IN ('public', 'private')),
    ADD COLUMN IF NOT EXISTS description TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS uq_school_groups_ownership ON school_groups (ownership);

INSERT INTO school_groups (name, slug, ownership, description) VALUES
    ('Public schools', 'public', 'public',
     'Government schools: funded by the state with free or low-cost tuition, the national curriculum, and teachers posted by the Teachers Service Commission.'),
    ('Private schools', 'private', 'private',
     'Schools funded by the fees parents pay: they set their own fees and may teach other curricula alongside the CBC.')
ON CONFLICT (slug) DO UPDATE
    SET ownership = EXCLUDED.ownership,
        description = COALESCE(school_groups.description, EXCLUDED.description);

-- A user who was limited to one of the old groups is moved to the group of
-- that old group's schools, and deactivated: the new group holds every school
-- of that kind, which is more than they were given. A platform administrator
-- reactivates them if that is intended.
UPDATE platform_users u
SET group_id = n.id, is_active = false
FROM school_groups old, school_groups n
WHERE u.group_id = old.id
  AND old.ownership IS NULL
  AND n.ownership = COALESCE(
        (SELECT t.ownership FROM tenants t WHERE t.group_id = old.id ORDER BY t.created_at LIMIT 1),
        'private');

UPDATE tenants t
SET group_id = g.id
FROM school_groups g
WHERE g.ownership = t.ownership AND t.group_id IS DISTINCT FROM g.id;

DELETE FROM school_groups WHERE ownership IS NULL;

ALTER TABLE school_groups ALTER COLUMN ownership SET NOT NULL;

-- The group follows the kind of school, whoever writes the row and however.
CREATE OR REPLACE FUNCTION tenants_group_from_ownership() RETURNS trigger AS $$
BEGIN
    SELECT id INTO NEW.group_id FROM school_groups WHERE ownership = NEW.ownership;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_tenants_group ON tenants;
CREATE TRIGGER set_tenants_group
    BEFORE INSERT OR UPDATE OF ownership, group_id ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION tenants_group_from_ownership();
