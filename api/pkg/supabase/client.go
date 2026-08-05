package supabase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Client wraps the Supabase Postgres pool and Auth admin API.
type Client struct {
	Pool *pgxpool.Pool

	supabaseURL string
	serviceKey  string
	http        *http.Client
}

// NewClient creates a new Supabase client with a pgx connection pool.
func NewClient(ctx context.Context, databaseURL, supabaseURL, serviceKey string) (*Client, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Client{
		Pool:        pool,
		supabaseURL: supabaseURL,
		serviceKey:  serviceKey,
		http:        &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// Close closes the underlying connection pool.
func (c *Client) Close() {
	c.Pool.Close()
}

// CreateUser creates a user in Supabase Auth with a temporary password.
// Returns the new user's UUID.
func (c *Client) CreateUser(ctx context.Context, email, password string) (string, error) {
	payload, err := json.Marshal(map[string]any{
		"email":         email,
		"password":      password,
		"email_confirm": true,
	})
	if err != nil {
		return "", fmt.Errorf("marshal create user: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.supabaseURL+"/auth/v1/admin/users", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create auth request: %w", err)
	}
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute create user: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read create user response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create user error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal create user response: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("create user response missing id")
	}
	return result.ID, nil
}

// DeleteUser removes a user from Supabase Auth.
func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.supabaseURL+"/auth/v1/admin/users/"+userID, nil)
	if err != nil {
		return fmt.Errorf("create delete user request: %w", err)
	}
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute delete user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete user error (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// SetTenantContext sets the app.tenant_id for RLS policies on the current connection.
// This must be called within a transaction before any RLS-protected queries.
func (c *Client) SetTenantContext(ctx context.Context, tenantID string) error {
	_, err := c.Pool.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID)
	if err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}
	return nil
}
