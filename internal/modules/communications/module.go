package communications

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/ajaypatel01/CampusDesk/internal/platform/whatsapp"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	repo *Repository
	wa   *whatsapp.Client
}

func New(pool *pgxpool.Pool, wa *whatsapp.Client) *Module {
	return &Module{repo: NewRepository(pool), wa: wa}
}

func (m *Module) Name() string { return "communications" }

func (m *Module) Mount(r chi.Router) {
	r.Route("/broadcasts", func(r chi.Router) {
		r.Get("/", m.ListBroadcasts)
		r.Post("/", m.SendBroadcast)
		r.Get("/{id}/recipients", m.ListRecipients)
	})
}

type SendBroadcastInput struct {
	SchoolID       string   `json:"school_id"`
	AcademicYearID string   `json:"academic_year_id"`
	Title          string   `json:"title"`
	Message        string   `json:"message"`
	Target         string   `json:"target"` // manual | grade | all_parents | staff
	GradeLevelID   string   `json:"grade_level_id"`
	Phones         []string `json:"phones"` // used when target=manual
	SentBy         string   `json:"sent_by"`
	// Template options (optional)
	IsTemplate   bool     `json:"is_template"`
	TemplateName string   `json:"template_name"`
	TemplateLang string   `json:"template_lang"`
	BodyParams   []string `json:"body_params"` // positional params for template body
}

func (m *Module) SendBroadcast(w http.ResponseWriter, r *http.Request) {
	if !m.wa.Enabled() {
		httpx.Error(w, http.StatusServiceUnavailable, "WhatsApp not configured")
		return
	}

	var in SendBroadcastInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	schoolID, _ := uuid.Parse(in.SchoolID)
	if schoolID == uuid.Nil || in.Title == "" || in.Message == "" {
		httpx.Error(w, http.StatusBadRequest, "school_id, title, message required")
		return
	}
	if in.Target == "" {
		in.Target = "manual"
	}

	// Resolve recipients
	phones, err := m.resolveRecipients(r.Context(), &in, schoolID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "resolve recipients: "+err.Error())
		return
	}
	if len(phones) == 0 {
		httpx.Error(w, http.StatusBadRequest, "no recipients found")
		return
	}

	b := &Broadcast{
		SchoolID:     schoolID,
		Title:        in.Title,
		Message:      in.Message,
		Target:       in.Target,
		IsTemplate:   in.IsTemplate,
		TemplateName: in.TemplateName,
		TemplateLang: in.TemplateLang,
		TotalCount:   len(phones),
	}
	if gid, err := uuid.Parse(in.GradeLevelID); err == nil && gid != uuid.Nil {
		b.GradeLevelID = &gid
	}
	if sid, err := uuid.Parse(in.SentBy); err == nil && sid != uuid.Nil {
		b.SentBy = &sid
	}
	if in.TemplateLang == "" {
		b.TemplateLang = "en_US"
	}

	if err := m.repo.CreateBroadcast(r.Context(), b); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}

	// Send in background so the HTTP response returns immediately
	go m.deliverBroadcast(context.Background(), b, phones, &in)

	httpx.JSON(w, http.StatusAccepted, map[string]interface{}{
		"id":          b.ID,
		"status":      "sending",
		"total_count": b.TotalCount,
	})
}

func (m *Module) resolveRecipients(ctx context.Context, in *SendBroadcastInput, schoolID uuid.UUID) ([]PhoneEntry, error) {
	switch in.Target {
	case "manual":
		var phones []PhoneEntry
		for _, p := range in.Phones {
			if p != "" {
				phones = append(phones, PhoneEntry{Phone: p})
			}
		}
		return phones, nil

	case "grade":
		gradeID, _ := uuid.Parse(in.GradeLevelID)
		yearID, _ := uuid.Parse(in.AcademicYearID)
		if gradeID == uuid.Nil || yearID == uuid.Nil {
			return nil, nil
		}
		return m.repo.LookupGradeParentPhones(ctx, schoolID, gradeID, yearID)

	case "all_parents":
		return m.repo.LookupAllParentPhones(ctx, schoolID)

	case "staff":
		return m.repo.LookupStaffPhones(ctx, schoolID)

	default:
		return nil, nil
	}
}

func (m *Module) deliverBroadcast(ctx context.Context, b *Broadcast, phones []PhoneEntry, in *SendBroadcastInput) {
	var sentCount, failedCount int

	for _, entry := range phones {
		now := time.Now()
		rec := &BroadcastRecipient{
			BroadcastID: b.ID,
			Phone:       entry.Phone,
			Name:        entry.Name,
			SentAt:      &now,
		}

		var sendErr error
		if in.IsTemplate && in.TemplateName != "" {
			lang := in.TemplateLang
			if lang == "" {
				lang = "en_US"
			}
			sendErr = m.wa.SendTemplate(entry.Phone, in.TemplateName, lang, in.BodyParams)
		} else {
			sendErr = m.wa.SendText(entry.Phone, in.Message)
		}

		if sendErr != nil {
			rec.Status = "failed"
			rec.ErrorMessage = sendErr.Error()
			rec.SentAt = nil
			failedCount++
		} else {
			rec.Status = "sent"
			sentCount++
		}

		// best-effort log; ignore individual insert errors
		_ = m.repo.AddRecipient(ctx, rec)
	}

	status := "done"
	if sentCount == 0 {
		status = "failed"
	}
	_ = m.repo.UpdateBroadcastCounts(ctx, b.ID, sentCount, failedCount, status)
}

func (m *Module) ListBroadcasts(w http.ResponseWriter, r *http.Request) {
	schoolID, _ := uuid.Parse(r.URL.Query().Get("school_id"))
	if schoolID == uuid.Nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	items, err := m.repo.ListBroadcasts(r.Context(), schoolID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (m *Module) ListRecipients(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid broadcast id")
		return
	}
	items, err := m.repo.ListRecipients(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}
