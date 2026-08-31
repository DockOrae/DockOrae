// Agent Client Compose 同步执行(§11:面板管 YAML/项目,Agent 执行)。
package agent

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
)

// ComposeRunReq 同步 compose 执行请求
type ComposeRunReq struct {
	Project string            `json:"project"`
	Yaml    string            `json:"yaml"`
	Files   map[string]string `json:"files"`
}

// ComposeRun 同步执行 compose 动作(start/stop/restart/down/build/up/pull),返回 {ok, output}
func (c *Client) ComposeRun(ctx context.Context, project, yaml string, files map[string]string, args ...string) (ComposeRunResult, error) {
	action := ""
	if len(args) > 0 {
		action = args[0]
	}
	req := ComposeRunReq{Project: project, Yaml: yaml, Files: files}
	var path string
	switch action {
	case "start", "stop", "restart", "build":
		path = "/v1/compose/managed/" + action
	case "down":
		path = "/v1/compose/managed/down"
		for _, a := range args[1:] {
			if a == "-v" {
				path += "?volumes=1"
			}
		}
	case "up", "pull":
		// 同步 up/pull(应用商店安装/升级):动作与参数由 Agent 固定 allowlist
		payload := map[string]any{
			"project": project, "yaml": yaml, "files": files,
			"action": action,
		}
		mode := ""
		for _, a := range args[1:] {
			if a == "--force-recreate" {
				mode = "recreate"
			}
		}
		if mode != "" {
			payload["mode"] = mode
		}
		data, err := c.Call(ctx, http.MethodPost, "/v1/compose/managed/run", payload, "")
		if err != nil {
			return ComposeRunResult{}, err
		}
		var res ComposeRunResult
		raw, _ := json.Marshal(data)
		_ = json.Unmarshal(raw, &res)
		return res, nil
	default:
		return ComposeRunResult{}, &AgentError{Status: 400, Code: "INVALID_REQUEST", Message: "不支持的 compose 动作: " + action}
	}
	data, err := c.Call(ctx, http.MethodPost, path, req, "")
	if err != nil {
		return ComposeRunResult{}, err
	}
	var res ComposeRunResult
	raw, _ := json.Marshal(data)
	_ = json.Unmarshal(raw, &res)
	return res, nil
}

// ComposeLogsWS 拨号 compose 日志 WebSocket(project + tail)
func (c *Client) ComposeLogsWS(ctx context.Context, project, tail string) (*websocket.Conn, error) {
	return c.DialWS(ctx, "/v1/compose/managed/logs?project="+project+"&tail="+tail)
}
