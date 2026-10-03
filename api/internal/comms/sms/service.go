package sms

import (
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SMSService backs the SMS template routes. Sending is not done here: see
// comms.CommsService.CreateAndSend and comms.Dispatcher.
type SMSService struct {
	pool *pgxpool.Pool
}

func NewSMSService(pool *pgxpool.Pool) *SMSService {
	return &SMSService{pool: pool}
}

var variablePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z_]+)\s*\}\}`)

// Variables returns the distinct {{names}} used in a template, in order.
func Variables(template string) []string {
	seen := map[string]struct{}{}
	var names []string
	for _, m := range variablePattern.FindAllStringSubmatch(template, -1) {
		name := strings.ToLower(m[1])
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

// InjectVariables replaces every {{name}} that has a value in data. Names are
// matched case-insensitively and may be padded ({{ parent_name }}). A name
// with no value is left in place, so the caller can detect it.
func InjectVariables(template string, data map[string]string) string {
	return variablePattern.ReplaceAllStringFunc(template, func(match string) string {
		name := strings.ToLower(variablePattern.FindStringSubmatch(match)[1])
		if value, ok := data[name]; ok {
			return value
		}
		return match
	})
}
