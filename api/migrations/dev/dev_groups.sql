-- Local development only. For a local database made before there were exactly
-- two groups: Baraka becomes the public school, and the group user (whom
-- migration 046 deactivated on moving them) is the private schools' user.
UPDATE tenants SET ownership = 'public'
WHERE id = 'a0000000-0000-0000-0000-000000000002';

UPDATE platform_users
SET group_id = (SELECT id FROM school_groups WHERE ownership = 'private'),
    full_name = 'Private Schools Manager',
    is_active = true
WHERE lower(email) = 'trust@juakali.test';
