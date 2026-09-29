// Package smsotp sends and verifies OTPs over SMS via MSG91's OTP API
// (https://docs.msg91.com/otp). MSG91 generates, stores, and expires the
// OTP entirely on their end -- there's deliberately no OTP-storage table in
// this app, just a request to send one and a request to verify one.
//
// Sending OTP SMS to Indian phone numbers requires the sender's own
// TRAI/DLT template registration on MSG91's dashboard -- something only the
// school itself can complete (business KYC), not something this code can
// do. Until AuthKey/TemplateID are configured, Enabled() is false and every
// call returns a clear "not configured" error rather than silently no-op'ing
// or failing in a confusing way.
package smsotp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

type Client struct {
	authKey    string
	senderID   string
	templateID string
	httpClient *http.Client
}

func New(authKey, senderID, templateID string) *Client {
	return &Client{
		authKey:    authKey,
		senderID:   senderID,
		templateID: templateID,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.authKey != "" && c.templateID != ""
}

var nonDigits = regexp.MustCompile(`[^0-9]`)

// normalizePhone strips everything but digits -- MSG91 expects a bare
// country-code-prefixed number (e.g. 919999999999), not "+91 99999 99999".
func normalizePhone(phone string) string {
	return nonDigits.ReplaceAllString(phone, "")
}

type msg91Response struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// SendOTP asks MSG91 to generate and text an OTP to phone using the
// configured DLT-approved template. The OTP itself never passes through
// this app -- MSG91 both generates and validates it.
func (c *Client) SendOTP(phone string) error {
	if !c.Enabled() {
		return fmt.Errorf("SMS OTP not configured")
	}
	q := url.Values{}
	q.Set("authkey", c.authKey)
	q.Set("template_id", c.templateID)
	q.Set("mobile", normalizePhone(phone))
	if c.senderID != "" {
		q.Set("sender", c.senderID)
	}
	return c.call("https://control.msg91.com/api/v5/otp?" + q.Encode())
}

// VerifyOTP checks otp against whatever MSG91 sent to phone. A false, nil
// return means "wrong or expired code", not a transport/config failure --
// callers should show a generic "invalid or expired OTP" either way.
func (c *Client) VerifyOTP(phone, otp string) (bool, error) {
	if !c.Enabled() {
		return false, fmt.Errorf("SMS OTP not configured")
	}
	q := url.Values{}
	q.Set("authkey", c.authKey)
	q.Set("mobile", normalizePhone(phone))
	q.Set("otp", otp)
	resp, err := c.httpClient.Get("https://control.msg91.com/api/v5/otp/verify?" + q.Encode())
	if err != nil {
		return false, fmt.Errorf("msg91 request: %w", err)
	}
	defer resp.Body.Close()
	var out msg91Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, fmt.Errorf("msg91 decode: %w", err)
	}
	if out.Type == "success" {
		return true, nil
	}
	return false, nil
}

func (c *Client) call(fullURL string) error {
	resp, err := c.httpClient.Get(fullURL)
	if err != nil {
		return fmt.Errorf("msg91 request: %w", err)
	}
	defer resp.Body.Close()
	var out msg91Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("msg91 decode: %w", err)
	}
	if out.Type != "success" {
		return fmt.Errorf("msg91 error: %s", out.Message)
	}
	return nil
}
