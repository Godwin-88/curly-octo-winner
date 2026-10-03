// Command devstub stands in for the hosted services the API depends on, so the
// whole stack can run on one machine (docker compose) without credentials and
// without sending a single real SMS.
//
// It is for local development only. The production image (api/Dockerfile)
// builds ./cmd/server alone and never contains this program.
//
// What it imitates:
//
//   - Supabase Auth password grant: POST /auth/v1/token
//     Any email is accepted with the password in DEV_PASSWORD
//     (default "password123"). Who the user is, and what they may do, still
//     comes from the staff table.
//     Accounts created or reset through the admin API (POST and PUT
//     /auth/v1/admin/users) use the password they were given instead, kept in
//     memory until the stub restarts.
//
//   - Africa's Talking bulk SMS: POST /version1/messaging
//     Nothing is sent. Each message is printed to the log, and a delivery
//     report is posted back to DLR_CALLBACK_URL a moment later, exactly as
//     Africa's Talking would. To exercise failures:
//     a number ending in 00 is refused at once (InvalidPhoneNumber),
//     a number ending in 99 is accepted and then reported Failed,
//     the message text "NOCREDIT" refuses every number (InsufficientBalance).
package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var counter atomic.Int64

// accounts created through the admin API: id -> email, email -> password.
var (
	accountsMu sync.Mutex
	emails     = map[string]string{}
	passwords  = map[string]string{}
)

func main() {
	password := envOr("DEV_PASSWORD", "password123")
	dlrURL := os.Getenv("DLR_CALLBACK_URL")
	port := envOr("PORT", "9090")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /auth/v1/token", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		accountsMu.Lock()
		expected, created := passwords[strings.ToLower(req.Email)]
		accountsMu.Unlock()
		if !created {
			expected = password
		}
		if req.Email == "" || req.Password != expected {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": "Invalid login credentials"})
			return
		}
		slog.Info("devstub: sign-in accepted", "email", req.Email)
		writeJSON(w, http.StatusOK, map[string]string{"access_token": "devstub-token", "token_type": "bearer"})
	})

	mux.HandleFunc("POST /auth/v1/admin/users", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		email := strings.ToLower(req.Email)
		accountsMu.Lock()
		defer accountsMu.Unlock()
		if _, exists := passwords[email]; exists {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error_code": "email_exists", "msg": "A user with this email address has already been registered"})
			return
		}
		id := fmt.Sprintf("00000000-0000-4000-8000-%012d", counter.Add(1))
		emails[id] = email
		passwords[email] = req.Password
		slog.Info("devstub: sign-in account created", "email", email)
		writeJSON(w, http.StatusOK, map[string]string{"id": id, "email": email})
	})

	mux.HandleFunc("PUT /auth/v1/admin/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Password string `json:"password"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		accountsMu.Lock()
		defer accountsMu.Unlock()
		email, ok := emails[r.PathValue("id")]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"msg": "User not found"})
			return
		}
		passwords[email] = req.Password
		slog.Info("devstub: password reset", "email", email)
		writeJSON(w, http.StatusOK, map[string]string{"id": r.PathValue("id"), "email": email})
	})

	mux.HandleFunc("POST /version1/messaging", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		message := r.PostForm.Get("message")
		from := r.PostForm.Get("from")
		numbers := strings.Split(r.PostForm.Get("to"), ",")

		type recipient struct {
			StatusCode int    `json:"statusCode"`
			Number     string `json:"number"`
			Cost       string `json:"cost"`
			Status     string `json:"status"`
			MessageID  string `json:"messageId"`
		}
		var recipients []recipient
		for _, number := range numbers {
			number = strings.TrimSpace(number)
			if number == "" {
				continue
			}
			switch {
			case strings.Contains(message, "NOCREDIT"):
				recipients = append(recipients, recipient{StatusCode: 405, Number: number, Cost: "0", Status: "InsufficientBalance", MessageID: "None"})
			case strings.HasSuffix(number, "00"):
				recipients = append(recipients, recipient{StatusCode: 403, Number: number, Cost: "0", Status: "InvalidPhoneNumber", MessageID: "None"})
			default:
				id := fmt.Sprintf("ATXid_dev_%d_%d", time.Now().UnixNano(), counter.Add(1))
				recipients = append(recipients, recipient{StatusCode: 101, Number: number, Cost: "KES 0.8000", Status: "Success", MessageID: id})
				slog.Info("devstub: SMS (not sent)", "to", number, "from", from, "message", message)
				if dlrURL != "" {
					go report(dlrURL, id, number)
				}
			}
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"SMSMessageData": map[string]any{
				"Message":    fmt.Sprintf("Sent to %d/%d", len(recipients), len(numbers)),
				"Recipients": recipients,
			},
		})
	})

	slog.Info("devstub listening", "port", port, "dlr_callback", dlrURL)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("devstub stopped", "error", err)
		os.Exit(1)
	}
}

// report posts a delivery report the way Africa's Talking does.
func report(dlrURL, id, number string) {
	time.Sleep(3 * time.Second)
	form := url.Values{"id": {id}, "phoneNumber": {number}, "status": {"Success"}}
	if strings.HasSuffix(number, "99") {
		form.Set("status", "Failed")
		form.Set("failureReason", "DeliveryFailure")
	}
	resp, err := http.PostForm(dlrURL, form)
	if err != nil {
		slog.Warn("devstub: delivery report not accepted", "error", err)
		return
	}
	resp.Body.Close()
	slog.Info("devstub: delivery report posted", "id", id, "status", form.Get("status"), "answer", resp.StatusCode)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
