package guardian_auth

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GuardianMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			if claims["role"] != "guardian" {
				http.Error(w, "Not a guardian token", http.StatusForbidden)
				return
			}

			guardianID, err := uuid.Parse(claims["guardian_id"].(string))
			if err != nil {
				http.Error(w, "Invalid guardian ID", http.StatusUnauthorized)
				return
			}

			tenantID, err := uuid.Parse(claims["tenant_id"].(string))
			if err != nil {
				http.Error(w, "Invalid tenant ID", http.StatusUnauthorized)
				return
			}

			r.Header.Set("X-Guardian-ID", guardianID.String())
			r.Header.Set("X-Tenant-ID", tenantID.String())
			next.ServeHTTP(w, r)
		})
	}
}
