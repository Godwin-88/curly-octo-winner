package upstash

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Upstash's REST API takes the *bare* command array as the request body and
// answers with {"result": ...}. Getting either side wrong breaks every
// Redis-backed feature silently: the Logout/login rate limiter answers 503
// (fail-closed), and sessions never reach Redis. These tests pin the wire
// format against a stub server.
func TestRedisClientCommandWireFormat(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-token")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		_, _ = w.Write([]byte(`{"result":1}`))
	}))
	defer srv.Close()

	c := NewRedisClient(srv.URL, "test-token")
	got, err := c.Incr(context.Background(), "loginrl:ip:127.0.0.1:1")
	if err != nil {
		t.Fatalf("Incr: %v", err)
	}
	if got != 1 {
		t.Errorf("Incr = %d, want 1 (envelope not unwrapped)", got)
	}

	var sent []string
	if err := json.Unmarshal([]byte(gotBody), &sent); err != nil {
		t.Fatalf("request body %q is not a bare JSON array: %v", gotBody, err)
	}
	if len(sent) != 2 || sent[0] != "INCR" || sent[1] != "loginrl:ip:127.0.0.1:1" {
		t.Errorf("command sent = %#v, want [INCR loginrl:ip:127.0.0.1:1]", sent)
	}
}

func TestRedisClientReplyShapes(t *testing.T) {
	tests := []struct {
		name    string
		reply   string
		call    func(*RedisClient) (any, error)
		want    any
		wantErr bool
	}{
		{
			name:  "GET string value",
			reply: `{"result":"cached"}`,
			call:  func(c *RedisClient) (any, error) { return c.Get(context.Background(), "k") },
			want:  "cached",
		},
		{
			name:  "GET missing key returns empty string",
			reply: `{"result":null}`,
			call:  func(c *RedisClient) (any, error) { return c.Get(context.Background(), "k") },
			want:  "",
		},
		{
			name:  "INCR integer",
			reply: `{"result":42}`,
			call:  func(c *RedisClient) (any, error) { return c.Incr(context.Background(), "k") },
			want:  int64(42),
		},
		{
			name:  "LLEN integer",
			reply: `{"result":7}`,
			call:  func(c *RedisClient) (any, error) { return c.LLen(context.Background(), "q") },
			want:  int64(7),
		},
		{
			name:  "BRPOP list",
			reply: `{"result":["queue","payload"]}`,
			call:  func(c *RedisClient) (any, error) { return c.BRPop(context.Background(), "queue", 1) },
			want:  "payload",
		},
		{
			name:  "BRPOP timeout returns empty",
			reply: `{"result":null}`,
			call:  func(c *RedisClient) (any, error) { return c.BRPop(context.Background(), "queue", 1) },
			want:  "",
		},
		{
			name:  "EVAL token bucket list",
			reply: `{"result":[1,4]}`,
			call: func(c *RedisClient) (any, error) {
				allowed, remaining, err := c.TokenBucket(context.Background(), "rl", 5, 1)
				if !allowed {
					return nil, err
				}
				return int64(remaining), err
			},
			want: int64(4),
		},
		{
			name:    "error field on HTTP 200",
			reply:   `{"error":"ERR wrong number of arguments"}`,
			call:    func(c *RedisClient) (any, error) { return c.Get(context.Background(), "k") },
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tc.reply))
			}))
			defer srv.Close()

			got, err := tc.call(NewRedisClient(srv.URL, "tok"))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error for reply %s, got %#v", tc.reply, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

// A non-200 response (Upstash answers 400 "expected JSON array" for a malformed
// request) must surface as an error, never as a silent success.
func TestRedisClientNonOKStatusIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"expected JSON array"}`))
	}))
	defer srv.Close()

	if _, err := NewRedisClient(srv.URL, "tok").Do(context.Background(), "PING"); err == nil {
		t.Fatal("expected an error for HTTP 400")
	}
}

// An unconfigured client must fail loudly with a message that names the missing
// environment variables, instead of an opaque transport error.
func TestRedisClientUnconfigured(t *testing.T) {
	if _, err := NewRedisClient("", "").Do(context.Background(), "PING"); err == nil {
		t.Fatal("expected an error when the client is unconfigured")
	}
}

func TestParseIntReply(t *testing.T) {
	cases := []struct {
		body    string
		want    int64
		wantErr bool
	}{
		{body: "1", want: 1},
		{body: "[3]", want: 3},
		{body: "not-a-number", wantErr: true},
	}
	for _, tc := range cases {
		got, err := ParseIntReply([]byte(tc.body))
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseIntReply(%q): expected an error", tc.body)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseIntReply(%q): %v", tc.body, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseIntReply(%q) = %d, want %d", tc.body, got, tc.want)
		}
	}
}
