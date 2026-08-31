package api

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/DockOrae/DockOrae/internal/agent"
	"github.com/DockOrae/DockOrae/internal/service"
)

// ============================================================
// Agent(宿主机控制平面)代理 API
// §53:Frontend → DockOrae → Agent。面板是唯一调用方。
// 安全:仅允许下方固定映射表内的端点,禁止任意路径透传(§6)。
// ============================================================

// agentGetEndpoints 面板 GET 路径 → Agent 路径(固定映射)
var agentGetEndpoints = map[string]string{
	"/agent/status":                 "/v1/health",
	"/agent/swap":                   "/v1/swap/status",
	"/agent/host/info":              "/v1/host/info",
	"/agent/host/hostname":          "/v1/host/hostname",
	"/agent/system/info":            "/v1/system/info",
	"/agent/system/timezone":        "/v1/system/timezone",
	"/agent/system/time":            "/v1/system/time",
	"/agent/system/update/check":    "/v1/system/update/check",
	"/agent/docker/status":          "/v1/docker/status",
	"/agent/docker/info":            "/v1/docker/info",
	"/agent/docker/version":         "/v1/docker/version",
	"/agent/docker/cleanup/preview": "/v1/docker/cleanup/preview",
	"/agent/compose/projects":       "/v1/compose/projects",
	"/agent/compose/status":         "/v1/compose/status",
	"/agent/compose/check_update":   "/v1/compose/check_update",
	"/agent/compose/history":        "/v1/compose/history",
	"/agent/binary/status":          "/v1/binary/status",
	"/agent/disk/usage":             "/v1/disk/usage",
	"/agent/disk/devices":           "/v1/disk/devices",
	"/agent/disk/mounts":            "/v1/disk/mounts",
	"/agent/sysctl/get":             "/v1/sysctl/get",
	"/agent/network/interfaces":     "/v1/network/interfaces",
	"/agent/network/routes":         "/v1/network/routes",
	"/agent/network/dns":            "/v1/network/dns",
}

// agentPostEndpoints 面板 POST 路径 → Agent 路径(固定映射)
var agentPostEndpoints = map[string]string{
	"/agent/host/hostname":    "/v1/host/hostname",
	"/agent/host/reboot":      "/v1/host/reboot",
	"/agent/system/timezone":  "/v1/system/timezone",
	"/agent/system/time/sync": "/v1/system/time/sync",
	"/agent/system/service":   "/v1/system/service",
	"/agent/system/update":    "/v1/system/update",
	"/agent/docker/service":   "/v1/docker/service",
	"/agent/docker/cleanup":   "/v1/docker/cleanup",
	"/agent/compose/pull":     "/v1/compose/pull",
	"/agent/compose/update":   "/v1/compose/update",
	"/agent/compose/rollback": "/v1/compose/rollback",
	"/agent/binary/check":     "/v1/binary/check_update",
	"/agent/binary/download":  "/v1/binary/download",
	"/agent/binary/install":   "/v1/binary/install",
	"/agent/binary/rollback":  "/v1/binary/rollback",
	"/agent/sysctl/set":       "/v1/sysctl/set",
}

// agentProxy GET 代理:透传 query string(如 project=xxx);直接返回 Agent 的 data(与面板 API 风格一致)
func agentProxyGet(panelPath string) H {
	return func(c *gin.Context, d *Deps) error {
		target := agentGetEndpoints[panelPath]
		if target == "" {
			return service.NewApiError(404, "agent endpoint not found")
		}
		if q := c.Request.URL.RawQuery; q != "" {
			target += "?" + q
		}
		data, err := d.St.Agent.Call(c.Request.Context(), http.MethodGet, target, nil, c.GetString("username"))
		if err != nil {
			return agentErr(err)
		}
		c.JSON(200, data)
		return nil
	}
}

// agentProxyPost POST 代理:透传请求体(confirm/size_mb 等);直接返回 Agent 的 data
func agentProxyPost(panelPath string) H {
	return func(c *gin.Context, d *Deps) error {
		target := agentPostEndpoints[panelPath]
		if target == "" {
			return service.NewApiError(404, "agent endpoint not found")
		}
		if q := c.Request.URL.RawQuery; q != "" {
			target += "?" + q
		}
		var payload any
		if c.Request.Body != nil {
			raw, _ := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
			if len(raw) > 0 && string(raw) != "null" {
				if err := jsonUnmarshal(raw, &payload); err != nil {
					return service.BadRequest("err.requestFailed")
				}
			}
		}
		data, err := d.St.Agent.Call(c.Request.Context(), http.MethodPost, target, payload, c.GetString("username"))
		if err != nil {
			return agentErr(err)
		}
		c.JSON(200, data)
		return nil
	}
}

// agentSwap Swap 操作路由:action 字段决定 create/resize/delete(§11-§17)
func agentSwap(c *gin.Context, d *Deps) error {
	var req struct {
		Action  string `json:"action"`
		SizeMB  int    `json:"size_mb"`
		Path    string `json:"path"`
		Confirm *bool  `json:"confirm"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	var target string
	switch req.Action {
	case "create":
		target = "/v1/swap/create"
	case "resize":
		target = "/v1/swap/resize"
	case "delete":
		target = "/v1/swap/delete"
	default:
		return service.BadRequest("agent.swap.invalidAction")
	}
	data, err := d.St.Agent.Call(c.Request.Context(), http.MethodPost, target, req, c.GetString("username"))
	if err != nil {
		return agentErr(err)
	}
	c.JSON(200, data)
	return nil
}

// agentErr Agent 错误 → 面板 ApiError(透传状态码与消息)
func agentErr(err error) error {
	var ae *agent.AgentError
	if errors.As(err, &ae) {
		return service.NewApiError(ae.Status, ae.Message)
	}
	return err
}
