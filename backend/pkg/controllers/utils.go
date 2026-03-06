package controllers

import (
	"errors"
	"homelab/pkg/common"
	"homelab/pkg/common/auth"
	"net"
	"net/http"
	"strconv"
	"strings"
)

func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	if errors.Is(err, auth.ErrPermissionDenied) {
		common.Error(w, r, http.StatusForbidden, 10002, err.Error())
		return
	}

	if errors.Is(err, auth.ErrUnauthorized) {
		common.Error(w, r, http.StatusUnauthorized, 10000, err.Error())
		return
	}

	if errors.Is(err, auth.ErrTotpRequired) {
		common.Error(w, r, http.StatusUnauthorized, 10001, err.Error())
		return
	}

	// Default to 500
	common.InternalServerError(w, r, 500, err.Error())
}

func getPaginationParams(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 {
		pageSize = 15
	}
	return page, pageSize
}

func GetIP(r *http.Request) string {
	var ip string
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip = strings.TrimSpace(strings.Split(xff, ",")[0])
	} else if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip = xri
	} else {
		ip = r.RemoteAddr
	}

	// Strip port if present (e.g. "127.0.0.1:1234" or "[::1]:1234")
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}
