package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	DatabaseURL            string
	SupabaseURL            string
	SupabaseServiceRoleKey string

	UpstashRedisURL    string
	UpstashRedisToken  string
	UpstashVectorURL   string
	UpstashVectorToken string
	UpstashSearchURL   string
	UpstashSearchToken string

	B2AccountID      string
	B2ApplicationKey string
	B2BucketName     string
	B2Endpoint       string

	ATAPIKey   string
	ATUsername string
	ATSenderID string
	// ATDLRToken is the secret path segment of the delivery-report callback
	// (/api/v1/webhooks/sms/dlr/{token}). Without it reports are refused.
	ATDLRToken string
	// ATBaseURL replaces the Africa's Talking host. Local development only.
	ATBaseURL string

	GroqAPIKey string

	MetaWAToken              string
	MetaWAPhoneNumberID      string
	MetaWAWebhookVerifyToken string

	MpesaConsumerKey    string
	MpesaConsumerSecret string
	MpesaPasskey        string
	MpesaShortCode      string
	MpesaCallbackURL    string
	MpesaBaseURL        string
	// MpesaEnabled states whether M-Pesa is switched on (MPESA_ENABLED). It is
	// kept as the raw string so "unset" is distinguishable from an explicit
	// "false": when unset the setting is inferred from the credentials, but an
	// operator who is not taking payments yet can say so outright and boot
	// without a Daraja allowlist.
	MpesaEnabled string
	// MpesaAllowedIPs restricts the M-Pesa webhook to these IPs/CIDRs
	// (MPESA_ALLOWED_IPS, comma separated). Empty = allow all (dev only).
	MpesaAllowedIPs []string
	// SignupMode says who may register a school themselves (SIGNUP_MODE):
	// "open", "code" (needs SIGNUP_CODE) or "closed". Left unset it is closed
	// in production and open in development.
	SignupMode string
	SignupCode string
	// MpesaWebhookToken is the secret path segment of the addresses Safaricom
	// reports to (MPESA_WEBHOOK_TOKEN). Paybill confirmations are refused
	// without it.
	MpesaWebhookToken string

	// CORSAllowedOrigins adds extra origins to the default CORS allowlist
	// (CORS_ALLOWED_ORIGINS, comma separated). The defaults already include
	// the production frontend, localhost dev ports, and *.vercel.app previews.
	CORSAllowedOrigins []string

	JWTSecret string
	// SettingsEncryptionKey seals the per-school integration credentials
	// (M-Pesa keys, WhatsApp tokens, ...). Optional: when empty the API falls
	// back to JWTSecret, which couples rotating the session secret to
	// re-entering every school credential — so set a dedicated value.
	SettingsEncryptionKey string
	Port                  string
	AppEnv                string
}

// Load reads and validates all required environment variables.
// It fails fast with a descriptive error if any required var is missing.
func Load() (*Config, error) {
	return LoadFrom(os.Getenv)
}

