package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ATClient is an Africa's Talking SMS REST client.
type ATClient struct {
	apiKey     string
	username   string
	baseURL    string
	senderID   string
	httpClient *http.Client
}

// BulkSMSRequest represents a bulk SMS send request.
type BulkSMSRequest struct {
	To      []string // E.164 format, e.g. +254712345678
	Message string
	From    string // Sender ID
}

// SMSResult represents the result for a single phone number.
type SMSResult struct {
	Phone     string
	Status    string // Success | Failed
	MessageID string
	Cost      string
}

// CostEstimate represents an estimated cost for a send operation.
type CostEstimate struct {
	RecipientCount int
	SMSUnits       int
	EstimatedKES   float64
}

// NewATClient creates a new Africa's Talking client.
// isProduction switches between sandbox and production base URLs.
func NewATClient(apiKey, username, senderID string, isProduction bool) *ATClient {
	baseURL := "https://api.sandbox.africastalking.com"
	if isProduction {
		baseURL = "https://api.africastalking.com"
	}

	return &ATClient{
		apiKey:     apiKey,
		username:   username,
		baseURL:    baseURL,
		senderID:   senderID,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// NormalizeKenyanPhone normalizes Kenyan phone numbers to E.164 format (+254...).
// Handles: 07..., 01..., +254..., 254..., etc.
func NormalizeKenyanPhone(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("empty phone number")
	}

	// Remove spaces, dashes, parentheses
	trimmed = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", ".", "").Replace(trimmed)

	// Handle +254... format
	if strings.HasPrefix(trimmed, "+254") {
		if len(trimmed) != 13 {
			return "", fmt.Errorf("invalid +254 phone number length: %s", raw)
		}
		return trimmed, nil
	}

	// Handle 254... format
	if strings.HasPrefix(trimmed, "254") {
		if len(trimmed) != 12 {
			return "", fmt.Errorf("invalid 254 phone number length: %s", raw)
		}
		return "+" + trimmed, nil
	}

	// Handle 07... format (Safaricom/Airtel/Telkom)
	if strings.HasPrefix(trimmed, "07") {
		if len(trimmed) != 10 {
			return "", fmt.Errorf("invalid 07 phone number length: %s", raw)
		}
		return "+254" + trimmed[1:], nil
	}

	// Handle 01... format (new Safaricom numbering)
	if strings.HasPrefix(trimmed, "01") {
		if len(trimmed) != 10 {
			return "", fmt.Errorf("invalid 01 phone number length: %s", raw)
		}
		return "+254" + trimmed[1:], nil
	}

	// Handle 7... or 1... format (missing leading 0)
	if len(trimmed) == 9 && (strings.HasPrefix(trimmed, "7") || strings.HasPrefix(trimmed, "1")) {
		return "+254" + trimmed, nil
	}

	return "", fmt.Errorf("unrecognized phone number format: %s", raw)
}

// ATBulkResponse is the response from AT's bulk SMS API.
type ATBulkResponse struct {
	SMSMessageData struct {
		Message    string `json:"Message"`
		Recipients []struct {
			StatusCode int    `json:"statusCode"`
			Number     string `json:"number"`
			Cost       string `json:"cost"`
			Status     string `json:"status"`
			MessageID  string `json:"messageId"`
		} `json:"Recipients"`
	} `json:"SMSMessageData"`
}

// SendBulk sends SMS to a list of phone numbers. Returns per-number results.
func (c *ATClient) SendBulk(ctx context.Context, req BulkSMSRequest) ([]SMSResult, error) {
	form := url.Values{}
	form.Set("username", c.username)
	form.Set("to", strings.Join(req.To, ","))
	form.Set("message", req.Message)
	if req.From != "" {
		form.Set("from", req.From)
	} else if c.senderID != "" {
		form.Set("from", c.senderID)
	}

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/version1/messaging", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create AT request: %w", err)
	}
	apiReq.Header.Set("apiKey", c.apiKey)
	apiReq.Header.Set("Accept", "application/json")
	apiReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(apiReq)
	if err != nil {
		return nil, fmt.Errorf("execute AT send: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read AT response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AT API error (status %d): %s", resp.StatusCode, string(body))
	}

	var atResp ATBulkResponse
	if err := json.Unmarshal(body, &atResp); err != nil {
		return nil, fmt.Errorf("unmarshal AT response: %w", err)
	}

	results := make([]SMSResult, 0, len(atResp.SMSMessageData.Recipients))
	for _, r := range atResp.SMSMessageData.Recipients {
		status := "Failed"
		if r.StatusCode == 101 || r.Status == "Success" {
			status = "Success"
		}
		results = append(results, SMSResult{
			Phone:     r.Number,
			Status:    status,
			MessageID: r.MessageID,
			Cost:      r.Cost,
		})
	}

	return results, nil
}

// EstimateCost returns estimated KES cost without sending.
// Kenya SMS rates (approx): ~0.80 KES per SMS unit (varies by network).
// A single SMS is 160 characters; 1 unit = 160 chars.
func (c *ATClient) EstimateCost(recipients int, messageUnits int) CostEstimate {
	ratePerUnit := 0.80 // KES per SMS unit (approximate sandbox rate)
	estimated := float64(recipients*messageUnits) * ratePerUnit
	return CostEstimate{
		RecipientCount: recipients,
		SMSUnits:       messageUnits,
		EstimatedKES:   estimated,
	}
}

// CalculateSMSUnits returns the number of SMS units for a message.
// 1 unit = 160 chars, 2 units = 320, 3 units = 480, etc.
func CalculateSMSUnits(message string) int {
	length := len([]rune(message))
	if length == 0 {
		return 0
	}
	units := (length + 159) / 160
	if units > 3 {
		units = 3 // Max 3 units per spec
	}
	return units
}

// parseCost parses a currency string like "KES 0.80" to a float.
func parseCost(s string) float64 {
	s = strings.TrimPrefix(strings.TrimSpace(s), "KES ")
	s = strings.TrimSuffix(s, "/SMS")
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}
