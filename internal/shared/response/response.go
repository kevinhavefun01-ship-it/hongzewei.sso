// Package response 提供统一 API 响应结构与错误输出入口。
// 所有 handler 必须经由此包输出,禁止手搓 c.JSON(302 重定向、健康检查等协议场景除外)。
package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"hongzewei.sso/internal/shared/contextx"
	"hongzewei.sso/internal/shared/errcode"

	"github.com/gin-gonic/gin"
)

// R 统一响应结构
type R struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

// Pagination 分页响应数据
type Pagination struct {
	List     any   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// Success HTTP 200 + code=1
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, R{
		Code:    1,
		Message: "success",
		Data:    data,
		TraceID: contextx.TraceID(c.Request.Context()),
	})
}

// SuccessPage 分页成功响应
func SuccessPage(c *gin.Context, list any, total int64, page, pageSize int) {
	Success(c, Pagination{List: list, Total: total, Page: page, PageSize: pageSize})
}

// Fail 按已注册错误码输出。未注册时降级为 CommonInternal。
// cause 只写入 gin.Errors(进日志),不对外暴露。
func Fail(c *gin.Context, code errcode.Code, cause ...error) {
	meta, ok := errcode.Lookup(code)
	if !ok {
		_ = c.Error(fmt.Errorf("response.Fail: 错误码 %d 未注册", code))
		code = errcode.CommonInternal
		meta = errcode.MustLookup(code)
	}
	writeErrors(c, cause...)
	c.JSON(meta.HTTPStatus, R{
		Code:    int(code),
		Message: meta.Message,
		TraceID: contextx.TraceID(c.Request.Context()),
	})
}

// FailMsg 与 Fail 等价,但用指定 message 覆盖默认文案(用于动态拼接字段名等场景)。
func FailMsg(c *gin.Context, code errcode.Code, message string, cause ...error) {
	meta, ok := errcode.Lookup(code)
	if !ok {
		code = errcode.CommonInternal
		meta = errcode.MustLookup(code)
		message = meta.Message
	}
	writeErrors(c, cause...)
	c.JSON(meta.HTTPStatus, R{
		Code:    int(code),
		Message: message,
		TraceID: contextx.TraceID(c.Request.Context()),
	})
}

// FailErr 当 err 为 *errcode.Error 时按其 Code 输出,否则降级 CommonInternal。
func FailErr(c *gin.Context, err error) {
	if err == nil {
		return
	}
	var e *errcode.Error
	if errors.As(err, &e) {
		if e.Msg != "" {
			FailMsg(c, e.Code, e.Msg, e.Cause)
		} else {
			Fail(c, e.Code, e.Cause)
		}
		return
	}
	Fail(c, errcode.CommonInternal, err)
}

// BindError 处理 gin 绑定错误,翻译成友好中文后按 CommonInvalidParam 输出。
func BindError(c *gin.Context, err error) {
	FailMsg(c, errcode.CommonInvalidParam, friendlyBindMsg(err), err)
}

func friendlyBindMsg(err error) string {
	if err == nil {
		return "请求参数错误"
	}
	if errors.Is(err, io.EOF) {
		return "请求体不能为空,请提供 JSON Body"
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Sprintf("请求体 JSON 格式错误(偏移量 %d)", syntaxErr.Offset)
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return fmt.Sprintf("字段 %q 类型不匹配,期望 %s", typeErr.Field, typeErr.Type.String())
	}
	return "请求参数错误: " + err.Error()
}

func writeErrors(c *gin.Context, cause ...error) {
	for _, e := range cause {
		if e != nil {
			_ = c.Error(e)
		}
	}
}
