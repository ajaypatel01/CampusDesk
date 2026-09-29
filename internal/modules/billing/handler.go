package billing

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc           *Service
	razorpayKeyID string
}

func NewHandler(svc *Service, razorpayKeyID string) *Handler {
	return &Handler{svc: svc, razorpayKeyID: razorpayKeyID}
}

func (h *Handler) ListPlans(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListPlans(r.Context(), true)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items, "razorpay_key_id": h.razorpayKeyID})
}

func (h *Handler) ListAllPlans(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListPlans(r.Context(), false)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *Handler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var in PlanInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	p, err := h.svc.CreatePlan(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in PlanInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	p, err := h.svc.UpdatePlan(r.Context(), id, in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	sub, err := h.svc.GetSubscription(r.Context(), schoolID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	payments, err := h.svc.ListPayments(r.Context(), sub.ID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"subscription": sub, "payments": payments})
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SchoolID     string `json:"school_id"`
		PlanID       string `json:"plan_id"`
		BillingCycle string `json:"billing_cycle"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	schoolID, err := uuid.Parse(in.SchoolID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid school_id")
		return
	}
	planID, err := uuid.Parse(in.PlanID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid plan_id")
		return
	}
	result, err := h.svc.Subscribe(r.Context(), schoolID, planID, in.BillingCycle)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	result.RazorpayKeyID = h.razorpayKeyID
	httpx.JSON(w, http.StatusOK, result)
}

func (h *Handler) ConfirmCheckout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RazorpaySubscriptionID string `json:"razorpay_subscription_id"`
		RazorpayPaymentID      string `json:"razorpay_payment_id"`
		RazorpaySignature      string `json:"razorpay_signature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.svc.ConfirmCheckout(r.Context(), in.RazorpaySubscriptionID, in.RazorpayPaymentID, in.RazorpaySignature); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "confirmed"})
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SchoolID         string `json:"school_id"`
		CancelAtCycleEnd bool   `json:"cancel_at_cycle_end"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	schoolID, err := uuid.Parse(in.SchoolID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid school_id")
		return
	}
	if err := h.svc.Cancel(r.Context(), schoolID, in.CancelAtCycleEnd); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// Webhook is a public endpoint (no JWT -- Razorpay calls this directly, not
// a logged-in user) authenticated purely by the X-Razorpay-Signature header
// against the raw body. Always returns 200 once the signature checks out,
// even for event types HandleWebhookEvent ignores, so Razorpay doesn't retry
// events this app doesn't act on.
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB cap, Razorpay payloads are small
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "cannot read body")
		return
	}
	if !h.svc.rzp.VerifyWebhookSignature(body, r.Header.Get("X-Razorpay-Signature")) {
		httpx.Error(w, http.StatusUnauthorized, "invalid signature")
		return
	}
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.svc.HandleWebhookEvent(r.Context(), event); err != nil {
		// Logged server-side by WriteServiceError's caller conventions
		// elsewhere; a webhook failure still gets acknowledged so Razorpay
		// doesn't retry-storm on a permanent (not transient) error such as
		// an unrecognized subscription id.
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "error", "detail": err.Error()})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
