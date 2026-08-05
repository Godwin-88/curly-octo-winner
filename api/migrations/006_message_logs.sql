-- 006_message_logs.sql
-- Per-recipient delivery log table

CREATE TABLE IF NOT EXISTS message_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    recipient_type VARCHAR(20) NOT NULL CHECK (recipient_type IN ('guardian', 'staff', 'supplier')),
    recipient_id UUID,
    phone VARCHAR(15) NOT NULL,
    channel VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'delivered', 'failed', 'read')),
    provider_message_id VARCHAR(255),
    delivered_at TIMESTAMPTZ,
    read_at TIMESTAMPTZ,
    error_code VARCHAR(50),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_message_logs_tenant_created ON message_logs (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_message_logs_message_status ON message_logs (message_id, status);
CREATE INDEX IF NOT EXISTS idx_message_logs_recipient ON message_logs (recipient_id);
CREATE INDEX IF NOT EXISTS idx_message_logs_provider ON message_logs (provider_message_id);

-- Trigger
CREATE TRIGGER set_message_logs_updated_at
    BEFORE UPDATE ON message_logs
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- RLS
ALTER TABLE message_logs ENABLE ROW LEVEL SECURITY;

CREATE POLICY "message_logs_select_policy" ON message_logs
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "message_logs_insert_policy" ON message_logs
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "message_logs_update_policy" ON message_logs
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "message_logs_delete_policy" ON message_logs
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);