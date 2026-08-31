// Package agent DockOrae 主程序侧的 Agent 客户端。
// 经 Unix Socket 调用 DockOrae-Agent(宿主机控制平面),遵循 §53:
// Frontend → DockOrae → Agent(面板是唯一调用方,Agent 不直接暴露给前端)。
// 本包不依赖 internal/service(避免 import cycle),错误以 AgentError 表达,由 api 层转换。
package agent

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// DefaultSocket 默认 Agent socket 路径
const DefaultSocket = "/run/dockorae/agent.sock"

// DefaultTokenFile 共享 token 文件(面板写入,Agent 读取)
const DefaultTokenFile = "/run/dockorae/agent.token"

// Client Agent 客户端
type Client struct {
	SocketPath string
	Token      string
	HTTP       *http.Client
}

// New 构造客户端
func New(socketPath, token string) *Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return &Client{
		SocketPath: socketPath,
		Token:      token,
		HTTP: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return dialer.DialContext(ctx, "unix", socketPath)
				},
			},
			Timeout: 120 * time.Second,
		},
	}
}

// SocketExists 检查 socket 文件是否存在(Agent 是否已部署)
func SocketExists(path string) bool {
	if path == "" {
		path = DefaultSocket
	}
	st, err := os.Stat(path)
	return err == nil && st.Mode()&os.ModeSocket != 0
}

// GenerateToken 生成 32 字节随机 token(十六进制)。
// crypto/rand 失败时直接 panic:安全凭据生成失败,绝不以弱 token(零字节)启动。
func GenerateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// WriteTokenFile 把 token 写入共享目录(Agent 启动时读取)
func WriteTokenFile(token string) error {
	dir := "/run/dockorae"
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(DefaultTokenFile, []byte(token), 0o640)
}

// AgentError Agent 调用错误(Status + Code + Message;api 层转面板 ApiError)
type AgentError struct {
	Status  int
	Code    string
	Message string
}

func (e *AgentError) Error() string { return fmt.Sprintf("[%s] %s", e.Code, e.Message) }

// Call 调用 Agent 结构化 API(method/path 均来自固定映射表,禁止任意透传)。
// payload 为请求体(可 nil);user 为面板已认证用户名(透传审计)。
// 返回 Agent 响应的 data 字段;错误以 *AgentError 返回。
func (c *Client) Call(ctx context.Context, method, path string, payload any, user string) (map[string]any, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, &AgentError{Status: 400, Code: "INVALID_REQUEST", Message: "请求体无效"}
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://unix"+path, body)
	if err != nil {
		return nil, &AgentError{Status: 500, Code: "INTERNAL", Message: "构造 Agent 请求失败"}
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	if user != "" {
		req.Header.Set("X-Agent-User", user)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, &AgentError{Status: 502, Code: "AGENT_UNAVAILABLE", Message: agentUnavailableMsg(c.SocketPath, err)}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, &AgentError{Status: 502, Code: "AGENT_UNAVAILABLE", Message: "读取 Agent 响应失败"}
	}
	var env struct {
		OK    bool           `json:"ok"`
		Data  map[string]any `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, &AgentError{Status: 502, Code: "AGENT_UNAVAILABLE", Message: "Agent 响应格式无效"}
	}
	if !env.OK || env.Error != nil {
		if env.Error != nil {
			return nil, &AgentError{Status: statusForCode(env.Error.Code), Code: env.Error.Code, Message: env.Error.Message}
		}
		return nil, &AgentError{Status: resp.StatusCode, Code: "AGENT_ERROR", Message: "Agent 操作失败"}
	}
	if env.Data == nil {
		env.Data = map[string]any{}
	}
	return env.Data, nil
}

// statusForCode Agent 错误码 → HTTP 状态(与 Agent errs.StatusFor 对齐的镜像)
func statusForCode(code string) int {
	switch code {
	case "INVALID_REQUEST", "SWAP_INVALID_SIZE", "INVALID_CONFIRM", "PATH_INVALID", "FILE_TOO_LARGE", "UNSUPPORTED_ARCH":
		return 400
	case "UNAUTHORIZED":
		return 401
	case "PERMISSION_DENIED", "DANGEROUS_PATH":
		return 403
	case "NOT_FOUND", "COMPOSE_PROJECT_NOT_FOUND", "FILE_NOT_FOUND":
		return 404
	case "CONFLICT", "OPERATION_IN_PROGRESS", "FILE_EXISTS":
		return 409
	case "AGENT_UNAVAILABLE", "DOCKER_UNAVAILABLE", "PTY_UNAVAILABLE", "TERMINAL_SESSION":
		return 502
	default:
		return 500
	}
}

func agentUnavailableMsg(socket string, err error) string {
	return fmt.Sprintf("Agent 不可用(socket: %s): %v", socket, err)
}
