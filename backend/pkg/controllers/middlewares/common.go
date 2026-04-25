package middlewares

import (
	"homelab/pkg/common"
	"net/http"
)

// PingHandler godoc
// @Summary Ping the server
// @Description Returns pong if the server is alive
// @Tags system
// @Produce json
// @Success 200 {string} string "pong"
// @Router /auth/ping [get]
func PingHandler(w http.ResponseWriter, r *http.Request) {
	common.Success(w, r, "pong")
}
