package controllers

import (
	"net"
	"net/http"
	"strconv"
	"strings"
)

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
