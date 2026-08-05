-- 001_tenants.sql
-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Updated_at trigger function (applied to all tables)
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    logo_url TEXT,
    subscription_tier VARCHAR(20) NOT NULL DEFAULT 'free',
    wa_phone_number_id VARCHAR(100),
    wa_business_account_id VARCHAR(100),
    at_sender_id VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index on tenants
CREATE INDEX IF NOT EXISTS idx_tenants_created_at ON tenants (created_at DESC);

-- Trigger for updated_at
CREATE TRIGGER set_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- RLS
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;

-- Tenants are managed by super admin; RLS policy allows service role only
CREATE POLICY "tenants_select_policy" ON tenants
    FOR SELECT
    USING (true);

CREATE POLICY "tenants_insert_policy" ON tenants
    FOR INSERT
    WITH CHECK (true);

CREATE POLICY "tenants_update_policy" ON tenants
    FOR UPDATE
    USING (true);

CREATE POLICY "tenants_delete_policy" ON tenants
    FOR DELETE
    USING (true);