-- 039_sms_delivery.sql
-- Makes bulk SMS recordable before it is sent, and its delivery state honest.
--
-- Motivation: a send used to insert `sent_by = uuid.Nil` (rejected by the FK to
-- staff), then hand the job to a Redis queue nothing consumed. Nothing was ever
-- delivered, and nothing recorded who should have received it.
--
-- The model after this migration:
--  - one `message_logs` row per recipient is written, as `pending`, in the same
--    transaction as the message — before Africa's Talking is called;
--  - `attempted_at` is stamped (and committed) immediately before the provider
--    call. A row that is `pending` with `attempted_at` set after a restart has
--    an unknown outcome and is never re-sent automatically: a parent must not
--    receive (and the school must not pay for) the same SMS twice;
--  - `sent` means the provider accepted it; `delivered` is only ever set by a
--    delivery receipt.

ALTER TABLE message_logs
    ADD COLUMN IF NOT EXISTS recipient_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS attempted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS sent_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS cost_cents INT,
    -- The personalised text for this recipient. NULL when the message has no
    -- variables, in which case messages.content is what was sent.
    ADD COLUMN IF NOT EXISTS rendered_content TEXT;

-- The contact book (036) is a recipient type of its own.
ALTER TABLE message_logs DROP CONSTRAINT IF EXISTS message_logs_recipient_type_check;
ALTER TABLE message_logs ADD CONSTRAINT message_logs_recipient_type_check
    CHECK (recipient_type IN ('guardian', 'staff', 'supplier', 'contact'));

-- One row per phone per message: the same parent is never texted twice by one
-- send, however many learners or contact entries point at the number.
CREATE UNIQUE INDEX IF NOT EXISTS idx_message_logs_message_phone
    ON message_logs (message_id, phone);

-- The dispatcher's work list.
CREATE INDEX IF NOT EXISTS idx_message_logs_pending
    ON message_logs (message_id, created_at)
    WHERE status = 'pending';

ALTER TABLE messages
    -- Chosen by the client per send attempt, so a double-click or a retried
    -- request returns the first message instead of sending (and billing) again.
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(100);

CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_idempotency
    ON messages (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- A cancelled schedule is not a failure.
ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_status_check;
ALTER TABLE messages ADD CONSTRAINT messages_status_check
    CHECK (status IN ('draft', 'scheduled', 'sending', 'sent', 'failed', 'cancelled'));

CREATE INDEX IF NOT EXISTS idx_messages_dispatch
    ON messages (status, scheduled_at)
    WHERE status IN ('scheduled', 'sending');
