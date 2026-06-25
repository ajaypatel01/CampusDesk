package books

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ajaypatel01/CampusDesk/internal/platform/httpx"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ---- Books catalog ----

func (h *Handler) CreateBook(w http.ResponseWriter, r *http.Request) {
	var in CreateBookInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	b, err := h.svc.CreateBook(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, b)
}

func (h *Handler) ListBooks(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	items, err := h.svc.ListBooks(r.Context(), schoolID, r.URL.Query().Get("search"))
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *Handler) GetBook(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	b, err := h.svc.GetBook(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

func (h *Handler) UpdateBook(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in UpdateBookInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	b, err := h.svc.UpdateBook(r.Context(), id, in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, b)
}

func (h *Handler) DeleteBook(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.DeleteBook(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

// ---- Book Lists ----

func (h *Handler) CreateBookList(w http.ResponseWriter, r *http.Request) {
	var in CreateBookListInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	bl, err := h.svc.CreateBookList(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, bl)
}

func (h *Handler) ListBookLists(w http.ResponseWriter, r *http.Request) {
	schoolID, err := uuid.Parse(r.URL.Query().Get("school_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "school_id required")
		return
	}
	yearID, err := uuid.Parse(r.URL.Query().Get("academic_year_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "academic_year_id required")
		return
	}
	items, err := h.svc.ListBookLists(r.Context(), schoolID, yearID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (h *Handler) GetBookListDetail(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	detail, err := h.svc.GetBookListDetail(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, detail)
}

func (h *Handler) AddItemToList(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	var in AddItemInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	item, err := h.svc.AddItemToList(r.Context(), id, in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, item)
}

func (h *Handler) RemoveItemFromList(w http.ResponseWriter, r *http.Request) {
	itemID, err := uuid.Parse(chi.URLParam(r, "item_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid item_id")
		return
	}
	if err := h.svc.RemoveItemFromList(r.Context(), itemID); err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

// ---- Receipts ----

func (h *Handler) RecordReceipt(w http.ResponseWriter, r *http.Request) {
	var in RecordReceiptInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid json body")
		return
	}
	rec, err := h.svc.RecordReceipt(r.Context(), in)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, rec)
}

func (h *Handler) ListReceipts(w http.ResponseWriter, r *http.Request) {
	bookListID, err := uuid.Parse(r.URL.Query().Get("book_list_id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "book_list_id required")
		return
	}
	items, err := h.svc.ListReceipts(r.Context(), bookListID)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]interface{}{"items": items})
}

// ---- PDF ----

func (h *Handler) DownloadBookListPDF(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	pdfBytes, filename, err := h.svc.GenerateBookListPDF(r.Context(), id)
	if err != nil {
		httpx.WriteServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
}
