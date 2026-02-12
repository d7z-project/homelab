package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func Router(r chi.Router) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", PingHandler)
	})
}

// PingHandler godoc
// @Summary Ping the server
// @Description get string "pong"
// @Tags example
// @Accept  json
// @Produce  json
// @Success 200 {string} string "pong"
// @Router /ping [get]
func PingHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`"pong"`))
}
