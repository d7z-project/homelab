package shared

import (
	"context"
	"errors"
	"net/http"
	"regexp"
)

var resourceIDRegex = regexp.MustCompile(`^[a-z0-9_\-]+$`)

// Resource represents a unified container for configuration (Meta) and running state (Status).
type Resource[M any, S any] struct {
	ID              string `json:"id" validate:"required"`
	Meta            M      `json:"meta" validate:"required"`
	Status          S      `json:"status" validate:"required"`
	Generation      int64  `json:"generation"`
	ResourceVersion int64  `json:"resourceVersion"`
}

// ConfigValidator defines internal self-consistency validation for resource specs.
type ConfigValidator interface {
	Validate(ctx context.Context) error
}

// Binder defines the API parameter binding validation interface.
type Binder interface {
	Bind(r *http.Request) error
}

// Bind delegates the HTTP request binding to the Meta structure.
func (r *Resource[M, S]) Bind(req *http.Request) error {
	if r.ID != "" && !resourceIDRegex.MatchString(r.ID) {
		return errors.New("invalid id format, only lowercase letters, numbers, underscores and hyphens are allowed")
	}
	if binder, ok := any(&r.Meta).(Binder); ok {
		return binder.Bind(req)
	}
	return nil
}
