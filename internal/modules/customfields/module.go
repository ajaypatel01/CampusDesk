package customfields

import (
	"encoding/json"
	"net/http"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Module struct {
	svc *Service
}

func New(pool *pgxpool.Pool) *Module {
	return &Module{svc: NewService(NewRepository(pool))}
}

func (m *Module) Name() string { return "customfields" }

// Mount gates every route to super_admin: the whole feature -- defining
// fields and seeing/entering their values -- is super_admin-only, not just
// the ability to add a new field.
func (m *Module) Mount(r chi.Router) {
	r.Route("/custom-fields", func(r chi.Router) {
		r.Use(httpx.RequireRole("super_admin"))
		r.Route("/definitions", func(r chi.Router) {
			r.Get("/", m.ListDefinitions)
			r.Post("/", m.CreateDefinition)
			r.Route("/{id}", func(r chi.Router) {
				r.Put("/", m.UpdateDefinition)
				r.Delete("/", m.DeleteDefinition)
			})
		})
		r.Route("/values", func(r chi.Router) {
			r.Get("/", m.ListValues)
			r.Put("/", m.UpsertValue)
		})
	})
}

func (m *Module) CreateDefinition(w http.ResponseWriter, r *http.Request) {
	var in DefinitionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	claims := httpx.ClaimsFromContext(r.Context())
	var createdBy uuid.UUID
	if claims != nil {
		createdBy, _ = uuid.Parse(claims.Sub)
	}
	d, err := m.svc.CreateDefinition(r.Context(), in, createdBy)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, d)
}

func (m *Module) ListDefinitions(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	entityType := r.URL.Query().Get("entity_type")
	items, err := m.svc.ListDefinitions(r.Context(), schoolID, entityType)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (m *Module) UpdateDefinition(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in UpdateDefinitionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	d, err := m.svc.UpdateDefinition(r.Context(), id, in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, d)
}

func (m *Module) DeleteDefinition(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := m.svc.DeleteDefinition(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

func (m *Module) ListValues(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	entityID, err := uuid.Parse(r.URL.Query().Get("entity_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "entity_id required")
		return
	}
	entityType := r.URL.Query().Get("entity_type")
	scopeID := NoScope
	if v := r.URL.Query().Get("scope_id"); v != "" {
		scopeID, err = uuid.Parse(v)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid scope_id")
			return
		}
	}
	items, err := m.svc.ListValuesForEntity(r.Context(), schoolID, entityType, entityID, scopeID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (m *Module) UpsertValue(w http.ResponseWriter, r *http.Request) {
	var in UpsertValueInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if in.ScopeID == uuid.Nil {
		in.ScopeID = NoScope
	}
	if err := m.svc.UpsertValue(r.Context(), in); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "saved"})
}
