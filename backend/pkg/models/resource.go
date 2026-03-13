package models

import (
	"context"
	"errors"
	"net/http"
	"regexp"
)

// Resource represents a unified container for configuration (Meta) and running state (Status).
type Resource[M any, S any] struct {
	ID              string `json:"id"`
	Meta            M      `json:"meta" validate:"required"`
	Status          S      `json:"status"`
	Generation      int64  `json:"generation"`      // Configuration version, increments only on Meta changes
	ResourceVersion int64  `json:"resourceVersion"` // Total object version, increments only on any change (Meta/Status)
}

// ConfigValidator defines the internal self-consistency validation interface that Meta models must implement.
type ConfigValidator interface {
	// Validate executes internal logic validation without relying on external databases or networks.
	Validate(ctx context.Context) error
}

// Binder defines the API parameter binding validation interface, mimicking render.Binder
type Binder interface {
	Bind(r *http.Request) error
}

// Bind delegates the HTTP request binding to the Meta structure, avoiding reflection overhead if possible.
func (r *Resource[M, S]) Bind(req *http.Request) error {
	if r.ID != "" {
		validID := regexp.MustCompile(`^[a-z0-9_\-]+$`)
		if !validID.MatchString(r.ID) {
			return errors.New("invalid id format, only lowercase letters, numbers, underscores and hyphens are allowed")
		}
	}

	// If the Meta type implements Bind, call it.
	if binder, ok := any(&r.Meta).(Binder); ok {
		return binder.Bind(req)
	}
	return nil
}
