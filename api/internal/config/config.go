package config

import (
	"fmt"
	"os"
	"strconv"
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

	MetaWAToken              string
	MetaWAPhoneNumberID      string
	MetaWAWebhookVerifyToken string

	MpesaConsumerKey    string
	MpesaConsumerSecret string
	MpesaPasskey        string
	MpesaShortCode      string
	MpesaCallbackURL    string
	MpesaBaseURL        string

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
		MetaWAToken:              os.Getenv("META_WA_TOKEN"),
		MetaWAPhoneNumberID:      os.Getenv("META_WA_PHONE_NUMBER_ID"),
		MetaWAWebhookVerifyToken: os.Getenv("META_WA_WEBHOOK_VERIFY_TOKEN"),
		MpesaConsumerKey:         os.Getenv("MPESA_CONSUMER_KEY"),
		MpesaConsumerSecret:      os.Getenv("MPESA_CONSUMER_SECRET"),
		MpesaPasskey:             os.Getenv("MPESA_PASSKEY"),
		MpesaShortCode:           os.Getenv("MPESA_SHORT_CODE"),
		MpesaCallbackURL:         os.Getenv("MPESA_CALLBACK_URL"),
		MpesaBaseURL:             os.Getenv("MPESA_BASE_URL"),
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
		"DATABASE_URL":                 cfg.DatabaseURL,
		"SUPABASE_URL":                 cfg.SupabaseURL,
		"SUPABASE_SERVICE_ROLE_KEY":    cfg.SupabaseServiceRoleKey,
		"UPSTASH_REDIS_REST_URL":       cfg.UpstashRedisURL,
		"UPSTASH_REDIS_REST_TOKEN":     cfg.UpstashRedisToken,
		"UPSTASH_VECTOR_REST_URL":      cfg.UpstashVectorURL,
		"UPSTASH_VECTOR_REST_TOKEN":    cfg.UpstashVectorToken,
		"UPSTASH_SEARCH_REST_URL":      cfg.UpstashSearchURL,
		"UPSTASH_SEARCH_REST_TOKEN":    cfg.UpstashSearchToken,
		"B2_ACCOUNT_ID":                cfg.B2AccountID,
		"B2_APPLICATION_KEY":           cfg.B2ApplicationKey,
		"B2_BUCKET_NAME":               cfg.B2BucketName,
		"B2_ENDPOINT":                  cfg.B2Endpoint,
		"AT_API_KEY":                   cfg.ATAPIKey,
		"AT_USERNAME":                  cfg.ATUsername,
		"AT_SENDER_ID":                 cfg.ATSenderID,
		"META_WA_TOKEN":                cfg.MetaWAToken,
		"META_WA_PHONE_NUMBER_ID":      cfg.MetaWAPhoneNumberID,
		"META_WA_WEBHOOK_VERIFY_TOKEN": cfg.MetaWAWebhookVerifyToken,
		"JWT_SECRET":                   cfg.JWTSecret,
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
