// Package errcode 提供统一错误码注册表与业务错误类型。
//
// 设计原则(参照成熟后端实践):
//   - 所有对外错误码集中注册,携带 HTTP 状态与默认中文文案;
//   - 业务层(application)返回 *Error,禁止裸 fmt.Errorf 冒泡到 handler;
//   - handler 用 response 包一行直出,不手搓 c.JSON。
package errcode

import (
	"fmt"
	"net/http"
	"sync"
)

// Code 业务错误码
type Code int

// Meta 错误码元信息
type Meta struct {
	HTTPStatus int
	Message    string
}

var (
	registry = make(map[Code]Meta)
	mu       sync.RWMutex
)

// Register 注册错误码。重复注册会 panic,便于启动期暴露冲突。
func Register(code Code, httpStatus int, message string) Code {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[code]; exists {
		panic(fmt.Sprintf("errcode: 重复注册错误码 %d", code))
	}
	registry[code] = Meta{HTTPStatus: httpStatus, Message: message}
	return code
}

// Lookup 查询错误码元信息
func Lookup(code Code) (Meta, bool) {
	mu.RLock()
	defer mu.RUnlock()
	m, ok := registry[code]
	return m, ok
}

// MustLookup 查询错误码,不存在返回 CommonInternal 的元信息
func MustLookup(code Code) Meta {
	if m, ok := Lookup(code); ok {
		return m
	}
	m, _ := Lookup(CommonInternal)
	return m
}

// Error 业务错误,携带错误码与可选的底层 cause(cause 只进日志,不对外暴露)。
type Error struct {
	Code  Code
	Msg   string // 可选:覆盖注册表默认文案
	Cause error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("errcode(%d): %v", e.Code, e.Cause)
	}
	return fmt.Sprintf("errcode(%d)", e.Code)
}

func (e *Error) Unwrap() error { return e.Cause }

// New 构造业务错误
func New(code Code, cause error) *Error {
	return &Error{Code: code, Cause: cause}
}

// NewMsg 构造业务错误并覆盖默认文案
func NewMsg(code Code, msg string, cause error) *Error {
	return &Error{Code: code, Msg: msg, Cause: cause}
}

// === 通用错误码(1000 段)===
var (
	CommonInternal      = Register(1000, http.StatusInternalServerError, "服务器内部错误")
	CommonInvalidParam  = Register(1001, http.StatusBadRequest, "请求参数错误")
	CommonUnauthorized  = Register(1002, http.StatusUnauthorized, "未登录或登录已过期")
	CommonTokenMissing  = Register(1003, http.StatusUnauthorized, "缺少认证令牌")
	CommonForbidden     = Register(1004, http.StatusForbidden, "无权限访问")
	CommonRouteNotFound = Register(1005, http.StatusNotFound, "路由不存在")
	CommonTooManyReq    = Register(1006, http.StatusTooManyRequests, "请求过于频繁,请稍后重试")
)
