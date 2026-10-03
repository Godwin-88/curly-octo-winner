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
//
//   - Safaricom Daraja: the access token, the STK push and its status query.
//     No phone is asked for anything. A moment later the result is posted to
//     the callback address the request named, as Safaricom would. By the last
//     two digits of the phone number:
//     00 the request is refused outright (no prompt),
//     99 the parent cancels the prompt,
//     98 the parent pays but no callback is ever sent (only the status query
//     reveals it),
//     anything else pays.
//     POST /dev/c2b imitates a parent paying the paybill from the M-Pesa menu:
//     form fields amount, account, shortcode (and optionally phone, name).
package main

import (
	"bytes"
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
	c2bURL := os.Getenv("C2B_CONFIRMATION_URL")

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

	mountDaraja(mux, c2bURL)

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

// --- Safaricom Daraja ---

// stkOutcomes holds what each STK request came to, for the status query.
var (
	stkMu       sync.Mutex
	stkOutcomes = map[string]string{} // checkout id -> result code
)

func mountDaraja(mux *http.ServeMux, c2bURL string) {
	mux.HandleFunc("GET /oauth/v1/generate", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"access_token": "dev-daraja-token", "expires_in": "3599"})
	})

	mux.HandleFunc("POST /mpesa/stkpush/v1/processrequest", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Amount           string `json:"Amount"`
			PhoneNumber      string `json:"PhoneNumber"`
			CallBackURL      string `json:"CallBackURL"`
			AccountReference string `json:"AccountReference"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if strings.HasSuffix(req.PhoneNumber, "00") {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"requestId": "dev", "errorCode": "400.002.02", "errorMessage": "Bad Request - Invalid PhoneNumber",
			})
			return
		}
		n := counter.Add(1)
		checkout := fmt.Sprintf("ws_CO_dev_%d_%d", time.Now().UnixNano(), n)
		merchant := fmt.Sprintf("dev-merchant-%d", n)
		slog.Info("devstub: M-Pesa prompt (no phone was asked)", "phone", req.PhoneNumber, "amount", req.Amount, "account", req.AccountReference)

		code, desc := "0", "The service request is processed successfully."
		if strings.HasSuffix(req.PhoneNumber, "99") {
			code, desc = "1032", "Request cancelled by user"
		}
		silent := strings.HasSuffix(req.PhoneNumber, "98")
		go settleSTK(req.CallBackURL, checkout, merchant, code, desc, req.Amount, req.PhoneNumber, silent)

		writeJSON(w, http.StatusOK, map[string]string{
			"MerchantRequestID": merchant, "CheckoutRequestID": checkout, "ResponseCode": "0",
			"ResponseDescription": "Success. Request accepted for processing",
			"CustomerMessage":     "Success. Request accepted for processing",
		})
	})

	mux.HandleFunc("POST /mpesa/stkpushquery/v1/query", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CheckoutRequestID string `json:"CheckoutRequestID"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		stkMu.Lock()
		code, done := stkOutcomes[req.CheckoutRequestID]
		stkMu.Unlock()
		if !done {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"requestId": "dev", "errorCode": "500.001.1001", "errorMessage": "The transaction is being processed",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"ResponseCode": "0", "CheckoutRequestID": req.CheckoutRequestID, "ResultCode": code, "ResultDesc": "dev result",
		})
	})

	mux.HandleFunc("POST /dev/c2b", func(w http.ResponseWriter, r *http.Request) {
		if c2bURL == "" {
			http.Error(w, "C2B_CONFIRMATION_URL is not set", http.StatusServiceUnavailable)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		transID := strings.ToUpper(fmt.Sprintf("DEV%X%d", time.Now().UnixNano()%0xFFFFFF, counter.Add(1)))
		if given := r.PostForm.Get("trans_id"); given != "" {
			transID = given
		}
		body, _ := json.Marshal(map[string]string{
			"TransactionType":   "Pay Bill",
			"TransID":           transID,
			"TransTime":         time.Now().In(time.FixedZone("EAT", 3*60*60)).Format("20060102150405"),
			"TransAmount":       r.PostForm.Get("amount"),
			"BusinessShortCode": r.PostForm.Get("shortcode"),
			"BillRefNumber":     r.PostForm.Get("account"),
			"MSISDN":            r.PostForm.Get("phone"),
			"FirstName":         r.PostForm.Get("name"),
		})
		resp, err := http.Post(c2bURL, "application/json", bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		resp.Body.Close()
		slog.Info("devstub: paybill payment posted", "trans_id", transID, "answer", resp.StatusCode)
		writeJSON(w, http.StatusOK, map[string]any{"trans_id": transID, "answer": resp.StatusCode})
	})
}

// settleSTK waits as long as a parent takes to enter a PIN, then reports the
// result the way Safaricom does.
func settleSTK(callbackURL, checkout, merchant, code, desc, amount, phone string, silent bool) {
	time.Sleep(3 * time.Second)
	stkMu.Lock()
	stkOutcomes[checkout] = code
	stkMu.Unlock()
	if silent || callbackURL == "" {
		slog.Info("devstub: M-Pesa result kept back (no callback)", "checkout", checkout, "result", code)
		return
	}

	callback := map[string]any{
		"MerchantRequestID": merchant, "CheckoutRequestID": checkout,
		"ResultCode": json.Number(code), "ResultDesc": desc,
	}
	if code == "0" {
		receipt := strings.ToUpper(fmt.Sprintf("SDEV%X", time.Now().UnixNano()%0xFFFFFFF))
		callback["CallbackMetadata"] = map[string]any{"Item": []map[string]any{
			{"Name": "Amount", "Value": json.Number(amount)},
			{"Name": "MpesaReceiptNumber", "Value": receipt},
			{"Name": "PhoneNumber", "Value": json.Number(phone)},
		}}
	}
	body, _ := json.Marshal(map[string]any{"Body": map[string]any{"stkCallback": callback}})
	resp, err := http.Post(callbackURL, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Warn("devstub: M-Pesa callback not accepted", "error", err)
		return
	}
	resp.Body.Close()
	slog.Info("devstub: M-Pesa callback posted", "checkout", checkout, "result", code, "answer", resp.StatusCode)
}
