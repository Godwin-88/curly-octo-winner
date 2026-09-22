-- Migration: 032_report_card_pdfs.sql
-- Description: Add report_card_pdfs table for generated PDF files

CREATE TABLE IF NOT EXISTS report_card_pdfs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    report_card_id UUID NOT NULL REFERENCES report_cards(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_url TEXT NOT NULL,
    file_size_bytes BIGINT,
    mime_type VARCHAR(100) DEFAULT 'application/pdf',
    generated_by UUID REFERENCES staff(id),
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_report_card_pdfs_report_card ON report_card_pdfs(report_card_id);
