package api

import (
	"errors"
	"log"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// ApiError 处理函数返回的错误 → {error: message} JSON
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

// dockerError 把 Docker SDK 错误映射为 HTTP 状态(等价旧版 bollard 映射)
func dockerError(err error) *ApiError {
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

// H 适配器:让 handler 可以返回 error,统一转 JSON(注入 AppState)
type H func(c *gin.Context, st *state.AppState) error

func (h H) Handler(st *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h(c, st); err != nil {
			ae := toApiError(err)
			c.AbortWithStatusJSON(ae.Status, gin.H{"error": ae.Message})
		}
	}
}

func toApiError(err error) *ApiError {
	var ae *ApiError
	if errors.As(err, &ae) {
		return ae
	}
	log.Printf("err.internal: %v", err)
	return NewApiError(500, "err.internalSimple")
}
