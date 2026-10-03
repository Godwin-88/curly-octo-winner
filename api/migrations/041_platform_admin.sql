-- 041_platform_admin.sql
-- Lets a platform or group user work in every screen of a school, and gives
-- the platform the records it needs to manage those users.
--
-- Inside a school, almost everything a person does is recorded against a
-- staff row (created_by, sent_by, approved_by ... all reference staff). A
-- platform or group user has no such row, so those screens refused them.
--
-- Rather than loosen every one of those foreign keys, a platform user gets one
-- staff row per school they open, created the first time they open it and
-- linked back here. The school's records then show a real, named actor
-- ("Jane Doe (Shule360 platform)"), and the existing integrity rules hold.
--
-- These rows are not staff of the school: they cannot sign in (sign-in ignores
-- them and their email is not a real address), and they are left out of the
-- staff directory and head counts.

ALTER TABLE staff
    ADD COLUMN IF NOT EXISTS platform_user_id UUID REFERENCES platform_users(id) ON DELETE CASCADE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_staff_platform_user
    ON staff (tenant_id, platform_user_id)
    WHERE platform_user_id IS NOT NULL;

-- The Supabase Auth user behind a platform user, so a password can be reset.
ALTER TABLE platform_users
    ADD COLUMN IF NOT EXISTS supabase_user_id UUID;
