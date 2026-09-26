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
	// MpesaAllowedIPs restricts the M-Pesa webhook to these IPs/CIDRs
	// (MPESA_ALLOWED_IPS, comma separated). Empty = allow all (dev only).
	MpesaAllowedIPs []string

	// CORSAllowedOrigins adds extra origins to the default CORS allowlist
	// (CORS_ALLOWED_ORIGINS, comma separated). The defaults already include
	// the production frontend, localhost dev ports, and *.vercel.app previews.
	CORSAllowedOrigins []string

	JWTSecret string
	Port      string
	AppEnv    string
}

// Load reads and validates all required environment variables.
// It fails fast with a descriptive error if any required var is missing.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:              os.Getenv("DATABASE_URL"),
		SupabaseURL:              os.Getenv("SUPABASE_URL"),
		SupabaseServiceRoleKey:   os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		UpstashRedisURL:          os.Getenv("UPSTASH_REDIS_REST_URL"),
		UpstashRedisToken:        os.Getenv("UPSTASH_REDIS_REST_TOKEN"),
		UpstashVectorURL:         os.Getenv("UPSTASH_VECTOR_REST_URL"),
		UpstashVectorToken:       os.Getenv("UPSTASH_VECTOR_REST_TOKEN"),
		UpstashSearchURL:         os.Getenv("UPSTASH_SEARCH_REST_URL"),
		UpstashSearchToken:       os.Getenv("UPSTASH_SEARCH_REST_TOKEN"),
		B2AccountID:              os.Getenv("B2_ACCOUNT_ID"),
		B2ApplicationKey:         os.Getenv("B2_APPLICATION_KEY"),
		B2BucketName:             os.Getenv("B2_BUCKET_NAME"),
		B2Endpoint:               os.Getenv("B2_ENDPOINT"),
		ATAPIKey:                 os.Getenv("AT_API_KEY"),
		ATUsername:               os.Getenv("AT_USERNAME"),
		ATSenderID:               os.Getenv("AT_SENDER_ID"),
		GroqAPIKey:               os.Getenv("GROQ_API_KEY"),
		MetaWAToken:              os.Getenv("META_WA_TOKEN"),
		MetaWAPhoneNumberID:      os.Getenv("META_WA_PHONE_NUMBER_ID"),
		MetaWAWebhookVerifyToken: os.Getenv("META_WA_WEBHOOK_VERIFY_TOKEN"),
		MpesaConsumerKey:         os.Getenv("MPESA_CONSUMER_KEY"),
		MpesaConsumerSecret:      os.Getenv("MPESA_CONSUMER_SECRET"),
		MpesaPasskey:             os.Getenv("MPESA_PASSKEY"),
		MpesaShortCode:           os.Getenv("MPESA_SHORT_CODE"),
		MpesaCallbackURL:         os.Getenv("MPESA_CALLBACK_URL"),
		MpesaBaseURL:             os.Getenv("MPESA_BASE_URL"),
		MpesaAllowedIPs:          splitCSV(os.Getenv("MPESA_ALLOWED_IPS")),
		CORSAllowedOrigins:       splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS")),
		JWTSecret:                os.Getenv("JWT_SECRET"),
		Port:                     os.Getenv("PORT"),
		AppEnv:                   os.Getenv("APP_ENV"),
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

	return cfg, nil
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
