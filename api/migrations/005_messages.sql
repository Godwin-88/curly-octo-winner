-- 005_messages.sql
-- Messages (campaigns) table

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('sms', 'whatsapp', 'both')),
    audience_type VARCHAR(30) NOT NULL,
    audience_filter JSONB NOT NULL DEFAULT '{}',
    content_type VARCHAR(20) NOT NULL DEFAULT 'text' CHECK (content_type IN ('text', 'template', 'media')),
    content TEXT NOT NULL,
    template_id VARCHAR(100),
    media_url TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'scheduled', 'sending', 'sent', 'failed')),
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    sent_by UUID REFERENCES staff(id),
    recipient_count INT NOT NULL DEFAULT 0,
    delivered_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_messages_tenant_created ON messages (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_messages_channel ON messages (tenant_id, channel);

-- Trigger
CREATE TRIGGER set_messages_updated_at
    BEFORE UPDATE ON messages
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- RLS
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;

CREATE POLICY "messages_select_policy" ON messages
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "messages_insert_policy" ON messages
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "messages_update_policy" ON messages
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "messages_delete_policy" ON messages
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);