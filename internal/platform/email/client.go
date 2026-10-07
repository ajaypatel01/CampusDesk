package email

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client sends email through Resend (https://resend.com/docs/api-reference/emails/send-email).
// The sending domain (the part after @ in from) must be verified in Resend.
type Client struct {
	apiKey   string
	from     string
	fromName string
	http     *http.Client
	endpoint string
}

func New(apiKey, from, fromName string) *Client {
	return &Client{
		apiKey: apiKey, from: from, fromName: fromName,
		http:     &http.Client{Timeout: 20 * time.Second},
		endpoint: "https://api.resend.com/emails",
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.apiKey != "" && c.from != ""
}

type attachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"` // base64
	ContentType string `json:"content_type,omitempty"`
}

type message struct {
	From        string       `json:"from"`
	To          []string     `json:"to"`
	Subject     string       `json:"subject"`
	HTML        string       `json:"html"`
	Attachments []attachment `json:"attachments,omitempty"`
}

// SendPDF sends an HTML email with one PDF attached.
func (c *Client) SendPDF(toEmail, toName, subject, htmlBody, filename string, pdfData []byte) error {
	return c.send(toEmail, toName, subject, htmlBody, []attachment{{
		Filename:    filename,
		Content:     base64.StdEncoding.EncodeToString(pdfData),
		ContentType: "application/pdf",
	}})
}

// SendText sends an HTML email without attachments.
func (c *Client) SendText(toEmail, toName, subject, htmlBody string) error {
	return c.send(toEmail, toName, subject, htmlBody, nil)
}

func (c *Client) send(toEmail, toName, subject, htmlBody string, attachments []attachment) error {
	if !c.Enabled() {
		return fmt.Errorf("email client not configured")
	}
	to := toEmail
	if toName != "" {
		to = fmt.Sprintf("%s <%s>", quoteName(toName), toEmail)
	}
	from := c.from
	if c.fromName != "" {
		from = fmt.Sprintf("%s <%s>", quoteName(c.fromName), c.from)
	}
	body, err := json.Marshal(message{From: from, To: []string{to}, Subject: subject, HTML: htmlBody, Attachments: attachments})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		var e struct {
			Name    string `json:"name"`
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &e) == nil && e.Message != "" {
			return fmt.Errorf("resend error %d (%s): %s", resp.StatusCode, e.Name, e.Message)
		}
		return fmt.Errorf("resend error %d: %s", resp.StatusCode, bytes.TrimSpace(raw))
	}
	return nil
}

// quoteName wraps a display name in quotes when it holds characters that are
// special in an address header (comma, angle brackets, quotes ...).
func quoteName(name string) string {
	if bytes.ContainsAny([]byte(name), `,<>"()[]:;@\`) {
		b, _ := json.Marshal(name) // escapes inner quotes and backslashes
		return string(b)
	}
	return name
}
