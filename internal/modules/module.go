package modules

import "github.com/go-chi/chi/v5"

// Module is implemented by each domain package to register its HTTP routes.
type Module interface {
	Name() string
	Mount(r chi.Router)
}
