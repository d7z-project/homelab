package common

import (
	"net/http"

	"github.com/go-chi/render"
	"gopkg.d7z.net/middleware/kv"
)

var DB kv.KV

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (rd *Response) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func Success(w http.ResponseWriter, r *http.Request, data interface{}) {
	render.Status(r, http.StatusOK)
	render.JSON(w, r, data)
}

func Error(w http.ResponseWriter, r *http.Request, httpStatus int, code int, message string) {
	render.Status(r, httpStatus)
	_ = render.Render(w, r, &Response{
		Code:    code,
		Message: message,
	})
}

func BadRequestError(w http.ResponseWriter, r *http.Request, code int, message string) {
	Error(w, r, http.StatusBadRequest, code, message)
}

func InternalServerError(w http.ResponseWriter, r *http.Request, code int, message string) {
	Error(w, r, http.StatusInternalServerError, code, message)
}

func UnauthorizedError(w http.ResponseWriter, r *http.Request, code int, message string) {
	Error(w, r, http.StatusUnauthorized, code, message)
}
