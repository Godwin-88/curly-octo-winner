-- 008_wa_messages.sql
-- WhatsApp message table (thread messages)

CREATE TABLE IF NOT EXISTS wa_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES wa_conversations(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    content_type VARCHAR(20) NOT NULL CHECK (content_type IN ('text', 'image', 'document', 'video', 'button', 'template')),
    content JSONB NOT NULL DEFAULT '{}',
    wa_message_id VARCHAR(255) UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_wa_messages_tenant_created ON wa_messages (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wa_messages_conversation_time ON wa_messages (conversation_id, timestamp);

-- Trigger
CREATE TRIGGER set_wa_messages_updated_at
    BEFORE UPDATE ON wa_messages
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- RLS
ALTER TABLE wa_messages ENABLE ROW LEVEL SECURITY;

CREATE POLICY "wa_messages_select_policy" ON wa_messages
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "wa_messages_insert_policy" ON wa_messages
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "wa_messages_update_policy" ON wa_messages
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "wa_messages_delete_policy" ON wa_messages
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Enable Realtime for frontend inbox updates
ALTER TABLE wa_messages REPLICA IDENTITY FULL;