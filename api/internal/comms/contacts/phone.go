// Package contacts manages the school's contact book: the people that can be
// messaged through Communications (SMS/WhatsApp), independent of whether they
// have a learner account.
package contacts

import (
	"fmt"
	"strings"
)

// NormalizePhone converts a Kenyan phone number in any of the formats a school
// actually types into E.164 (+2547XXXXXXXX), which is the only form stored.
//
// Accepted:
//
//	0712345678        local
//	+254 712 345 678  international, with spaces/dashes
//	254712345678      country code without +
//	712345678         bare subscriber number
//	0712-345-678      punctuation
//
// Rejected: short numbers, landlines, and numbers from other country codes.
// A clear error here is worth more than a silent drop: these are the numbers
// the school is about to spend money texting.
func NormalizePhone(raw string) (string, error) {
	cleaned := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '-', '(', ')', '.', '/', '\t':
			return -1
		}
		return r
	}, strings.TrimSpace(raw))

	if cleaned == "" {
		return "", fmt.Errorf("phone number is required")
	}

	// Strip a leading + and normalize the country code.
	digits := strings.TrimPrefix(cleaned, "+")

	switch {
	case strings.HasPrefix(digits, "254"):
		digits = strings.TrimPrefix(digits, "254")
	case strings.HasPrefix(digits, "0"):
		// Local format: 0712345678 -> 712345678
		digits = strings.TrimPrefix(digits, "0")
	}

	// Kenyan mobile numbers are exactly 9 digits starting with 7. Landlines
	// (020, 011, ...) and other ranges are rejected: a school should not spend
	// money texting a switchboard.
	if len(digits) != 9 {
		return "", fmt.Errorf("%q is not a valid Kenyan phone number (expected 9 digits after the country code)", strings.TrimSpace(raw))
	}
	if digits[0] != '7' {
		return "", fmt.Errorf("%q does not look like a Kenyan mobile number (it should start with 07)", strings.TrimSpace(raw))
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("%q is not a valid phone number (digits only)", strings.TrimSpace(raw))
		}
	}

	return "+254" + digits, nil
}

// ValidateEmail performs a deliberately shallow check: reject the obvious
// mistakes without pretending to implement RFC 5322.
func ValidateEmail(raw string) error {
	email := strings.TrimSpace(raw)
	if email == "" {
		return nil
	}
	at := strings.Index(email, "@")
	if at <= 0 || at == len(email)-1 || strings.Count(email, "@") != 1 {
		return fmt.Errorf("%q is not a valid email address", email)
	}
	domain := email[at+1:]
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return fmt.Errorf("%q is not a valid email address", email)
	}
	return nil
}

// NormalizeTags trims, lowercases, de-duplicates and caps the tag list, and
// rejects the characters that would break the audience filter URL.
func NormalizeTags(tags []string) ([]string, error) {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))

	for _, t := range tags {
		for _, part := range strings.Split(t, ",") {
			tag := strings.ToLower(strings.TrimSpace(part))
			if tag == "" {
				continue
			}
			if len(tag) > 40 {
				return nil, fmt.Errorf("tag %q is longer than 40 characters", tag)
			}
			if strings.ContainsAny(tag, ",\"'") {
				return nil, fmt.Errorf("tag %q contains an unsupported character (commas and quotes are not allowed)", tag)
			}
			if _, dup := seen[tag]; dup {
				continue
			}
			seen[tag] = struct{}{}
			out = append(out, tag)
		}
	}

	if len(out) > 20 {
		return nil, fmt.Errorf("a contact can have at most 20 tags (got %d)", len(out))
	}
	return out, nil
}
