// Package fast2sms sends one-time codes by SMS through Fast2SMS's OTP route
// (https://docs.fast2sms.com). The OTP route needs no DLT template: the SMS
// reads "Your OTP: 123456" from Fast2SMS's own sender ID.
package fast2sms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultURL = "https://www.fast2sms.com/dev/bulkV2"

type Client struct {
	apiKey     string
	url        string
	httpClient *http.Client
}

// New returns a client; url "" means Fast2SMS itself (others: local testing).
func New(apiKey, url string) *Client {
	if url == "" {
		url = defaultURL
	}
	return &Client{apiKey: apiKey, url: url, httpClient: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) Enabled() bool { return c != nil && c.apiKey != "" }

// SendOTP texts code to a 10-digit Indian mobile number.
func (c *Client) SendOTP(phone, code string) error {
	if !c.Enabled() {
		return fmt.Errorf("fast2sms not configured")
	}
	body, _ := json.Marshal(map[string]string{
		"route":            "otp",
		"variables_values": code,
		"numbers":          phone,
	})
	req, err := http.NewRequest("POST", c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var out struct {
		Return  bool        `json:"return"`
		Message interface{} `json:"message"` // a string, or a list of strings on errors
	}
	if err := json.Unmarshal(raw, &out); err != nil || !out.Return {
		return fmt.Errorf("fast2sms %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}
