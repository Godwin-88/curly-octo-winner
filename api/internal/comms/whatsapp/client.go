package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrOutsideWindow is returned when trying to send free-form text outside the 24h window.
var ErrOutsideWindow = fmt.Errorf("whatsapp: outside 24-hour conversation window")

// WAClient is a Meta WhatsApp Cloud API client.
type WAClient struct {
	token         string
	phoneNumberID string
	baseURL       string
	httpClient    *http.Client
}

// TemplateMessageRequest represents a template message send request.
type TemplateMessageRequest struct {
	To         string
	Template   string
	Language   string // e.g. "en"
	Components []TemplateComponent
}

// TemplateComponent represents a component in a WhatsApp template.
type TemplateComponent struct {
	Type       string          `json:"type"` // "header", "body", "button"
	Parameters []TemplateParam `json:"parameters,omitempty"`
	SubType    string          `json:"sub_type,omitempty"`
	Index      int             `json:"index,omitempty"`
}

// TemplateParam is a single parameter value in a template component.
type TemplateParam struct {
	Type     string      `json:"type"` // "text", "image", "document", "video"
	Text     string      `json:"text,omitempty"`
	Image    *MediaParam `json:"image,omitempty"`
	Document *MediaParam `json:"document,omitempty"`
	Video    *MediaParam `json:"video,omitempty"`
}

// MediaParam represents media parameters for template components.
type MediaParam struct {
	Link string `json:"link"`
}

// WATemplate represents a template submitted to Meta for approval.
type WATemplate struct {
	Name       string `json:"name"`
	Language   string `json:"language"`
	Category   string `json:"category"` // "UTILITY", "MARKETING", "AUTHENTICATION"
	Body       string `json:"body"`
	HeaderType string `json:"header_type,omitempty"` // "TEXT", "IMAGE", "DOCUMENT"
	HeaderText string `json:"header_text,omitempty"`
	FooterText string `json:"footer_text,omitempty"`
}

// NewWAClient creates a new WhatsApp Cloud API client.
func NewWAClient(token, phoneNumberID string) *WAClient {
	return &WAClient{
		token:         token,
		phoneNumberID: phoneNumberID,
		baseURL:       "https://graph.facebook.com/v19.0",
		httpClient:    &http.Client{Timeout: 15 * time.Second},
	}
}

// SendTemplate sends a pre-approved template message.
// Returns the WhatsApp message ID.
func (c *WAClient) SendTemplate(ctx context.Context, req TemplateMessageRequest) (string, error) {
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                req.To,
		"type":              "template",
		"template": map[string]any{
			"name":     req.Template,
			"language": map[string]string{"code": req.Language},
		},
	}

	if len(req.Components) > 0 {
		payload["template"].(map[string]any)["components"] = req.Components
	}

	return c.sendPayload(ctx, payload)
}

// SendText sends a free-form text (only valid within 24h window).
func (c *WAClient) SendText(ctx context.Context, to, body string) (string, error) {
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text":              map[string]string{"body": body},
	}
	return c.sendPayload(ctx, payload)
}

// SendDocument sends a document (PDF) with caption.
func (c *WAClient) SendDocument(ctx context.Context, to, documentURL, filename, caption string) (string, error) {
	document := map[string]any{
		"link": documentURL,
	}
	if filename != "" {
		document["filename"] = filename
	}
	if caption != "" {
		document["caption"] = caption
	}

	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "document",
		"document":          document,
	}
	return c.sendPayload(ctx, payload)
}

// SendImage sends an image with caption.
func (c *WAClient) SendImage(ctx context.Context, to, imageURL, caption string) (string, error) {
	image := map[string]any{"link": imageURL}
	if caption != "" {
		image["caption"] = caption
	}

	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "image",
		"image":             image,
	}
	return c.sendPayload(ctx, payload)
}

// SubmitTemplate submits a new template for Meta approval.
func (c *WAClient) SubmitTemplate(ctx context.Context, template WATemplate) error {
	payload := map[string]any{
		"name":     template.Name,
		"language": template.Language,
		"category": template.Category,
		"components": []map[string]any{
			{
				"type": "BODY",
				"text": template.Body,
			},
		},
	}

	if template.HeaderType != "" {
		header := map[string]any{
			"type":   "HEADER",
			"format": template.HeaderType,
		}
		if template.HeaderText != "" {
			header["text"] = template.HeaderText
		}
		payload["components"].([]map[string]any)[0] = header
	}

	if template.FooterText != "" {
		payload["components"] = append(payload["components"].([]map[string]any), map[string]any{
			"type": "FOOTER",
			"text": template.FooterText,
		})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal template: %w", err)
	}

	// POST to /{WABA_ID}/message_templates
	url := fmt.Sprintf("%s/%s/message_templates", c.baseURL, c.phoneNumberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create template request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute template submit: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read template response: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("template submit error (status %d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// GetTemplateStatus polls approval status for a template name.
// Returns status: "PENDING", "APPROVED", "REJECTED".
func (c *WAClient) GetTemplateStatus(ctx context.Context, templateName string) (string, error) {
	url := fmt.Sprintf("%s/%s/message_templates?name=%s", c.baseURL, c.phoneNumberID, templateName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create template status request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute template status: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read template status response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("template status error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal template status: %w", err)
	}
	if len(result.Data) == 0 {
		return "", fmt.Errorf("template %s not found", templateName)
	}
	return result.Data[0].Status, nil
}

// sendPayload is the internal helper for POSTing to the messages endpoint.
func (c *WAClient) sendPayload(ctx context.Context, payload map[string]any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal message payload: %w", err)
	}

	url := fmt.Sprintf("%s/%s/messages", c.baseURL, c.phoneNumberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create message request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute message send: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read message response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("message send error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal message response: %w", err)
	}
	if len(result.Messages) == 0 {
		return "", fmt.Errorf("message response missing message ID")
	}
	return result.Messages[0].ID, nil
}
