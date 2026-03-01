package middlewares

import (
	"homelab/pkg/common"
	"net/http"
)

func PingHandler(w http.ResponseWriter, r *http.Request) {
	common.Success(w, r, "pong")
}
