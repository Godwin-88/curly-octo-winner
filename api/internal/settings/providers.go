package settings

// Provider connectivity tests.
//
// The Settings screen lets a principal verify their own credentials without
// waiting for a failed payment or a silently undelivered SMS. Each test
// performs the smallest real call that proves the credentials work, against the
// provider's own API (see providers_more.go for the remaining probes).
//
// A test never returns a credential to the browser: only a status and a short,
// human-readable message.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// testTimeout keeps a slow provider from holding the HTTP handler open.
const testTimeout = 12 * time.Second

// TestResult is the outcome of a connectivity test.
type TestResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// PlatformCredentials supplies the environment-level credentials used when a
// school inherits the platform defaults (use_platform_default = true).
type PlatformCredentials struct {
	MpesaConsumerKey    string
	MpesaConsumerSecret string
	MpesaBaseURL        string
	ATAPIKey            string
	ATUsername          string
	ATBaseURL           string
	MetaWAToken         string
	MetaWAGraphURL      string
	B2AccountID         string
	B2ApplicationKey    string
	B2Endpoint          string
	GroqAPIKey          string
	GroqBaseURL         string
	UpstashRedisURL     string
	UpstashRedisToken   string
}

// TestProvider verifies a provider's credentials. The school's own values are
// used when configured; otherwise the platform defaults apply.
func (s *Service) TestProvider(ctx context.Context, tenantID uuid.UUID, provider string, platform PlatformCredentials) (*TestResult, error) {
	if !IsProvider(provider) {
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}

	config, secrets, useDefault, err := s.ResolveSecrets(ctx, tenantID, provider)
	if err != nil {
		return nil, err
	}
	if config == nil {
		config = map[string]any{}
	}
	if useDefault {
		secrets = nil // fall back to the environment values below
	}

	switch provider {
	case "mpesa":
		return testMpesa(ctx, secrets, config, platform)
	case "africastalking":
		return testAfricaTalking(ctx, secrets, config, platform)
	case "whatsapp":
		return testWhatsApp(ctx, secrets, config, platform)
	case "backblaze":
		return testBackblaze(ctx, secrets, config, platform)
	case "groq":
		return testGroq(ctx, secrets, config, platform)
	case "upstash":
		return testUpstash(ctx, secrets, config, platform)
	}
	return &TestResult{Message: "no test implemented for this provider"}, nil
}

// pick resolves a credential: the school's secret wins, then its non-secret
// config value, then the platform default.
//
// Placeholder values (the `your_…`/`changeme` text that ships in .env.example)
// count as "not configured": a principal must never see a test failure caused
// by a placeholder that was never filled in.
func pick(secrets map[string]string, config map[string]any, platform, secretKey, configKey string) string {
	if v := strings.TrimSpace(secrets[secretKey]); v != "" && !isPlaceholder(v) {
		return v
	}
	if v, ok := config[configKey].(string); ok && strings.TrimSpace(v) != "" && !isPlaceholder(v) {
		return strings.TrimSpace(v)
	}
	if isPlaceholder(platform) {
		return ""
	}
	return platform
}

