// Package service 分层架构中的业务逻辑层:组合 docker/model/db/settings 等,
// 供 api 层调用;不依赖 gin(错误用 ApiError 表达,由 api 层映射为 HTTP 响应)。
package service

import (
	"errors"
	"log"

	cerrdefs "github.com/containerd/errdefs"
)

// ApiError 业务错误 → api 层转 {error: message} JSON
type ApiError struct {
	Status  int
	Message string
}

func (e *ApiError) Error() string { return e.Message }

func NewApiError(status int, message string) *ApiError {
	return &ApiError{Status: status, Message: message}
}

func BadRequest(message string) *ApiError {
	return NewApiError(400, message)
}

// DockerError 把 Docker SDK 错误映射为 HTTP 状态(等价旧版 bollard 映射)
func DockerError(err error) *ApiError {
	status := 502 // Bad Gateway
	switch {
	case cerrdefs.IsNotFound(err):
		status = 404
	case cerrdefs.IsConflict(err):
		status = 409
	case cerrdefs.IsPermissionDenied(err):
		status = 403
	case cerrdefs.IsInvalidArgument(err):
		status = 400
	}
	log.Printf("docker api error: %v", err)
	return NewApiError(status, err.Error())
}

// AsApiError 从 error 链中提取 ApiError;docker SDK 错误按 cerrdefs 映射;
// 其余普通错误统一按 500 处理
func AsApiError(err error) *ApiError {
	var ae *ApiError
	if errors.As(err, &ae) {
		return ae
	}
	switch {
	case cerrdefs.IsNotFound(err):
		return NewApiError(404, err.Error())
	case cerrdefs.IsConflict(err):
		return NewApiError(409, err.Error())
	case cerrdefs.IsPermissionDenied(err):
		return NewApiError(403, err.Error())
	case cerrdefs.IsInvalidArgument(err):
		return NewApiError(400, err.Error())
	}
	log.Printf("err.internal: %v", err)
	return NewApiError(500, "err.internalSimple")
}

// strOr 取 map 中字符串值,不存在给默认值
func strOr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
