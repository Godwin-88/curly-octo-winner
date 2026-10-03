package mpesa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client is a minimal Safaricom Daraja API client for M-Pesa STK Push (Lipa Na M-Pesa Online).
// Sandbox base URL is used by default; switch to production URL in production.
type Client struct {
	consumerKey    string
	consumerSecret string
	passkey        string
	shortCode      string
	baseURL        string
	http           *http.Client

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

// RejectedError is a request Daraja answered and refused: nothing was sent to
// the customer's phone.
//
// Any other error from STKPush — a timeout, a dropped connection — leaves the
// outcome unknown: the prompt may already be on the phone.
type RejectedError struct {
	StatusCode int
	Body       string
}

func (e *RejectedError) Error() string {
	return fmt.Sprintf("M-Pesa refused the request (HTTP %d): %s", e.StatusCode, e.Body)
}

// ShortCode is the paybill or till this client collects into.
func (c *Client) ShortCode() string { return c.shortCode }

// NewClient creates a Daraja API client.
// url defaults to the Daraja sandbox URL if empty.
func NewClient(consumerKey, consumerSecret, passkey, shortCode, url string) *Client {
	if url == "" {
		url = "https://sandbox.safaricom.co.ke"
	}
	return &Client{
		consumerKey:    consumerKey,
		consumerSecret: consumerSecret,
		passkey:        passkey,
		shortCode:      shortCode,
		baseURL:        strings.TrimRight(url, "/"),
		http:           &http.Client{Timeout: 20 * time.Second},
	}
}

// tokenResponse is the OAuth2 access token response from Daraja.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

// accessToken obtains an OAuth2 token from the Daraja API.
func (c *Client) accessToken(ctx context.Context) (string, error) {
	// Daraja tokens last an hour; one is reused until a minute before it ends.
	c.mu.Lock()
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		token := c.token
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()

	creds := base64.StdEncoding.EncodeToString([]byte(c.consumerKey + ":" + c.consumerSecret))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/oauth/v1/generate?grant_type=client_credentials", nil)
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+creds)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("request access token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		// No token, so nothing was asked of the customer.
		return "", &RejectedError{StatusCode: resp.StatusCode, Body: "the consumer key or secret was not accepted"}
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("daraja returned empty access token")
	}
	c.mu.Lock()
	c.token = tr.AccessToken
	c.tokenExpiry = time.Now().Add(50 * time.Minute)
	c.mu.Unlock()
	return tr.AccessToken, nil
}

// STKPushRequest is the Lipa Na M-Pesa Online request payload.
type STKPushRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            string `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

// STKPushResponse is the response from the STK Push endpoint.
type STKPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

// STKPush initiates a Lipa Na M-Pesa Online STK push to the given phone.
// phone must be in format 2547XXXXXXXX. amount is in KES whole units.
func (c *Client) STKPush(ctx context.Context, phone, amount, accountRef, callbackURL string) (*STKPushResponse, error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, err
	}

	timestamp, password := c.password()

	payload := STKPushRequest{
		BusinessShortCode: c.shortCode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            amount,
		PartyA:            phone,
		PartyB:            c.shortCode,
		PhoneNumber:       phone,
		CallBackURL:       callbackURL,
		AccountReference:  accountRef,
		TransactionDesc:   "School Fee Payment",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal stk push payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/mpesa/stkpush/v1/processrequest", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create stk push request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute stk push: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read stk push response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &RejectedError{StatusCode: resp.StatusCode, Body: errorMessage(respBody)}
	}

	var stk STKPushResponse
	if err := json.Unmarshal(respBody, &stk); err != nil {
		return nil, fmt.Errorf("decode stk push response: %w", err)
	}
	if stk.ResponseCode != "0" || stk.CheckoutRequestID == "" {
		return nil, &RejectedError{StatusCode: resp.StatusCode, Body: firstNonEmpty(stk.ResponseDescription, "the request was not accepted")}
	}
	return &stk, nil
}

// EAT is the timezone Daraja timestamps are written in.
var EAT = time.FixedZone("EAT", 3*60*60)

func (c *Client) password() (timestamp, password string) {
	timestamp = time.Now().In(EAT).Format("20060102150405")
	password = base64.StdEncoding.EncodeToString([]byte(c.shortCode + c.passkey + timestamp))
	return timestamp, password
}

// STKQueryResult is what Daraja knows about an STK request.
type STKQueryResult struct {
	// Final is false while the customer has not answered the prompt yet.
	Final      bool
	ResultCode string
	ResultDesc string
}

// Paid reports that the customer completed the payment.
func (r STKQueryResult) Paid() bool { return r.Final && r.ResultCode == "0" }

// STKQuery asks Daraja for the outcome of an STK request. It is how a payment
// is settled when the callback never arrives.
func (c *Client) STKQuery(ctx context.Context, checkoutRequestID string) (*STKQueryResult, error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	timestamp, password := c.password()
	body, err := json.Marshal(map[string]string{
		"BusinessShortCode": c.shortCode,
		"Password":          password,
		"Timestamp":         timestamp,
		"CheckoutRequestID": checkoutRequestID,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/mpesa/stkpushquery/v1/query", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute stk query: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read stk query response: %w", err)
	}

	var parsed struct {
		ResultCode   json.RawMessage `json:"ResultCode"`
		ResultDesc   string          `json:"ResultDesc"`
		ErrorCode    string          `json:"errorCode"`
		ErrorMessage string          `json:"errorMessage"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode stk query response: %w", err)
	}
	// While the prompt is still on the phone Daraja answers with an error
	// ("The transaction is being processed") instead of a result.
	if len(parsed.ResultCode) == 0 {
		if resp.StatusCode == http.StatusOK || strings.Contains(strings.ToLower(parsed.ErrorMessage), "being processed") {
			return &STKQueryResult{Final: false, ResultDesc: parsed.ErrorMessage}, nil
		}
		return nil, fmt.Errorf("stk query: HTTP %d %s %s", resp.StatusCode, parsed.ErrorCode, parsed.ErrorMessage)
	}
	return &STKQueryResult{
		Final:      true,
		ResultCode: strings.Trim(string(parsed.ResultCode), `"`),
		ResultDesc: parsed.ResultDesc,
	}, nil
}

// errorMessage pulls Daraja's own explanation out of an error body.
func errorMessage(body []byte) string {
	var parsed struct {
		ErrorMessage string `json:"errorMessage"`
	}
	if json.Unmarshal(body, &parsed) == nil && parsed.ErrorMessage != "" {
		return parsed.ErrorMessage
	}
	text := strings.TrimSpace(string(body))
	if len(text) > 200 {
		text = text[:200] + "…"
	}
	return text
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
