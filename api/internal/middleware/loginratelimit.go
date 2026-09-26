package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shule360/api/pkg/httputil"
	"github.com/shule360/api/pkg/upstash"
)

// CheckLoginRateLimit is a fail-closed fixed-window rate limiter for
// unauthenticated login endpoints. It is called from inside the login handlers
// (after the request body is decoded) so it can key on both the client IP and
// the submitted identifier (email or phone).
//
// Unlike RateLimit (which fails open because it guards general API traffic),
// this fails *closed*: if Redis is unavailable, login attempts are rejected.
// A Redis outage should degrade login availability, not permit password/PIN
// brute-forcing.
//
// Returns false if the request must not proceed (a 429/503 has been written).
func CheckLoginRateLimit(ctx context.Context, w http.ResponseWriter, redis *upstash.RedisClient, ip, identifier string, windowSeconds, maxPerIP, maxPerIdentity int) bool {
	if redis == nil {
		httputil.RespondError(w, http.StatusServiceUnavailable, "RATE_LIMITER_UNAVAILABLE", "Login is temporarily unavailable. Please try again shortly.")
		return false
	}

	window := timeNow() / int64(windowSeconds)
	allowed := true

	if ip != "" {
		ok, err := incrWindow(ctx, redis, fmt.Sprintf("loginrl:ip:%s:%d", ip, window), windowSeconds*2)
		if err != nil {
			httputil.RespondError(w, http.StatusServiceUnavailable, "RATE_LIMITER_UNAVAILABLE", "Login is temporarily unavailable. Please try again shortly.")
			return false
		}
		allowed = allowed && ok
	}

	if allowed && identifier != "" {
		ok, err := incrWindow(ctx, redis, fmt.Sprintf("loginrl:id:%s:%d", strings.ToLower(identifier), window), windowSeconds*2)
		if err != nil {
			httputil.RespondError(w, http.StatusServiceUnavailable, "RATE_LIMITER_UNAVAILABLE", "Login is temporarily unavailable. Please try again shortly.")
			return false
		}
		allowed = allowed && ok
	}

	if !allowed {
		w.Header().Set("Retry-After", strconv.Itoa(windowSeconds))
		httputil.RespondError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many login attempts. Please try again in a few minutes.")
		return false
	}
	return true
}

// incrWindow increments the fixed-window counter and reports whether the
// caller may proceed (count <= max). EXPIRE is set only on the first
// increment so windows self-clean.
func incrWindow(ctx context.Context, redis *upstash.RedisClient, key string, ttlSeconds int) (bool, error) {
	body, err := redis.Do(ctx, "INCR", key)
	if err != nil {
		return false, err
	}
	count, err := upstash.ParseIntReply(body)
	if err != nil {
		return false, fmt.Errorf("parse INCR response: %w", err)
	}
	if count == 1 {
		if _, err := redis.Do(ctx, "EXPIRE", key, strconv.Itoa(ttlSeconds)); err != nil {
			return false, err
		}
	}
	return count <= int64(loginMaxAttempts), nil
}

// loginMaxAttempts bounds attempts per IP and per identifier within a window.
const loginMaxAttempts = 5

// timeNow is a variable (not a func) so tests can freeze time.
var timeNow = time.Now().Unix

// ClientIP prefers the first hop of X-Forwarded-For (Fly.io sets it at the
// edge) and falls back to the socket address. The port is stripped from the
// fallback: a "host:port" rate-limit key would give every TCP connection its
// own budget and quietly weaken per-IP brute-force protection.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.IndexByte(xff, ','); idx >= 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
