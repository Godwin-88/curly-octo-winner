-- 040_school_groups.sql
-- Three levels of context: Platform > School group > School.
--
-- Until now every signed-in user belonged to exactly one school (staff.tenant_id)
-- and nothing sat above a school. This adds:
--
--  - school_groups: an owner of several schools (a sponsor, a diocese, a chain).
--    A school belongs to at most one group; most belong to none.
--  - platform_users: people who are not staff of any one school. `platform`
--    scope may open any school; `group` scope may open the schools of its group.
--
-- The rule the API enforces (middleware.Auth): the client names the school it
-- wants to work in, the session narrows that to what it is allowed, and a
-- school outside it answers 404 — the same as a school that does not exist.
-- A school-staff session is always pinned to its own school and cannot widen
-- itself.
--
-- Neither table carries a tenant_id, so the per-tenant RLS policies used by the
-- rest of the schema do not apply. RLS is enabled with no policy: only the
-- service role (the API) can read or write them.

CREATE TABLE IF NOT EXISTS school_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER set_school_groups_updated_at
    BEFORE UPDATE ON school_groups
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE school_groups ENABLE ROW LEVEL SECURITY;

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS group_id UUID REFERENCES school_groups(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_tenants_group ON tenants (group_id);

CREATE TABLE IF NOT EXISTS platform_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    scope VARCHAR(20) NOT NULL CHECK (scope IN ('platform', 'group')),
    -- Required for, and only for, group scope.
    group_id UUID REFERENCES school_groups(id) ON DELETE CASCADE,
    -- What this person may do inside a school they open, in the same terms as
    -- staff_role, so the existing role checks apply unchanged.
    role VARCHAR(30) NOT NULL DEFAULT 'principal'
        CHECK (role IN ('super_admin', 'principal', 'bursar', 'hr', 'transport_manager', 'teacher')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((scope = 'group') = (group_id IS NOT NULL))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_platform_users_email ON platform_users (lower(email));

CREATE TRIGGER set_platform_users_updated_at
    BEFORE UPDATE ON platform_users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE platform_users ENABLE ROW LEVEL SECURITY;

-- A message sent by a platform or group user has no staff row to point at.
ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS sent_by_operator UUID REFERENCES platform_users(id) ON DELETE SET NULL;
