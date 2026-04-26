package controllers

import (
	"net/http"

	"homelab/pkg/common"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func BindRequest[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return req, false
	}
	if binder, ok := any(&req).(interface{ Bind(*http.Request) error }); ok {
		if err := binder.Bind(r); err != nil {
			common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
			return req, false
		}
	}
	return req, true
}

func BindOptionalRequest[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if r.ContentLength == 0 {
		return req, true
	}
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return req, false
	}
	if binder, ok := any(&req).(interface{ Bind(*http.Request) error }); ok {
		if err := binder.Bind(r); err != nil {
			common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
			return req, false
		}
	}
	return req, true
}

func DecodeJSONRequest[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return req, false
	}
	return req, true
}

func GetSearchCursorParams(r *http.Request) (string, int, string) {
	cursor, limit := GetCursorParams(r)
	return cursor, limit, r.URL.Query().Get("search")
}

func PathID(r *http.Request, param string) string {
	return chi.URLParam(r, param)
}
