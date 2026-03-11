package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 业务错误码，0 表示成功，其余表示失败。
// 约定：客户端错误以 4 开头，服务端错误以 5 开头，与 HTTP status 保持直觉对应。
const (
	CodeOK            = 0
	CodeBadRequest    = 400
	CodeUnauthorized  = 401
	CodeForbidden     = 403
	CodeNotFound      = 404
	CodeInternalError = 500
)

// Response 是所有 JSON API 的统一响应体。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// OK 返回成功响应（HTTP 200）。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: CodeOK, Message: "ok", Data: data})
}

// Fail 返回自定义业务错误响应，code 为业务错误码。
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Response{Code: code, Message: msg})
}

// Created 返回创建成功响应（HTTP 201），携带新建资源数据。
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{Code: CodeOK, Message: "ok", Data: data})
}

// BadRequest 返回客户端参数错误响应（HTTP 400）。
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Response{Code: CodeBadRequest, Message: msg})
}

// NotFound 返回资源不存在响应（HTTP 404）。
func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, Response{Code: CodeNotFound, Message: msg})
}

// InternalError 返回服务端错误响应（HTTP 500）。
func InternalError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, Response{Code: CodeInternalError, Message: msg})
}

