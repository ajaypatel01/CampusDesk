// Package razorpay is a thin client for the subset of Razorpay's API this
// app needs for subscription billing: creating a subscription against a
// plan already set up on the Razorpay dashboard, and verifying the two
// kinds of signature Razorpay sends -- the one returned to the browser after
// checkout, and the one on each webhook call. Same shape as the other
// external-service clients in internal/platform (whatsapp, smsotp, email):
// Enabled() is false and every call is a no-op error until real credentials
// are configured, so the rest of the app can wire this up before the
// business's own Razorpay account exists.
package razorpay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	keyID         string
	keySecret     string
	webhookSecret string
	httpClient    *http.Client
}

// New constructs a client. webhookSecret is the separate secret configured
// under Razorpay's Webhooks dashboard page (not the API key secret) -- used
// only to verify inbound webhook calls, never sent to Razorpay.
func New(keyID, keySecret, webhookSecret string) *Client {
	return &Client{
		keyID:         keyID,
		keySecret:     keySecret,
		webhookSecret: webhookSecret,
		httpClient:    &http.Client{Timeout: 20 * time.Second},
	}
}

// Enabled reports whether real API credentials are configured. WebhookSecret
// is checked separately by WebhooksEnabled since a school could plausibly
// test subscription creation before the webhook endpoint is registered.
func (c *Client) Enabled() bool {
	return c != nil && c.keyID != "" && c.keySecret != ""
}

func (c *Client) WebhooksEnabled() bool {
	return c != nil && c.webhookSecret != ""
}

// Subscription is the subset of Razorpay's subscription object this app
// reads back. Status values: created, authenticated, active, pending,
// halted, cancelled, completed, expired -- see Razorpay's own docs for the
// full lifecycle; the webhook handler is what actually reacts to these.
type Subscription struct {
	ID         string `json:"id"`
	PlanID     string `json:"plan_id"`
	Status     string `json:"status"`
	ShortURL   string `json:"short_url"`
	CustomerID string `json:"customer_notify"`
}

// CreateSubscriptionInput mirrors Razorpay's create-subscription body. Notes
// carries our own school_id/plan_id so the webhook handler (which only gets
// Razorpay IDs) can look up which local row a given event belongs to without
// a second round trip.
type CreateSubscriptionInput struct {
	RazorpayPlanID string
	TotalCount     int // number of billing cycles Razorpay will run before stopping, e.g. 12 for a monthly plan billed for a year
	CustomerNotify bool
	Notes          map[string]string
}

func (c *Client) CreateSubscription(in CreateSubscriptionInput) (*Subscription, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("razorpay client not configured")
	}
	notify := 0
	if in.CustomerNotify {
		notify = 1
	}
	body := map[string]interface{}{
		"plan_id":         in.RazorpayPlanID,
		"total_count":     in.TotalCount,
		"customer_notify": notify,
		"notes":           in.Notes,
	}
	var sub Subscription
	if err := c.post("/v1/subscriptions", body, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

// CancelSubscription stops future billing. cancelAtCycleEnd=true lets the
// current paid period run out first rather than cancelling immediately.
func (c *Client) CancelSubscription(razorpaySubscriptionID string, cancelAtCycleEnd bool) error {
	if !c.Enabled() {
		return fmt.Errorf("razorpay client not configured")
	}
	body := map[string]interface{}{"cancel_at_cycle_end": cancelAtCycleEnd}
	return c.post(fmt.Sprintf("/v1/subscriptions/%s/cancel", razorpaySubscriptionID), body, nil)
}

func (c *Client) post(path string, body interface{}, out interface{}) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.razorpay.com"+path, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.keyID, c.keySecret)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("razorpay %s: status %d: %s", path, resp.StatusCode, string(respBody))
	}
	if out != nil {
		return json.Unmarshal(respBody, out)
	}
	return nil
}

// VerifyCheckoutSignature checks the signature Razorpay Checkout hands back
// to the browser on successful payment (razorpay_payment_id +
// razorpay_subscription_id, HMAC-SHA256 with the API key secret) -- verify
// this before trusting the frontend's "payment succeeded" callback, since
// that call is otherwise just an unauthenticated claim from the browser.
func (c *Client) VerifyCheckoutSignature(subscriptionID, paymentID, signature string) bool {
	if !c.Enabled() {
		return false
	}
	payload := paymentID + "|" + subscriptionID
	return hmacMatches(payload, c.keySecret, signature)
}

// VerifyWebhookSignature checks the X-Razorpay-Signature header against the
// raw request body (must be the exact bytes received, before any JSON
// re-encoding) using the separate webhook secret.
func (c *Client) VerifyWebhookSignature(rawBody []byte, signature string) bool {
	if !c.WebhooksEnabled() {
		return false
	}
	return hmacMatches(string(rawBody), c.webhookSecret, signature)
}

func hmacMatches(payload, secret, signature string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
