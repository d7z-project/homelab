package controllers

import (
	"context"
	"errors"
	"net/http"
)

type ContextKey string

var errControllerDependenciesNotConfigured = errors.New("controller dependencies not configured")

func WithValue[T any](key ContextKey, value T) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), key, value)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ValueFromRequest[T any](w http.ResponseWriter, r *http.Request, key ContextKey, valid func(T) bool) (T, bool) {
	var zero T
	value, ok := r.Context().Value(key).(T)
	if !ok {
		HandleError(w, r, errControllerDependenciesNotConfigured)
		return zero, false
	}
	if valid != nil && !valid(value) {
		HandleError(w, r, errControllerDependenciesNotConfigured)
		return zero, false
	}
	return value, true
}
