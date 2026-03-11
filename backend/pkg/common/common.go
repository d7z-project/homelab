package common

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
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
	parts := strings.Split(module, "/")
	ns := append([]string{"system", "sync", "version"}, parts[:len(parts)-1]...)
	key := parts[len(parts)-1]
	_ = DB.Child(ns...).Put(ctx, key, strconv.FormatInt(version, 10), kv.TTLKeep)
	return version
}

// GetGlobalVersion 获取指定模块的全局版本号
func GetGlobalVersion(ctx context.Context, module string) int64 {
	parts := strings.Split(module, "/")
	ns := append([]string{"system", "sync", "version"}, parts[:len(parts)-1]...)
	key := parts[len(parts)-1]
	val, err := DB.Child(ns...).Get(ctx, key)
	if err != nil || val == "" {
		return 0
	}
	v, _ := strconv.ParseInt(val, 10, 64)
	return v
}

// LockWithTimeout 尝试获取分布式锁，带重试和超时机制
func LockWithTimeout(ctx context.Context, lockKey string, timeout time.Duration) (func(), error) {
	if Locker == nil {
		return func() {}, nil // 单机模式或未配置锁，跳过逻辑
	}
	start := time.Now()
	for {
		release := Locker.TryLock(ctx, lockKey)
		if release != nil {
			return release, nil
		}
		if timeout > 0 && time.Since(start) >= timeout {
			return nil, fmt.Errorf("lock timeout: %s", lockKey)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// WithJitter 执行带随机抖动的函数，用于平滑 I/O 惊群
func WithJitter(fn func(), maxDelay time.Duration) {
	delay := time.Duration(rand.Int63n(int64(maxDelay)))
	time.AfterFunc(delay, fn)
}

// IsComment 检查一行文本是否为注释行（以 #, ;, //, ! 开头）
func IsComment(line string) bool {
	line = strings.TrimSpace(line)
	return line == "" ||
		strings.HasPrefix(line, "#") ||
		strings.HasPrefix(line, ";") ||
		strings.HasPrefix(line, "//") ||
		strings.HasPrefix(line, "!")
}

var ErrNotFound = errors.New("resource not found")
var ErrBadRequest = errors.New("bad request")
var ErrConflict = errors.New("resource conflict")
var ErrInvalidConfig = errors.New("invalid configuration")

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type CursorResponse struct {
	Total      int         `json:"total"`
	NextCursor string      `json:"nextCursor,omitempty"`
	HasMore    bool        `json:"hasMore"`
	Items      interface{} `json:"items"`
}

func (rd *Response) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}

func Success(w http.ResponseWriter, r *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	render.Status(r, http.StatusOK)
	render.JSON(w, r, data)
}

type CursorProvider interface {
	GetItems() interface{}
	GetTotal() int64
	GetCursor() string
	HasMoreData() bool
}

func CursorSuccess(w http.ResponseWriter, r *http.Request, provider CursorProvider) {
	Success(w, r, &CursorResponse{
		Total:      int(provider.GetTotal()),
		NextCursor: provider.GetCursor(),
		HasMore:    provider.HasMoreData(),
		Items:      provider.GetItems(),
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

// ValidateURL 简单的 SSRF 防护校验
func ValidateURL(urlStr string, allowPrivate bool) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	hostname := u.Hostname()
	if hostname == "localhost" && !allowPrivate {
		return fmt.Errorf("SSRF detected: localhost is forbidden")
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		// 如果解析失败，可能是 IP 直连或者非法的
		ip := net.ParseIP(hostname)
		if ip != nil {
			if isPrivateIP(ip) && !allowPrivate {
				return fmt.Errorf("SSRF detected: private IP %s is forbidden", ip)
			}
			return nil
		}
		return nil
	}

	for _, ip := range ips {
		if isPrivateIP(ip) && !allowPrivate {
			return fmt.Errorf("SSRF detected: host %s resolves to private IP %s", hostname, ip)
		}
	}

	return nil
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}
	return false
}
