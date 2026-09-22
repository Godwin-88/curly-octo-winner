-- Migration: 033_guardian_auth.sql
-- Description: Add PIN-based authentication for guardian/parent portal

ALTER TABLE guardians ADD COLUMN IF NOT EXISTS pin_hash TEXT;

CREATE TABLE IF NOT EXISTS guardian_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    guardian_id UUID NOT NULL REFERENCES guardians(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_guardian_sessions_token ON guardian_sessions(token);
CREATE INDEX IF NOT EXISTS idx_guardian_sessions_guardian ON guardian_sessions(guardian_id);
