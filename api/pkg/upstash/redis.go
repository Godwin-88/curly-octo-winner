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

// RedisClient is a minimal REST client for Upstash Redis.
// Upstash Redis exposes a REST API, so no TCP connection is needed.
type RedisClient struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewRedisClient creates a new Upstash Redis REST client.
func NewRedisClient(baseURL, token string) *RedisClient {
	return &RedisClient{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Do executes a raw Redis command and returns the decoded reply payload.
//
// Upstash's REST API expects the command itself as a bare JSON array
// (["SET","k","v"]); the {"command":[...]} envelope is rejected with HTTP 400
// "expected JSON array". Replies arrive wrapped as {"result": ...} (or
// {"error": ...}), so Do unwraps the envelope and returns the raw Redis reply —
// bare string, number, array or null — which is what the typed helpers below
// parse.
func (c *RedisClient) Do(ctx context.Context, command ...string) ([]byte, error) {
	if c.baseURL == "" || c.token == "" {
		return nil, fmt.Errorf("upstash redis is not configured (set UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN)")
	}

	payload, err := json.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("marshal redis command: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create redis request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute redis command: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read redis response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("redis error (status %d): %s", resp.StatusCode, string(body))
	}

	// Unwrap {"result": ...}; fall back to the raw body for endpoints that
	// answer with a bare value.
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err == nil {
		if errRaw, ok := envelope["error"]; ok {
			var msg string
			if json.Unmarshal(errRaw, &msg) != nil {
				msg = string(errRaw)
			}
			return nil, fmt.Errorf("redis error: %s", msg)
		}
		if result, ok := envelope["result"]; ok {
			return result, nil
		}
	}
	return body, nil
}

// Set stores a string value with an optional TTL in seconds.
func (c *RedisClient) Set(ctx context.Context, key, value string, ttlSeconds int) error {
	args := []string{"SET", key, value}
	if ttlSeconds > 0 {
		args = append(args, "EX", fmt.Sprintf("%d", ttlSeconds))
	}
	_, err := c.Do(ctx, args...)
	return err
}

// Get retrieves a string value. Returns ("", nil) if key doesn't exist.
func (c *RedisClient) Get(ctx context.Context, key string) (string, error) {
	body, err := c.Do(ctx, "GET", key)
	if err != nil {
		return "", err
	}
	// Upstash returns the value as a JSON string, e.g. "value" or null
	var val *string
	if err := json.Unmarshal(body, &val); err != nil {
		return "", fmt.Errorf("unmarshal redis GET response: %w", err)
	}
	if val == nil {
		return "", nil
	}
	return *val, nil
}

// Del removes one or more keys.
func (c *RedisClient) Del(ctx context.Context, keys ...string) error {
	args := append([]string{"DEL"}, keys...)
	_, err := c.Do(ctx, args...)
	return err
}

// Incr increments a key and returns the new value.
func (c *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	body, err := c.Do(ctx, "INCR", key)
	if err != nil {
		return 0, err
	}
	var val int64
	if err := json.Unmarshal(body, &val); err != nil {
		return 0, fmt.Errorf("unmarshal redis INCR response: %w", err)
	}
	return val, nil
}

// Expire sets a TTL on a key in seconds.
func (c *RedisClient) Expire(ctx context.Context, key string, ttlSeconds int) error {
	_, err := c.Do(ctx, "EXPIRE", key, fmt.Sprintf("%d", ttlSeconds))
	return err
}

// LPush pushes values onto the head of a list (FIFO queue).
func (c *RedisClient) LPush(ctx context.Context, key string, values ...string) error {
	args := append([]string{"LPUSH", key}, values...)
	_, err := c.Do(ctx, args...)
	return err
}

// BRPop pops a value from the tail of a list, blocking up to timeoutSeconds.
// Returns ("", nil) if timeout is reached.
func (c *RedisClient) BRPop(ctx context.Context, key string, timeoutSeconds int) (string, error) {
	body, err := c.Do(ctx, "BRPOP", key, fmt.Sprintf("%d", timeoutSeconds))
	if err != nil {
		return "", err
	}
	// Response is [key, value] or null on timeout
	var result []string
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("unmarshal redis BRPOP response: %w", err)
	}
	if len(result) < 2 {
		return "", nil
	}
	return result[1], nil
}

// LLen returns the length of a list.
func (c *RedisClient) LLen(ctx context.Context, key string) (int64, error) {
	body, err := c.Do(ctx, "LLEN", key)
	if err != nil {
		return 0, err
	}
	var val int64
	if err := json.Unmarshal(body, &val); err != nil {
		return 0, fmt.Errorf("unmarshal redis LLEN response: %w", err)
	}
	return val, nil
}

// TokenBucket implements a token-bucket rate limiter using Redis.
// Returns (allowed bool, remaining int, err error).
func (c *RedisClient) TokenBucket(ctx context.Context, key string, capacity, refillPerSecond int) (bool, int, error) {
	// Use a Lua script for atomicity
	script := `
local current = tonumber(redis.call('GET', KEYS[1]) or '0')
local last_refill = tonumber(redis.call('GET', KEYS[1]..':ts') or '0')
local now = tonumber(ARGV[1])
local capacity = tonumber(ARGV[2])
local refill = tonumber(ARGV[3])

if current == 0 or last_refill == 0 then
    current = capacity
    last_refill = now
end

local elapsed = now - last_refill
current = math.min(capacity, current + (elapsed * refill))
redis.call('SET', KEYS[1], current)
redis.call('SET', KEYS[1]..':ts', now)

if current >= 1 then
    redis.call('DECR', KEYS[1])
    return {1, current - 1}
else
    return {0, 0}
end
`
	// Upstash REST API supports EVAL with the script as first arg
	body, err := c.Do(ctx, "EVAL", script, "1", key, fmt.Sprintf("%d", time.Now().Unix()), fmt.Sprintf("%d", capacity), fmt.Sprintf("%d", refillPerSecond))
	if err != nil {
		return false, 0, err
	}

	var result []int64
	if err := json.Unmarshal(body, &result); err != nil {
		return false, 0, fmt.Errorf("unmarshal token bucket response: %w", err)
	}
	if len(result) < 2 {
		return false, 0, fmt.Errorf("unexpected token bucket response: %s", string(body))
	}
	return result[0] == 1, int(result[1]), nil
}

// ParseIntReply parses an Upstash REST reply that is expected to be an integer
// (e.g. INCR, EXPIRE). Upstash returns bare JSON numbers or one-element arrays
// depending on the command, so both shapes are accepted.
func ParseIntReply(body []byte) (int64, error) {
	var n int64
	if err := json.Unmarshal(body, &n); err == nil {
		return n, nil
	}
	var arr []int64
	if err := json.Unmarshal(body, &arr); err == nil && len(arr) > 0 {
		return arr[0], nil
	}
	return 0, fmt.Errorf("unexpected integer reply: %s", string(body))
}
