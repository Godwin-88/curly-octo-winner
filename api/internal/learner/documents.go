package learner

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// LearnerDocument represents an uploaded document for a learner.
type LearnerDocument struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	LearnerID  uuid.UUID  `json:"learner_id"`
	DocType    string     `json:"doc_type"`
	FileName   string     `json:"file_name"`
	FileURL    string     `json:"file_url"`
	MimeType   string     `json:"mime_type,omitempty"`
	FileSize   int64      `json:"file_size,omitempty"`
	UploadedBy *uuid.UUID `json:"uploaded_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// CreateDocumentRequest is the request payload for uploading a document.
type CreateDocumentRequest struct {
	LearnerID  uuid.UUID  `json:"learner_id"`
	DocType    string     `json:"doc_type"`
	FileName   string     `json:"file_name"`
	FileURL    string     `json:"file_url"`
	MimeType   string     `json:"mime_type,omitempty"`
	FileSize   int64      `json:"file_size,omitempty"`
	UploadedBy *uuid.UUID `json:"uploaded_by,omitempty"`
}

// UploadDocument records a document reference for a learner.
func (s *Service) UploadDocument(ctx context.Context, tenantID uuid.UUID, req CreateDocumentRequest) (*LearnerDocument, error) {
	if req.DocType == "" {
		return nil, fmt.Errorf("doc_type is required")
	}
	if req.FileName == "" {
		return nil, fmt.Errorf("file_name is required")
	}
	if req.FileURL == "" {
		return nil, fmt.Errorf("file_url is required")
	}

	// Ensure learner exists and belongs to tenant
	if _, err := s.GetByID(ctx, tenantID, req.LearnerID); err != nil {
		return nil, err
	}

	var d LearnerDocument
	err := s.pool.QueryRow(ctx, `
		INSERT INTO learner_documents (tenant_id, learner_id, doc_type, file_name, file_url, mime_type, file_size, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, tenant_id, learner_id, doc_type, file_name, file_url, mime_type, file_size, uploaded_by, created_at, updated_at
	`, tenantID, req.LearnerID, req.DocType, req.FileName, req.FileURL,
		req.MimeType, req.FileSize, req.UploadedBy).Scan(
		&d.ID, &d.TenantID, &d.LearnerID, &d.DocType, &d.FileName, &d.FileURL,
		&d.MimeType, &d.FileSize, &d.UploadedBy, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert learner document: %w", err)
	}
	return &d, nil
}

// ListDocuments returns all documents for a learner.
func (s *Service) ListDocuments(ctx context.Context, tenantID, learnerID uuid.UUID) ([]LearnerDocument, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, learner_id, doc_type, file_name, file_url, mime_type, file_size, uploaded_by, created_at, updated_at
		FROM learner_documents
		WHERE tenant_id = $1 AND learner_id = $2
		ORDER BY created_at DESC
	`, tenantID, learnerID)
	if err != nil {
		return nil, fmt.Errorf("query learner documents: %w", err)
	}
	defer rows.Close()

	var docs []LearnerDocument
	for rows.Next() {
		var d LearnerDocument
		if err := rows.Scan(
			&d.ID, &d.TenantID, &d.LearnerID, &d.DocType, &d.FileName, &d.FileURL,
			&d.MimeType, &d.FileSize, &d.UploadedBy, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan learner document: %w", err)
		}
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

// GetDocument returns a single document.
func (s *Service) GetDocument(ctx context.Context, tenantID, docID uuid.UUID) (*LearnerDocument, error) {
	var d LearnerDocument
	err := s.pool.QueryRow(ctx, `
		SELECT id, tenant_id, learner_id, doc_type, file_name, file_url, mime_type, file_size, uploaded_by, created_at, updated_at
		FROM learner_documents
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, docID).Scan(
		&d.ID, &d.TenantID, &d.LearnerID, &d.DocType, &d.FileName, &d.FileURL,
		&d.MimeType, &d.FileSize, &d.UploadedBy, &d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("document not found")
		}
		return nil, fmt.Errorf("query learner document: %w", err)
	}
	return &d, nil
}

// DeleteDocument removes a document record.
func (s *Service) DeleteDocument(ctx context.Context, tenantID, docID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM learner_documents
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, docID)
	if err != nil {
		return fmt.Errorf("delete learner document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("document not found")
	}
	return nil
}
