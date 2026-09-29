package whatsappbot

// InboundPayload is the subset of Meta's WhatsApp webhook payload this app
// reads. See https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks/payload-examples
// for the full shape -- fields not listed here are simply ignored by Go's
// JSON decoder. Meta batches multiple changes/messages per call in theory;
// this app just walks every message in every entry/change and replies to
// each independently.
type InboundPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Messages []struct {
					From string `json:"from"` // sender's phone number, no "+", e.g. "919876543210"
					Type string `json:"type"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// SenderPhones extracts the "from" of every text-bearing message in the
// payload. Only "type": "text" is handled -- a sender sending a document,
// image, or reaction gets no reply, since there's no attached text to
// trigger the summary from (a future version could still reply to any
// message type; kept narrow for now).
func (p InboundPayload) SenderPhones() []string {
	var phones []string
	for _, entry := range p.Entry {
		for _, change := range entry.Changes {
			for _, msg := range change.Value.Messages {
				if msg.Type == "text" && msg.From != "" {
					phones = append(phones, msg.From)
				}
			}
		}
	}
	return phones
}
