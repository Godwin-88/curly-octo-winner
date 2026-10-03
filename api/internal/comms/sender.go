package comms

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shule360/api/internal/comms/sms"
)

// SecretResolver is settings.Service.ResolveSecrets: a school's stored
// credentials for a provider, or usePlatformDefault when it inherits ours.
type SecretResolver interface {
	ResolveSecrets(ctx context.Context, tenantID uuid.UUID, provider string) (config map[string]any, secrets map[string]string, usePlatformDefault bool, err error)
}

// PlatformSMS is the platform's own Africa's Talking account (from the
// environment), used by schools that have not configured their own.
type PlatformSMS struct {
	APIKey   string
	Username string
	SenderID string
	// BaseURL overrides the provider host (AT_BASE_URL). Local development
	// only: it applies to every school, including ones with their own account.
	BaseURL string
}

// NewSenderFactory builds the per-school sender lookup.
//
// Credentials: the school's own (Settings → Integrations → Africa's Talking)
// when it has saved an API key and username; the platform's otherwise.
// Sender id: the school's integration setting, then tenants.at_sender_id, then
// the platform's. An empty sender id is allowed and sends from the provider's
// default short code.
func NewSenderFactory(pool *pgxpool.Pool, resolver SecretResolver, platform PlatformSMS) SenderFactory {
	return func(ctx context.Context, tenantID uuid.UUID) (Sender, error) {
		var tenantSender *string
		if err := pool.QueryRow(ctx, `SELECT at_sender_id FROM tenants WHERE id = $1`, tenantID).Scan(&tenantSender); err != nil {
			return nil, err
		}

		apiKey, username := platform.APIKey, platform.Username
		senderID := firstNonEmpty(deref(tenantSender), platform.SenderID)

		if resolver != nil {
			config, secrets, usePlatform, err := resolver.ResolveSecrets(ctx, tenantID, "africastalking")
			if err != nil {
				return nil, err
			}
			if !usePlatform {
				ownKey := strings.TrimSpace(secrets["api_key"])
				ownUser := configString(config, secrets, "username")
				if ownKey != "" && ownUser != "" {
					apiKey, username = ownKey, ownUser
				}
				if own := configString(config, secrets, "sender_id"); own != "" {
					senderID = own
				}
			}
		}

		if !usable(apiKey) || !usable(username) {
			return nil, &NotConfiguredError{Reason: "SMS is not set up for this school: add the Africa's Talking API key and username under Settings → Integrations."}
		}
		// isProduction is true: NewATClient still routes the "sandbox" username
		// to the sandbox host.
		return sms.NewATClient(apiKey, username, senderID, true).WithBaseURL(platform.BaseURL), nil
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

func configString(config map[string]any, secrets map[string]string, key string) string {
	if v, ok := config[key].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(secrets[key])
}

// usable rejects blanks and the placeholder text that ships in .env.example
// ("your_at_api_key_here"), which would otherwise be sent to the provider as
// if it were a credential.
func usable(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return false
	}
	for _, marker := range []string{"your_", "your-", "changeme", "change-me", "placeholder", "<"} {
		if strings.Contains(v, marker) {
			return false
		}
	}
	return true
}
