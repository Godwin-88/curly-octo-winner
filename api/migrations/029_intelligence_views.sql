-- 029_intelligence_views.sql
-- Digital Intelligence (EPIC 8): Financial analytics, communication analytics, and AI knowledge base

-- ============================================================
-- FINANCIAL ANALYTICS VIEWS
-- ============================================================

-- Fee collection summary: per term/year, total billed, collected, outstanding, collection rate
CREATE OR REPLACE VIEW fee_collection_summary AS
SELECT
    tenant_id,
    term,
    year,
    COUNT(*) AS invoice_count,
    SUM(total_cents) AS total_billed_cents,
    SUM(discount_cents) AS total_discount_cents,
    SUM(paid_cents) AS total_collected_cents,
    SUM(total_cents - discount_cents - paid_cents) AS outstanding_cents,
    CASE
        WHEN SUM(total_cents - discount_cents) > 0
        THEN ROUND(100.0 * SUM(paid_cents) / NULLIF(SUM(total_cents - discount_cents), 0), 2)
        ELSE 0
    END AS collection_rate
FROM invoices
WHERE status != 'void'
GROUP BY tenant_id, term, year;

-- Payment channel breakdown: per term/year, amount by channel
CREATE OR REPLACE VIEW payment_channel_breakdown AS
SELECT
    p.tenant_id,
    i.term,
    i.year,
    p.channel,
    COUNT(*) AS payment_count,
    SUM(p.amount_cents) AS total_cents
FROM payments p
JOIN invoices i ON i.id = p.invoice_id
WHERE p.status = 'completed'
GROUP BY p.tenant_id, i.term, i.year, p.channel;

-- Fee defaulters: learners with outstanding balances per term/year
CREATE OR REPLACE VIEW fee_defaulters AS
SELECT
    tenant_id,
    learner_id,
    term,
    year,
    invoice_number,
    total_cents,
    discount_cents,
    paid_cents,
    (total_cents - discount_cents - paid_cents) AS balance_cents,
    due_date,
    status
FROM invoices
WHERE status IN ('unpaid', 'partially_paid', 'overdue')
  AND (total_cents - discount_cents - paid_cents) > 0;

-- Monthly collection trend: per month, total collected
CREATE OR REPLACE VIEW monthly_collection_trend AS
SELECT
    tenant_id,
    date_trunc('month', paid_at) AS month,
    COUNT(*) AS payment_count,
    SUM(amount_cents) AS total_cents
FROM payments
WHERE status = 'completed' AND paid_at IS NOT NULL
GROUP BY tenant_id, date_trunc('month', paid_at);

-- ============================================================
-- COMMUNICATION ANALYTICS VIEWS
-- ============================================================

-- Campaign delivery summary: per message, delivery stats
CREATE OR REPLACE VIEW campaign_delivery_summary AS
SELECT
    m.id AS message_id,
    m.tenant_id,
    m.channel,
    m.audience_type,
    m.status,
    m.sent_at,
    m.recipient_count,
    m.delivered_count,
    m.failed_count,
    CASE
        WHEN m.recipient_count > 0
        THEN ROUND(100.0 * m.delivered_count / NULLIF(m.recipient_count, 0), 2)
        ELSE 0
    END AS delivery_rate
FROM messages m;

-- Channel reach: per channel, total messages and recipients
CREATE OR REPLACE VIEW channel_reach AS
SELECT
    tenant_id,
    channel,
    COUNT(*) AS campaign_count,
    SUM(recipient_count) AS total_recipients,
    SUM(delivered_count) AS total_delivered,
    SUM(failed_count) AS total_failed
FROM messages
WHERE status = 'sent'
GROUP BY tenant_id, channel;

-- Failed number analysis: per message, failed logs with error codes
CREATE OR REPLACE VIEW failed_number_analysis AS
SELECT
    ml.tenant_id,
    ml.message_id,
    m.channel,
    ml.phone,
    ml.error_code,
    ml.error_message,
    ml.created_at
FROM message_logs ml
JOIN messages m ON m.id = ml.message_id
WHERE ml.status = 'failed';

-- ============================================================
-- AI KNOWLEDGE BASE (Upstash Vector)
-- ============================================================

-- FAQ knowledge base: parent queries and automated responses
CREATE TABLE IF NOT EXISTS faq_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    category VARCHAR(50) NOT NULL DEFAULT 'general'
        CHECK (category IN ('fees', 'results', 'transport', 'timetable', 'general')),
    keywords TEXT[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_faq_entries_tenant ON faq_entries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_faq_entries_category ON faq_entries (tenant_id, category);

CREATE TRIGGER set_faq_entries_updated_at
    BEFORE UPDATE ON faq_entries
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE faq_entries ENABLE ROW LEVEL SECURITY;

CREATE POLICY "faq_entries_select_policy" ON faq_entries
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "faq_entries_insert_policy" ON faq_entries
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "faq_entries_update_policy" ON faq_entries
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "faq_entries_delete_policy" ON faq_entries
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Message template embeddings: stores vector metadata for semantic search
CREATE TABLE IF NOT EXISTS message_template_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    template_id VARCHAR(100),
    content TEXT NOT NULL,
    purpose VARCHAR(255),
    tone VARCHAR(20) NOT NULL DEFAULT 'formal'
        CHECK (tone IN ('formal', 'friendly', 'urgent')),
    language VARCHAR(10) NOT NULL DEFAULT 'en'
        CHECK (language IN ('en', 'sw')),
    vector_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_message_template_embeddings_tenant ON message_template_embeddings (tenant_id);

ALTER TABLE message_template_embeddings ENABLE ROW LEVEL SECURITY;

CREATE POLICY "message_template_embeddings_select_policy" ON message_template_embeddings
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "message_template_embeddings_insert_policy" ON message_template_embeddings
    FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "message_template_embeddings_update_policy" ON message_template_embeddings
    FOR UPDATE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

CREATE POLICY "message_template_embeddings_delete_policy" ON message_template_embeddings
    FOR DELETE
    USING (tenant_id = current_setting('app.tenant_id')::uuid);