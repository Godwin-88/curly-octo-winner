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

// VectorClient is a REST client for Upstash Vector.
// Used for semantic search over message templates and FAQ knowledge base.
type VectorClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// VectorPoint represents a single vector with metadata.
type VectorPoint struct {
	ID       string         `json:"id"`
	Vector   []float32      `json:"vector"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// VectorQueryResult is a single match from a vector query.
type VectorQueryResult struct {
	ID       string         `json:"id"`
	Score    float64        `json:"score"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewVectorClient creates a new Upstash Vector REST client.
func NewVectorClient(baseURL, token string) *VectorClient {
	return &VectorClient{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Upsert inserts or updates vectors in the index.
func (c *VectorClient) Upsert(ctx context.Context, namespace string, points []VectorPoint) error {
	payload, err := json.Marshal(map[string]any{
		"namespace": namespace,
		"vectors":   points,
	})
	if err != nil {
		return fmt.Errorf("marshal vector upsert: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/upsert", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create vector upsert request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute vector upsert: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read vector upsert response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vector upsert error (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// Query performs a vector similarity search and returns top-k matches.
func (c *VectorClient) Query(ctx context.Context, namespace string, vector []float32, topK int) ([]VectorQueryResult, error) {
	payload, err := json.Marshal(map[string]any{
		"namespace":       namespace,
		"vector":          vector,
		"topK":            topK,
		"includeMetadata": true,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal vector query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/query", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create vector query request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute vector query: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read vector query response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vector query error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []VectorQueryResult `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal vector query response: %w", err)
	}
	return result.Result, nil
}

// Delete removes vectors by ID.
func (c *VectorClient) Delete(ctx context.Context, namespace string, ids []string) error {
	payload, err := json.Marshal(map[string]any{
		"namespace": namespace,
		"ids":       ids,
	})
	if err != nil {
		return fmt.Errorf("marshal vector delete: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/delete", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create vector delete request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("execute vector delete: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read vector delete response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vector delete error (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}
