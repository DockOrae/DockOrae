// Agent Client 裸 JSON 接口:宿主终端长轮询 + 宿主文件端点。
// 与 Call 的区别:Agent 端返回裸 JSON(非 {ok,data} 信封),透传状态码与响应体。
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RawResponse 裸响应(状态码 + 原始 body)
type RawResponse struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

// RawCall 调用 Agent 裸 JSON 端点(method/path 来自固定映射表)。
// query 为已编码的 query string(可空);payload 为请求体(可 nil);user 为面板用户名。
func (c *Client) RawCall(ctx context.Context, method, path, query string, payload any, user string) (*RawResponse, error) {
	full := path
	if query != "" {
		full += "?" + query
	}
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, &AgentError{Status: 400, Code: "INVALID_REQUEST", Message: "请求体无效"}
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://unix"+full, body)
	if err != nil {
		return nil, &AgentError{Status: 500, Code: "INTERNAL", Message: "构造 Agent 请求失败"}
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if user != "" {
		req.Header.Set("X-Agent-User", user)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, &AgentError{Status: 502, Code: "AGENT_UNAVAILABLE", Message: agentUnavailableMsg(c.SocketPath, err)}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 128<<20))
	if err != nil {
		return nil, &AgentError{Status: 502, Code: "AGENT_UNAVAILABLE", Message: "读取 Agent 响应失败"}
	}
	if resp.StatusCode >= 400 {
		// 尝试解析 Agent Problem 信封({type,title,status,code,detail,requestId})
		var problem struct {
			Code    string `json:"code"`
			Title   string `json:"title"`
			Detail  string `json:"detail"`
			Message string `json:"message"`
		}
		if json.Unmarshal(raw, &problem) == nil && (problem.Code != "" || problem.Message != "") {
			msg := problem.Message
			if msg == "" {
				msg = problem.Detail
			}
			if msg == "" {
				msg = problem.Title
			}
			if msg == "" {
				msg = fmt.Sprintf("Agent 请求失败(%d)", resp.StatusCode)
			}
			return nil, &AgentError{Status: resp.StatusCode, Code: problem.Code, Message: msg}
		}
		return nil, &AgentError{Status: resp.StatusCode, Code: "AGENT_ERROR", Message: fmt.Sprintf("Agent 请求失败(%d)", resp.StatusCode)}
	}
	return &RawResponse{StatusCode: resp.StatusCode, Body: raw, Header: resp.Header}, nil
}

// RawStream 流式响应(下载/上传/压缩,直接透传 body)
func (c *Client) RawStream(ctx context.Context, method, path, query string, body io.Reader, user string, header http.Header) (*http.Response, error) {
	full := path
	if query != "" {
		full += "?" + query
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://unix"+full, body)
	if err != nil {
		return nil, &AgentError{Status: 500, Code: "INTERNAL", Message: "构造 Agent 请求失败"}
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if user != "" {
		req.Header.Set("X-Agent-User", user)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, &AgentError{Status: 502, Code: "AGENT_UNAVAILABLE", Message: agentUnavailableMsg(c.SocketPath, err)}
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		var problem struct {
			Code  string `json:"code"`
			Title string `json:"title"`
		}
		if json.Unmarshal(raw, &problem) == nil && problem.Code != "" {
			return nil, &AgentError{Status: resp.StatusCode, Code: problem.Code, Message: problem.Title}
		}
		return nil, &AgentError{Status: resp.StatusCode, Code: "AGENT_ERROR", Message: "Agent 流请求失败"}
	}
	return resp, nil
}
