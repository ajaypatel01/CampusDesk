package whatsapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	phoneNumberID string
	accessToken   string
	apiVersion    string
	httpClient    *http.Client
}

func New(phoneNumberID, accessToken, apiVersion string) *Client {
	if apiVersion == "" {
		apiVersion = "v19.0"
	}
	return &Client{
		phoneNumberID: phoneNumberID,
		accessToken:   accessToken,
		apiVersion:    apiVersion,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.phoneNumberID != "" && c.accessToken != ""
}

func (c *Client) baseURL() string {
	return fmt.Sprintf("https://graph.facebook.com/%s/%s", c.apiVersion, c.phoneNumberID)
}

// SendDocument uploads a PDF to Meta media storage then sends it to the recipient.
func (c *Client) SendDocument(toPhone, caption, filename string, pdfBytes []byte) error {
	if !c.Enabled() {
		return fmt.Errorf("whatsapp client not configured")
	}
	mediaID, err := c.uploadMedia(pdfBytes, filename)
	if err != nil {
		return fmt.Errorf("upload media: %w", err)
	}
	return c.sendDocumentByMediaID(toPhone, mediaID, caption, filename)
}

// SendText sends a plain WhatsApp text message.
func (c *Client) SendText(toPhone, text string) error {
	if !c.Enabled() {
		return fmt.Errorf("whatsapp client not configured")
	}
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                sanitizePhone(toPhone),
		"type":              "text",
		"text":              map[string]string{"body": text},
	}
	return c.postMessages(payload)
}

func (c *Client) uploadMedia(data []byte, filename string) (string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	_ = w.WriteField("messaging_product", "whatsapp")
	_ = w.WriteField("type", "application/pdf")

	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, bytes.NewReader(data)); err != nil {
		return "", err
	}
	w.Close()

	req, err := http.NewRequest("POST", c.baseURL()+"/media", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode media response: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("empty media_id from meta api (status %d)", resp.StatusCode)
	}
	return result.ID, nil
}

func (c *Client) sendDocumentByMediaID(toPhone, mediaID, caption, filename string) error {
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                sanitizePhone(toPhone),
		"type":              "document",
		"document": map[string]string{
			"id":       mediaID,
			"filename": filename,
			"caption":  caption,
		},
	}
	return c.postMessages(payload)
}

func (c *Client) postMessages(payload interface{}) error {
	b, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", c.baseURL()+"/messages", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("meta api %d: %s", resp.StatusCode, body)
	}
	return nil
}

// SendTemplate sends a pre-approved Meta template message.
// bodyParams are positional replacements for {{1}}, {{2}}, ... in the template body.
func (c *Client) SendTemplate(toPhone, templateName, language string, bodyParams []string) error {
	if !c.Enabled() {
		return fmt.Errorf("whatsapp client not configured")
	}

	var components []map[string]interface{}
	if len(bodyParams) > 0 {
		params := make([]map[string]string, 0, len(bodyParams))
		for _, p := range bodyParams {
			params = append(params, map[string]string{"type": "text", "text": p})
		}
		components = append(components, map[string]interface{}{
			"type":       "body",
			"parameters": params,
		})
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                sanitizePhone(toPhone),
		"type":              "template",
		"template": map[string]interface{}{
			"name":       templateName,
			"language":   map[string]string{"code": language},
			"components": components,
		},
	}
	return c.postMessages(payload)
}

// sanitizePhone strips spaces and dashes; E.164 format expected (e.g. 919876543210).
func sanitizePhone(phone string) string {
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.TrimPrefix(phone, "+")
	return phone
}
