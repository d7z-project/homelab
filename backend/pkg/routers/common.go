package routers

import (
	"homelab/pkg/common"
	"net/http"
)

// PingHandler godoc
// @Summary Ping the server
// @Description get string "pong"
// @Tags example
// @Accept  json
// @Produce  json
// @Success 200 {string} string "pong"
// @Router /ping [get]
func PingHandler(w http.ResponseWriter, r *http.Request) {
	common.Success(w, r, "pong")
}
