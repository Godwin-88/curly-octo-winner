package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/shule360/api/pkg/httputil"
)

// AllowIPs restricts a route to requests originating from the given IP
// addresses or CIDR ranges (e.g. Safaricom Daraja callback egress IPs for the
// M-Pesa webhook). An empty list allows all traffic so development and
// sandbox integrations keep working; production should always set
// MPESA_ALLOWED_IPS.
func AllowIPs(allowed []string) func(http.Handler) http.Handler {
	if len(allowed) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	var networks []*net.IPNet
	for _, entry := range allowed {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			entry += "/32"
		}
		if _, ipNet, err := net.ParseCIDR(entry); err == nil {
			networks = append(networks, ipNet)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := net.ParseIP(ClientIP(r))
			if ip == nil {
				httputil.RespondForbidden(w, "FORBIDDEN", "Unable to verify source IP")
				return
			}
			for _, n := range networks {
				if n.Contains(ip) {
					next.ServeHTTP(w, r)
					return
				}
			}
			httputil.RespondForbidden(w, "FORBIDDEN", "Source not allowed")
		})
	}
}
