-- Local development only (applied by docker-compose, never in production):
-- a school group, a second school, and one user at each level above a school,
-- so the context bar's three levels can be tried. All sign in with the dev
-- stand-in's password.
INSERT INTO school_groups (id, name, slug)
VALUES ('b0000000-0000-0000-0000-000000000001', 'Jua Kali Schools Trust', 'juakali-trust')
ON CONFLICT DO NOTHING;

UPDATE tenants SET group_id = 'b0000000-0000-0000-0000-000000000001'
WHERE id = 'a0000000-0000-0000-0000-000000000001';

INSERT INTO tenants (id, name, slug)
VALUES ('a0000000-0000-0000-0000-000000000002', 'Baraka Academy', 'baraka')
ON CONFLICT DO NOTHING;

INSERT INTO platform_users (email, full_name, scope, role)
VALUES ('ops@shule360.test', 'Platform Operator', 'platform', 'super_admin')
ON CONFLICT DO NOTHING;

INSERT INTO platform_users (email, full_name, scope, group_id, role)
VALUES ('trust@juakali.test', 'Trust Director', 'group', 'b0000000-0000-0000-0000-000000000001', 'principal')
ON CONFLICT DO NOTHING;
