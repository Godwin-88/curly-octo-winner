package upstash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SearchClient is a REST client for Upstash Search.
// Used for full-text search over learner names, supplier names, transaction refs.
type SearchClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// SearchDocument represents a document indexed in Upstash Search.
type SearchDocument struct {
	ID       string         `json:"id"`
	Fields   map[string]any `json:"fields"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// SearchResult is a single match from a search query.
type SearchResult struct {
	ID       string         `json:"id"`
	Score    float64        `json:"score"`
	Fields   map[string]any `json:"fields"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewSearchClient creates a new Upstash Search REST client.
func NewSearchClient(baseURL, token string) *SearchClient {
	return &SearchClient{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Index adds or updates documents in the search index.
func (c *SearchClient) Index(ctx context.Context, namespace string, docs []SearchDocument) error {
	payload, err := json.Marshal(map[string]any{
		"namespace": namespace,
		"documents": docs,
	})
	if err != nil {
		return fmt.Errorf("marshal search index: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/index", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create search index request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute search index: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read search index response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("search index error (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// Query performs a full-text search and returns top-k matches.
func (c *SearchClient) Query(ctx context.Context, namespace, query string, topK int) ([]SearchResult, error) {
	payload, err := json.Marshal(map[string]any{
		"namespace": namespace,
		"query":     query,
		"topK":      topK,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal search query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/query", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create search query request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute search query: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read search query response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search query error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []SearchResult `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal search query response: %w", err)
	}
	return result.Result, nil
}

// Delete removes documents by ID.
func (c *SearchClient) Delete(ctx context.Context, namespace string, ids []string) error {
	payload, err := json.Marshal(map[string]any{
		"namespace": namespace,
		"ids":       ids,
	})
	if err != nil {
		return fmt.Errorf("marshal search delete: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/delete", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create search delete request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute search delete: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read search delete response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("search delete error (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}
