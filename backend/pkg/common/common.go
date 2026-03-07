package common

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/spf13/afero"
	"gopkg.d7z.net/middleware/kv"
	"gopkg.d7z.net/middleware/lock"
	"gopkg.d7z.net/middleware/subscribe"
)

var DB kv.KV
var Locker lock.Locker
var Subscriber subscribe.Subscriber
var FS afero.Fs
var TempDir afero.Fs

// --- 分布式协调辅助函数 ---

// UpdateGlobalVersion 更新指定模块的全局版本号
func UpdateGlobalVersion(ctx context.Context, module string) int64 {
	version := time.Now().UnixNano()
	_ = DB.Child("system", "sync", "version").Put(ctx, module, strconv.FormatInt(version, 10), kv.TTLKeep)
	return version
}

// GetGlobalVersion 获取指定模块的全局版本号
func GetGlobalVersion(ctx context.Context, module string) int64 {
	val, err := DB.Child("system", "sync", "version").Get(ctx, module)
	if err != nil || val == "" {
		return 0
	}
	v, _ := strconv.ParseInt(val, 10, 64)
	return v
}

// NotifyCluster 发送集群广播事件
func NotifyCluster(ctx context.Context, event string, payload string) {
	if Subscriber != nil {
		_ = Subscriber.Publish(ctx, "homelab:cluster:events", event+":"+payload)
	}
}

// WithJitter 执行带随机抖动的函数，用于平滑 I/O 惊群
func WithJitter(fn func(), maxDelay time.Duration) {
	delay := time.Duration(rand.Int63n(int64(maxDelay)))
	time.AfterFunc(delay, fn)
}

var ErrNotFound = errors.New("resource not found")
var ErrBadRequest = errors.New("bad request")

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PaginatedResponse struct {
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	Items    interface{} `json:"items"`
}

func (rd *Response) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func Success(w http.ResponseWriter, r *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	render.Status(r, http.StatusOK)
	render.JSON(w, r, data)
}

func PaginatedSuccess(w http.ResponseWriter, r *http.Request, items interface{}, total int, page int, pageSize int) {
	Success(w, r, &PaginatedResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	})
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