func isPlaceholder(v string) bool {
	lower := strings.ToLower(strings.TrimSpace(v))
	if lower == "" {
		return true
	}
	for _, marker := range []string{"your_", "your-", "changeme", "change-me", "placeholder", "xxx", "<", "todo", "replace"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// summarise trims a provider's error body so the UI shows a sentence, not a
// page of HTML.
func summarise(body string) string {
	if body == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(body), "<") {
		return "the provider returned an HTML error page"
	}
	body = strings.Join(strings.Fields(body), " ")
	if len(body) > 160 {
		body = body[:160] + "…"
	}
	return body
}

func doRequest(ctx context.Context, req *http.Request) (int, string, error) {
	client := &http.Client{Timeout: testTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	return resp.StatusCode, strings.TrimSpace(string(body)), nil
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

func trimScheme(base string) string {
	if !strings.HasPrefix(base, "http") {
		base = "https://" + base
	}
	return strings.TrimSuffix(base, "/")
}

// testMpesa requests a Daraja OAuth token: it exercises the consumer key and
// secret exactly like an STK push would.
func testMpesa(ctx context.Context, secrets map[string]string, config map[string]any, platform PlatformCredentials) (*TestResult, error) {
	key := pick(secrets, config, platform.MpesaConsumerKey, "consumer_key", "consumer_key")
	secret := pick(secrets, config, platform.MpesaConsumerSecret, "consumer_secret", "consumer_secret")
	if key == "" || secret == "" {
		return &TestResult{Message: "Add the Daraja consumer key and secret first."}, nil
	}
	base := pick(secrets, config, platform.MpesaBaseURL, "base_url", "base_url")
	if base == "" {
		base = "https://sandbox.safaricomm.co.ke"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		trimScheme(base)+"/oauth/v1/generate?grant_type=client_credentials", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(key, secret)

	status, body, err := doRequest(ctx, req)
	if err != nil {
		return &TestResult{Message: "Could not reach Safaricom Daraja: " + err.Error()}, nil
	}
	if status == http.StatusOK {
		return &TestResult{OK: true, Message: "Daraja accepted the credentials."}, nil
	}
	return &TestResult{Message: fmt.Sprintf("Daraja rejected the credentials (HTTP %d). %s", status, summarise(body))}, nil
}

// testAfricaTalking validates the API key and username.
func testAfricaTalking(ctx context.Context, secrets map[string]string, config map[string]any, platform PlatformCredentials) (*TestResult, error) {
	apiKey := pick(secrets, config, platform.ATAPIKey, "api_key", "api_key")
	username := pick(secrets, config, platform.ATUsername, "username", "username")
	if apiKey == "" || username == "" {
		return &TestResult{Message: "Add the Africa's Talking API key and username first."}, nil
	}
	base := pick(secrets, config, platform.ATBaseURL, "base_url", "base_url")
	if base == "" {
		base = "https://api.africastalking.com"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/version/1/account/user?username=%s", trimScheme(base), url.QueryEscape(username)), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apiKey", apiKey)

	status, body, err := doRequest(ctx, req)
	if err != nil {
		return &TestResult{Message: "Could not reach Africa's Talking: " + err.Error()}, nil
	}
	if status == http.StatusOK {
		return &TestResult{OK: true, Message: "Africa's Talking accepted the credentials."}, nil
	}
	return &TestResult{Message: fmt.Sprintf("Africa's Talking rejected the credentials (HTTP %d). %s", status, summarise(body))}, nil
}

// testWhatsApp validates the Cloud API token against the Graph API.
func testWhatsApp(ctx context.Context, secrets map[string]string, config map[string]any, platform PlatformCredentials) (*TestResult, error) {
	token := pick(secrets, config, platform.MetaWAToken, "access_token", "access_token")
	if token == "" {
		return &TestResult{Message: "Add the WhatsApp Cloud API access token first."}, nil
	}
	base := pick(secrets, config, platform.MetaWAGraphURL, "graph_url", "graph_url")
	if base == "" {
		base = "https://graph.facebook.com/v21.0"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimScheme(base)+"/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	status, body, err := doRequest(ctx, req)
	if err != nil {
		return &TestResult{Message: "Could not reach the WhatsApp Graph API: " + err.Error()}, nil
	}
	if status == http.StatusOK {
		var payload struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal([]byte(body), &payload)
		msg := "WhatsApp Cloud API accepted the token."
		if payload.Name != "" {
			msg = "Connected to WhatsApp business account: " + payload.Name
		}
		return &TestResult{OK: true, Message: msg}, nil
	}
	return &TestResult{Message: fmt.Sprintf("WhatsApp rejected the token (HTTP %d). %s", status, summarise(body))}, nil
}

// testBackblaze authorizes against B2, validating both key id and key.
func testBackblaze(ctx context.Context, secrets map[string]string, config map[string]any, platform PlatformCredentials) (*TestResult, error) {
	accountID := pick(secrets, config, platform.B2AccountID, "account_id", "account_id")
	appKey := pick(secrets, config, platform.B2ApplicationKey, "application_key", "application_key")
	if accountID == "" || appKey == "" {
		return &TestResult{Message: "Add the Backblaze B2 key ID and application key first."}, nil
	}
	base := pick(secrets, config, platform.B2Endpoint, "endpoint", "endpoint")
	if base == "" {
		base = "https://api.backblazeb2.com"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		trimScheme(base)+"/b2api/v2/b2_authorize_account", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Basic "+basicAuth(accountID, appKey))
	req.Header.Set("Content-Length", "0")

	status, body, err := doRequest(ctx, req)
	if err != nil {
		return &TestResult{Message: "Could not reach Backblaze B2: " + err.Error()}, nil
	}
	if status == http.StatusOK {
		return &TestResult{OK: true, Message: "Backblaze B2 accepted the credentials."}, nil
	}
	return &TestResult{Message: fmt.Sprintf("Backblaze rejected the credentials (HTTP %d). %s", status, summarise(body))}, nil
}

// testGroq lists models, validating the API key without spending tokens.
func testGroq(ctx context.Context, secrets map[string]string, config map[string]any, platform PlatformCredentials) (*TestResult, error) {
	key := pick(secrets, config, platform.GroqAPIKey, "api_key", "api_key")
	if key == "" {
		return &TestResult{Message: "Add the Groq API key first."}, nil
	}
	base := pick(secrets, config, platform.GroqBaseURL, "base_url", "base_url")
	if base == "" {
		base = "https://api.groq.com/openai/v1"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimScheme(base)+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)

	status, body, err := doRequest(ctx, req)
	if err != nil {
		return &TestResult{Message: "Could not reach Groq: " + err.Error()}, nil
	}
	if status == http.StatusOK {
		return &TestResult{OK: true, Message: "Groq accepted the API key."}, nil
	}
	return &TestResult{Message: fmt.Sprintf("Groq rejected the API key (HTTP %d). %s", status, summarise(body))}, nil
}

// testUpstash pings the Redis REST endpoint — the same call the login rate
// limiter makes on every sign-in.
func testUpstash(ctx context.Context, secrets map[string]string, config map[string]any, platform PlatformCredentials) (*TestResult, error) {
	base := pick(secrets, config, platform.UpstashRedisURL, "redis_url", "redis_url")
	token := pick(secrets, config, platform.UpstashRedisToken, "redis_token", "redis_token")
	if base == "" || token == "" {
		return &TestResult{Message: "Add the Upstash Redis REST URL and token first."}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimScheme(base)+"/ping", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	status, body, err := doRequest(ctx, req)
	if err != nil {
		return &TestResult{Message: "Could not reach Upstash: " + err.Error()}, nil
	}
	if status == http.StatusOK && strings.Contains(body, "PONG") {
		return &TestResult{OK: true, Message: "Upstash Redis responded to PING."}, nil
	}
	return &TestResult{Message: fmt.Sprintf("Upstash rejected the credentials (HTTP %d). %s", status, summarise(body))}, nil
}
