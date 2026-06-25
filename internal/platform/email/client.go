package email

import (
	"encoding/base64"
	"fmt"

	sgmail "github.com/sendgrid/sendgrid-go/helpers/mail"
	sendgrid "github.com/sendgrid/sendgrid-go"
)

type Client struct {
	apiKey   string
	from     string
	fromName string
}

func New(apiKey, from, fromName string) *Client {
	return &Client{apiKey: apiKey, from: from, fromName: fromName}
}

func (c *Client) Enabled() bool {
	return c != nil && c.apiKey != "" && c.from != ""
}

// SendPDF sends a single PDF attachment email via SendGrid.
func (c *Client) SendPDF(toEmail, toName, subject, htmlBody, filename string, pdfData []byte) error {
	if !c.Enabled() {
		return fmt.Errorf("email client not configured")
	}

	from := sgmail.NewEmail(c.fromName, c.from)
	to := sgmail.NewEmail(toName, toEmail)

	message := sgmail.NewSingleEmail(from, subject, to, "", htmlBody)

	encoded := base64.StdEncoding.EncodeToString(pdfData)
	attachment := &sgmail.Attachment{
		Content:     encoded,
		Type:        "application/pdf",
		Filename:    filename,
		Disposition: "attachment",
	}
	message.AddAttachment(attachment)

	client := sendgrid.NewSendClient(c.apiKey)
	resp, err := client.Send(message)
	if err != nil {
		return fmt.Errorf("sendgrid send: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid error %d: %s", resp.StatusCode, resp.Body)
	}
	return nil
}

// SendText sends a plain-text email without attachments.
func (c *Client) SendText(toEmail, toName, subject, htmlBody string) error {
	if !c.Enabled() {
		return fmt.Errorf("email client not configured")
	}

	from := sgmail.NewEmail(c.fromName, c.from)
	to := sgmail.NewEmail(toName, toEmail)
	message := sgmail.NewSingleEmail(from, subject, to, "", htmlBody)

	client := sendgrid.NewSendClient(c.apiKey)
	resp, err := client.Send(message)
	if err != nil {
		return fmt.Errorf("sendgrid send: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid error %d: %s", resp.StatusCode, resp.Body)
	}
	return nil
}
