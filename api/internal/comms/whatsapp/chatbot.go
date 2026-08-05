package whatsapp

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Guardian represents a guardian in the chatbot handlers.
type Guardian struct {
	ID           uuid.UUID `json:"id"`
	FullName     string    `json:"full_name"`
	PhonePrimary string    `json:"phone_primary"`
	PhoneWA      string    `json:"phone_wa"`
}

// Learner represents a learner in the chatbot handlers.
type Learner struct {
	ID         uuid.UUID `json:"id"`
	FullName   string    `json:"full_name"`
	Grade      string    `json:"grade"`
	Stream     string    `json:"stream"`
	FeeBalance float64   `json:"fee_balance"`
}

// Tenant represents the tenant in chatbot handlers.
type Tenant struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ChatbotHandlerFunc is a function that handles a matched chatbot rule.
type ChatbotHandlerFunc func(ctx context.Context, tenant Tenant, guardian Guardian) (string, error)

// ChatbotRule represents a single keyword-based chatbot rule.
type ChatbotRule struct {
	Keywords []string // case-insensitive match
	Handler  ChatbotHandlerFunc
}

// Chatbot is the rules-based chatbot engine.
type Chatbot struct {
	rules            []ChatbotRule
	unmatchedHandler ChatbotHandlerFunc
}

// NewChatbot creates a new chatbot with the default rule set.
func NewChatbot() *Chatbot {
	cb := &Chatbot{
		rules: []ChatbotRule{
			{
				Keywords: []string{"fee", "fees", "fee balance", "ada"},
				Handler:  feeBalanceHandler,
			},
			{
				Keywords: []string{"results", "report", "ripoti", "matokeo"},
				Handler:  resultsHandler,
			},
			{
				Keywords: []string{"transport", "bus", "basi"},
				Handler:  transportHandler,
			},
			{
				Keywords: []string{"timetable", "ratiba"},
				Handler:  timetableHandler,
			},
		},
		unmatchedHandler: unmatchedHandler,
	}
	return cb
}

// Handle processes an inbound message and returns a reply or "" if no reply.
// Returns (reply, matched bool, err).
func (c *Chatbot) Handle(ctx context.Context, tenant Tenant, guardian Guardian, message string) (string, bool, error) {
	normalized := strings.ToLower(strings.TrimSpace(message))

	// Extract keywords
	extracted := extractKeywords(normalized)

	if len(extracted) == 0 {
		return "", false, nil
	}

	for _, rule := range c.rules {
		for _, keyword := range rule.Keywords {
			if containsKeyword(normalized, keyword) {
				reply, err := rule.Handler(ctx, tenant, guardian)
				if err != nil {
					return "", true, fmt.Errorf("chatbot rule %q: %w", keyword, err)
				}
				return reply, true, nil
			}
		}
	}

	// No rule matched after keyword extraction
	return "", false, nil
}

// HandleUnmatched returns the fallback response for unmatched messages.
func (c *Chatbot) HandleUnmatched(ctx context.Context, tenant Tenant, guardian Guardian, message string) (string, error) {
	return c.unmatchedHandler(ctx, tenant, guardian)
}

// extractKeywords extracts known keywords from a message.
func extractKeywords(message string) []string {
	known := []string{"fee", "fees", "results", "report", "transport", "bus", "timetable", "ada", "ripoti", "matokeo", "ratiba", "basi"}
	var found []string
	for _, kw := range known {
		if strings.Contains(message, kw) {
			found = append(found, kw)
		}
	}
	return found
}

// containsKeyword checks if a keyword is present in the message.
func containsKeyword(message, keyword string) bool {
	return strings.Contains(message, keyword)
}

// feeBalanceHandler returns the learner's current fee balance (stub with mock data).
func feeBalanceHandler(ctx context.Context, tenant Tenant, guardian Guardian) (string, error) {
	// TODO: Query fee_invoices for guardian's learners
	// Stub: return mock balance data
	return fmt.Sprintf(`Dear %s,

Here is the current fee balance for your ward:

*Learner:* Sandbox Learner
*Grade:* Grade 4
*Term 1 Balance:* KES 12,500
*Transport (optional):* KES 3,000
*Total Outstanding:* KES 15,500

Please pay at the school bursar's office or via M-Pesa Paybill 123456 (Account: learner UPI).

Thank you.`,
		guardian.FullName), nil
}

// resultsHandler returns info about results availability.
func resultsHandler(ctx context.Context, tenant Tenant, guardian Guardian) (string, error) {
	return "Your child's latest report will be available at term end. Contact the school office for more information.", nil
}

// transportHandler returns transport program info.
func transportHandler(ctx context.Context, tenant Tenant, guardian Guardian) (string, error) {
	return "Bus tracking is live. Reply STOP to opt out of transport alerts.", nil
}

// timetableHandler returns timetable info.
func timetableHandler(ctx context.Context, tenant Tenant, guardian Guardian) (string, error) {
	return "Please contact your class teacher for the current timetable.", nil
}

// unmatchedHandler is the fallback for unrecognized queries.
func unmatchedHandler(ctx context.Context, tenant Tenant, guardian Guardian) (string, error) {
	return "Your message has been received. A staff member will respond shortly.", nil
}
