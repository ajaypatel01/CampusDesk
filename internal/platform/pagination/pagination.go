package pagination

import (
	"net/http"
	"strconv"
)

type Params struct {
	Limit  int
	Offset int
}

const (
	defaultLimit = 20
	maxLimit     = 100
)

func FromRequest(r *http.Request) Params {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}

	return Params{Limit: limit, Offset: offset}
}

type ListResponse struct {
	Items  interface{} `json:"items"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

func NewListResponse(items interface{}, total, limit, offset int) ListResponse {
	return ListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}
