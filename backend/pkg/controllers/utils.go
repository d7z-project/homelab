package controllers

import (
	"errors"
	"homelab/pkg/common"
	"homelab/pkg/common/auth"
	"net/http"
	"strconv"
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
