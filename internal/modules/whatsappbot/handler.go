package whatsappbot

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/whatsapp"
)

type Handler struct {
	svc *Service
	wa  *whatsapp.Client
}

func NewHandler(svc *Service, wa *whatsapp.Client) *Handler {
	return &Handler{svc: svc, wa: wa}
}

// VerifyWebhook handles Meta's one-time GET handshake when this URL is
// registered (or re-verified) on the app dashboard: echo back hub.challenge
// if hub.verify_token matches what's configured, otherwise refuse.
func (h *Handler) VerifyWebhook(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("hub.mode") != "subscribe" || !h.wa.MatchesVerifyToken(r.URL.Query().Get("hub.verify_token")) {
		httpx.Error(w, http.StatusForbidden, "verification failed")
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(r.URL.Query().Get("hub.challenge")))
}

// ReceiveMessage is the actual inbound-message webhook. Always answers 200
// once the signature checks out, even when the reply itself couldn't be
// sent (unrecognized number, etc.) -- Meta retries on non-2xx, and none of
// this module's own failure modes are the kind a retry would fix.
func (h *Handler) ReceiveMessage(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "cannot read body")
		return
	}
	if !h.wa.VerifyWebhookSignature(body, r.Header.Get("X-Hub-Signature-256")) {
		httpx.Error(w, http.StatusUnauthorized, "invalid signature")
		return
	}
	var payload InboundPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	for _, phone := range payload.SenderPhones() {
		if err := h.svc.HandleInboundMessage(r.Context(), phone); err != nil {
			log.Printf("whatsappbot: reply to %s failed: %v", phone, err)
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
