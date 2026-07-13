// Package web 通过 go:embed 内嵌前端静态页,编译进单二进制,免额外前端构建。
package web

import _ "embed"

// LoginHTML 登录页
//
//go:embed login.html
var LoginHTML []byte

// AdminHTML 管理页
//
//go:embed admin.html
var AdminHTML []byte
