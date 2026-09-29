package billing

// WebhookEvent is the subset of Razorpay's webhook payload shape this app
// reads. Razorpay always wraps the entities that changed under
// payload.<entity>.entity -- see https://razorpay.com/docs/webhooks/payloads/
// for the full shape; fields not listed here are simply ignored by Go's JSON
// decoder rather than causing an error.
type WebhookEvent struct {
	Event   string `json:"event"`
	Payload struct {
		Subscription struct {
			Entity WebhookSubscriptionEntity `json:"entity"`
		} `json:"subscription"`
		Payment struct {
			Entity WebhookPaymentEntity `json:"entity"`
		} `json:"payment"`
	} `json:"payload"`
}

type WebhookSubscriptionEntity struct {
	ID           string `json:"id"`
	PlanID       string `json:"plan_id"`
	Status       string `json:"status"`
	CurrentStart int64  `json:"current_start"` // unix seconds
	CurrentEnd   int64  `json:"current_end"`
}

type WebhookPaymentEntity struct {
	ID             string `json:"id"`
	SubscriptionID string `json:"subscription_id"`
	Amount         int    `json:"amount"` // paise
	Status         string `json:"status"`
	CreatedAt      int64  `json:"created_at"` // unix seconds
}