// LoadFrom is Load with an injectable environment reader, so the production
// guards below can be exercised in tests without mutating the process
// environment (which is not safe to do in parallel).
func LoadFrom(getenv func(string) string) (*Config, error) {
	cfg := &Config{
		DatabaseURL:              getenv("DATABASE_URL"),
		SupabaseURL:              getenv("SUPABASE_URL"),
		SupabaseServiceRoleKey:   getenv("SUPABASE_SERVICE_ROLE_KEY"),
		UpstashRedisURL:          getenv("UPSTASH_REDIS_REST_URL"),
		UpstashRedisToken:        getenv("UPSTASH_REDIS_REST_TOKEN"),
		UpstashVectorURL:         getenv("UPSTASH_VECTOR_REST_URL"),
		UpstashVectorToken:       getenv("UPSTASH_VECTOR_REST_TOKEN"),
		UpstashSearchURL:         getenv("UPSTASH_SEARCH_REST_URL"),
		UpstashSearchToken:       getenv("UPSTASH_SEARCH_REST_TOKEN"),
		B2AccountID:              getenv("B2_ACCOUNT_ID"),
		B2ApplicationKey:         getenv("B2_APPLICATION_KEY"),
		B2BucketName:             getenv("B2_BUCKET_NAME"),
		B2Endpoint:               getenv("B2_ENDPOINT"),
		ATAPIKey:                 getenv("AT_API_KEY"),
		ATUsername:               getenv("AT_USERNAME"),
		ATSenderID:               getenv("AT_SENDER_ID"),
		ATDLRToken:               getenv("AT_DLR_TOKEN"),
		ATBaseURL:                getenv("AT_BASE_URL"),
		GroqAPIKey:               getenv("GROQ_API_KEY"),
		MetaWAToken:              getenv("META_WA_TOKEN"),
		MetaWAPhoneNumberID:      getenv("META_WA_PHONE_NUMBER_ID"),
		MetaWAWebhookVerifyToken: getenv("META_WA_WEBHOOK_VERIFY_TOKEN"),
		MpesaConsumerKey:         getenv("MPESA_CONSUMER_KEY"),
		MpesaConsumerSecret:      getenv("MPESA_CONSUMER_SECRET"),
		MpesaPasskey:             getenv("MPESA_PASSKEY"),
		MpesaShortCode:           getenv("MPESA_SHORT_CODE"),
		MpesaCallbackURL:         getenv("MPESA_CALLBACK_URL"),
		MpesaBaseURL:             getenv("MPESA_BASE_URL"),
		MpesaEnabled:             getenv("MPESA_ENABLED"),
		MpesaAllowedIPs:          splitCSV(getenv("MPESA_ALLOWED_IPS")),
		MpesaWebhookToken:        getenv("MPESA_WEBHOOK_TOKEN"),
		SignupMode:               strings.ToLower(strings.TrimSpace(getenv("SIGNUP_MODE"))),
		SignupCode:               getenv("SIGNUP_CODE"),
		CORSAllowedOrigins:       splitCSV(getenv("CORS_ALLOWED_ORIGINS")),
		JWTSecret:                getenv("JWT_SECRET"),
		SettingsEncryptionKey:    getenv("SETTINGS_ENCRYPTION_KEY"),
		Port:                     getenv("PORT"),
		AppEnv:                   getenv("APP_ENV"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.AppEnv == "" {
		cfg.AppEnv = "development"
	}

	required := map[string]string{
		"DATABASE_URL":              cfg.DatabaseURL,
		"SUPABASE_URL":              cfg.SupabaseURL,
		"SUPABASE_SERVICE_ROLE_KEY": cfg.SupabaseServiceRoleKey,
		// JWT_SECRET must be provided explicitly: an empty secret would let
		// anyone forge tokens signed with HMAC("").
		"JWT_SECRET": cfg.JWTSecret,
	}

	var missing []string
	for name, val := range required {
		if val == "" {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	// Validate port is numeric
	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return nil, fmt.Errorf("PORT must be a valid number: %w", err)
	}

	// Validate APP_ENV
	if cfg.AppEnv != "development" && cfg.AppEnv != "production" {
		return nil, fmt.Errorf("APP_ENV must be 'development' or 'production', got %q", cfg.AppEnv)
	}

	// Production safety: AT_BASE_URL redirects every school's SMS to another
	// host. It is a development switch and must never be set on a live server.
	if cfg.IsProduction() && cfg.ATBaseURL != "" {
		return nil, fmt.Errorf("AT_BASE_URL must not be set in production (it redirects all SMS away from Africa's Talking)")
	}

	switch cfg.SignupMode {
	case "":
		cfg.SignupMode = "open"
		if cfg.IsProduction() {
			cfg.SignupMode = "closed"
		}
	case "open", "closed":
	case "code":
		if len(strings.TrimSpace(cfg.SignupCode)) < 8 {
			return nil, fmt.Errorf("SIGNUP_MODE=code needs a SIGNUP_CODE of at least 8 characters")
		}
	default:
		return nil, fmt.Errorf("SIGNUP_MODE must be open, code or closed, got %q", cfg.SignupMode)
	}

	// Production safety: MPESA_BASE_URL decides where every school's payment
	// requests go. On a live server it may only be Safaricom.
	if cfg.IsProduction() && cfg.MpesaBaseURL != "" {
		switch strings.TrimRight(cfg.MpesaBaseURL, "/") {
		case "https://api.safaricom.co.ke", "https://sandbox.safaricom.co.ke":
		default:
			return nil, fmt.Errorf("MPESA_BASE_URL must be https://api.safaricom.co.ke or https://sandbox.safaricom.co.ke in production")
		}
	}

	// Production safety: never boot with M-Pesa switched on and an
	// unauthenticated, allowlist-free callback endpoint. /webhooks/mpesa/stk
	// takes an unauthenticated POST unless MPESA_WEBHOOK_TOKEN is set
	// (handler.webhookAllowed), so with M-Pesa live the allowlist is the only
	// thing standing between the internet and a forged payment confirmation.
	//
	// This only applies when M-Pesa is actually on. An operator still filling
	// in Daraja credentials has no reason to block the whole API — and SMS,
	// which does not depend on any of this, should not go down waiting for a
	// paybill that is not live yet.
	if cfg.IsProduction() && cfg.IsMpesaEnabled() && len(cfg.MpesaAllowedIPs) == 0 {
		return nil, fmt.Errorf("MPESA_ALLOWED_IPS must be set in production while M-Pesa is enabled (set MPESA_ENABLED=false to run without M-Pesa)")
	}

	return cfg, nil
}

// IsMpesaEnabled reports whether M-Pesa should be treated as switched on.
//
// MPESA_ENABLED decides outright when it is set. When it is unset the setting
// is inferred from the credentials, as before — except that the example values
// shipped in .env.example no longer count as configured. They used to: an
// unfilled MPESA_PASSKEY placeholder is a non-empty string, so a service that
// had never taken a payment looked configured and refused to boot.
func (c *Config) IsMpesaEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(c.MpesaEnabled)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	}
	return c.MpesaConsumerKey != "" && c.MpesaPasskey != "" &&
		!looksPlaceholder(c.MpesaConsumerKey) && !looksPlaceholder(c.MpesaPasskey)
}

// looksPlaceholder reports whether v is one of the markers .env.example uses
// for a value the operator still has to fill in.
//
// Only unmistakable markers are matched. Guessing wrong in the permissive
// direction would treat a real Daraja credential as absent and quietly disable
// the allowlist guard, so ambiguous markers are deliberately excluded: a real
// consumer key is a base64url blob that could legitimately contain "xxx".
func looksPlaceholder(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	for _, marker := range []string{
		"your_", "your-", "changeme", "change-me", "change_me",
		"placeholder", "replaceme", "replace-me", "replace_me",
		"todo", "<", ">",
	} {
		if strings.Contains(v, marker) {
			return true
		}
	}
	return false
}

// IsProduction returns true when running in production mode.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// IsDevelopment returns true when running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// splitCSV splits a comma-separated environment value into trimmed,
// non-empty entries.
func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
