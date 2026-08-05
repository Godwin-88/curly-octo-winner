-- 007_wa_conversations.sql
-- WhatsApp conversation table

CREATE TABLE IF NOT EXISTS wa_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    guardian_id UUID REFERENCES guardians(id),
    wa_contact_phone VARCHAR(15) NOT NULL,
    wa_contact_name VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'waiting', 'resolved')),
    assigned_to UUID REFERENCES staff(id),
    last_message_at TIMESTAMPTZ,
    last_message_preview TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, wa_contact_phone)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_wa_conv_tenant_created ON wa_conversations (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wa_conv_status_last ON wa_conversations (tenant_id, status, last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_wa_conv_assigned ON wa_conversations (tenant_id, assigned_to);

-- Trigger
CREATE TRIGGER set_wa_conversations_updated_at
    BEFORE UPDATE ON wa_conversations
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- RLS
ALTER TABLE wa_conversations ENABLE ROW LEVEL SECURITY;

CREATE POLICY "wa_conversations_select_policy" ON wa_conversations
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "wa_conversations_insert_policy" ON wa_conversations
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "wa_conversations_update_policy" ON wa_conversations
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "wa_conversations_delete_policy" ON wa_conversations
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Enable Realtime for frontend inbox updates
ALTER TABLE wa_conversations REPLICA IDENTITY FULL;