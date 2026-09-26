package middleware

import (
	"context"
	"net/http"
	"testing"
)

// Rate-limit keys are derived from ClientIP, so the value must be stable per
// client: "host:port" keys would give each TCP connection its own attempt
// budget and weaken login brute-force protection.
func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		remoteAddr string
		want       string
	}{
		{name: "single forwarded hop", xff: "203.0.113.5", remoteAddr: "10.0.0.1:5555", want: "203.0.113.5"},
		{name: "multiple forwarded hops takes the client", xff: "203.0.113.5, 70.41.3.18, 150.172.238.178", remoteAddr: "10.0.0.1:5555", want: "203.0.113.5"},
		{name: "forwarded hop with padding", xff: "  198.51.100.7 , 10.0.0.9", remoteAddr: "10.0.0.1:5555", want: "198.51.100.7"},
		{name: "socket address loses its port", remoteAddr: "127.0.0.1:54321", want: "127.0.0.1"},
		{name: "ipv6 socket address loses its port", remoteAddr: "[::1]:8080", want: "::1"},
		{name: "socket address without a port is used as-is", remoteAddr: "192.0.2.10", want: "192.0.2.10"},
		{name: "empty remote address", remoteAddr: "", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{RemoteAddr: tc.remoteAddr, Header: http.Header{}}
			if tc.xff != "" {
				r.Header.Set("X-Forwarded-For", tc.xff)
			}
			if got := ClientIP(r); got != tc.want {
				t.Errorf("ClientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

// A missing or unreachable Redis must fail *closed* for login: 503, never a
// silent allow.
func TestCheckLoginRateLimitFailsClosedWithoutRedis(t *testing.T) {
	w := &responseRecorder{header: http.Header{}}
	if CheckLoginRateLimit(context.Background(), w, nil, "127.0.0.1", "demo:key", 900, 5, 5) {
		t.Fatal("expected the request to be rejected when Redis is unavailable")
	}
	if w.status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want %d", w.status, http.StatusServiceUnavailable)
	}
}

type responseRecorder struct {
	header http.Header
	status int
	body   []byte
}

func (r *responseRecorder) Header() http.Header { return r.header }

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return len(b), nil
}

func (r *responseRecorder) WriteHeader(status int) { r.status = status }
