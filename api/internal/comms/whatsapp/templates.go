package whatsapp

import (
	"context"
	"fmt"
	"time"
)

// TemplateStatus represents the approval status of a WhatsApp template.
type TemplateStatus string

const (
	TemplateStatusPending  TemplateStatus = "PENDING"
	TemplateStatusApproved TemplateStatus = "APPROVED"
	TemplateStatusRejected TemplateStatus = "REJECTED"
)

// TemplateMessage is a stored template record in the database.
type TemplateMessage struct {
	ID             string
	Name           string
	Category       string
	Language       string
	Body           string
	HeaderType     string
	HeaderText     string
	FooterText     string
	Status         TemplateStatus
	CreatedAt      time.Time
	RejectedReason string `json:"rejected_reason,omitempty"`
}

// TemplateService handles template submission and status polling.
type TemplateService struct {
	client *WAClient
}

// NewTemplateService creates a new template service.
func NewTemplateService(client *WAClient) *TemplateService {
	return &TemplateService{client: client}
}

// SubmitTemplateForApproval submits a template and stores it.
func (s *TemplateService) SubmitTemplateForApproval(ctx context.Context, template WATemplate) (*TemplateMessage, error) {
	if err := s.client.SubmitTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("submit template: %w", err)
	}

	return &TemplateMessage{
		Name:       template.Name,
		Category:   template.Category,
		Language:   template.Language,
		Body:       template.Body,
		HeaderType: template.HeaderType,
		HeaderText: template.HeaderText,
		FooterText: template.FooterText,
		Status:     TemplateStatusPending,
		CreatedAt:  time.Now(),
	}, nil
}

// PollTemplateStatus checks the approval status of a template.
func (s *TemplateService) PollTemplateStatus(ctx context.Context, templateName string) (TemplateStatus, error) {
	status, err := s.client.GetTemplateStatus(ctx, templateName)
	if err != nil {
		return "", fmt.Errorf("get template status: %w", err)
	}

	switch status {
	case "APPROVED":
		return TemplateStatusApproved, nil
	case "REJECTED":
		return TemplateStatusRejected, nil
	default:
		return TemplateStatusPending, nil
	}
}

// IsApproved returns true if the template has been approved by Meta.
func (s *TemplateService) IsApproved(ctx context.Context, templateName string) (bool, error) {
	status, err := s.PollTemplateStatus(ctx, templateName)
	if err != nil {
		return false, err
	}
	return status == TemplateStatusApproved, nil
}
