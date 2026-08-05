package mpesa

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
}

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
		baseURL:        url,
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
		return "", fmt.Errorf("daraja token error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("daraja returned empty access token")
	}
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

	timestamp := time.Now().Format("20060102150405")
	password := base64.StdEncoding.EncodeToString([]byte(c.shortCode + c.passkey + timestamp))

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
		return nil, fmt.Errorf("daraja stk push error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var stk STKPushResponse
	if err := json.Unmarshal(respBody, &stk); err != nil {
		return nil, fmt.Errorf("decode stk push response: %w", err)
	}
	return &stk, nil
}
