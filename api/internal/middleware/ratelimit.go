package middleware

import (
	"net/http"

	"github.com/shule360/api/pkg/httputil"
	"github.com/shule360/api/pkg/upstash"
)

// RateLimit returns a middleware that rate-limits requests per tenant using a
// token-bucket algorithm backed by Upstash Redis.
// capacity: maximum burst size
// refillPerSecond: tokens added per second
func RateLimit(redis *upstash.RedisClient, capacity, refillPerSecond int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, ok := GetTenantID(r)
			if !ok {
				// If no tenant ID, allow through (auth middleware will catch it)
				next.ServeHTTP(w, r)
				return
			}

			key := "ratelimit:" + r.Method + ":" + tenantID.String()
			allowed, remaining, err := redis.TokenBucket(r.Context(), key, capacity, refillPerSecond)
			if err != nil {
				// If Redis is unavailable, allow through but log
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Remaining", http.StatusText(remaining))

			if !allowed {
				httputil.RespondError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests. Please try again later.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
