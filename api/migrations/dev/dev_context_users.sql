-- Local development only (applied by docker-compose, never in production):
-- a second school of the other kind, and one user at each level above a
-- school, so the context bar's three levels can be tried. All sign in with the
-- dev stand-in's password.
--
-- Jua Kali Primary is private; Baraka is a public school. The group user is
-- limited to private schools, so they reach Jua Kali and not Baraka.
INSERT INTO tenants (id, name, slug, ownership)
VALUES ('a0000000-0000-0000-0000-000000000002', 'Baraka Academy', 'baraka', 'public')
ON CONFLICT DO NOTHING;

INSERT INTO platform_users (email, full_name, scope, role)
VALUES ('ops@shule360.test', 'Platform Operator', 'platform', 'super_admin')
ON CONFLICT DO NOTHING;

INSERT INTO platform_users (email, full_name, scope, group_id, role)
SELECT 'trust@juakali.test', 'Private Schools Manager', 'group', g.id, 'principal'
FROM school_groups g WHERE g.ownership = 'private'
ON CONFLICT DO NOTHING;
