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

	if errors.Is(err, common.ErrNotFound) {
		common.Error(w, r, http.StatusNotFound, 404, err.Error())
		return
	}

	if errors.Is(err, common.ErrBadRequest) {
		common.Error(w, r, http.StatusBadRequest, 400, err.Error())
		return
	}

	// Default to 500
	common.InternalServerError(w, r, 500, err.Error())
}

func getCursorParams(r *http.Request) (string, int) {
	cursor := r.URL.Query().Get("cursor")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return cursor, limit
}
