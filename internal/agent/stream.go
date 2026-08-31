// Agent Client 流式接口:镜像拉取 NDJSON 进度、compose 流式输出。
package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// StreamBody NDJSON 流式响应体
type StreamBody struct {
	Body io.ReadCloser
	Sc   *bufio.Scanner
}

// pullStream 拉取镜像(NDJSON 进度流)。
// Agent 返回 application/x-ndjson;错误信封(application/json)转为 AgentError。
func (c *Client) PullImageStream(ctx context.Context, ref string) (*StreamBody, error) {
	return c.ndjson(ctx, http.MethodPost, "/v1/docker/images/pull", map[string]any{"image": ref})
}

// ComposeStreamReq 流式 compose 执行请求
type ComposeStreamReq struct {
	Project string            `json:"project"`
	Yaml    string            `json:"yaml"`
	Files   map[string]string `json:"files"`
}

// ComposeUpStream 流式 compose up(NDJSON:每行 {"type":"line","data":...},结束 {"type":"done",...})
func (c *Client) ComposeUpStream(ctx context.Context, req ComposeStreamReq, args string) (*StreamBody, error) {
	path := "/v1/compose/managed/up"
	if args != "" {
		path += "?args=" + args
	}
	return c.ndjson(ctx, http.MethodPost, path, req)
}

// ComposePullStream 流式 compose pull
func (c *Client) ComposePullStream(ctx context.Context, req ComposeStreamReq) (*StreamBody, error) {
	return c.ndjson(ctx, http.MethodPost, "/v1/compose/managed/pull", req)
}

// ndjson 发起请求并校验响应类型,返回逐行 scanner
func (c *Client) ndjson(ctx context.Context, method, path string, payload any) (*StreamBody, error) {
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
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, &AgentError{Status: 502, Code: "AGENT_UNAVAILABLE", Message: agentUnavailableMsg(c.SocketPath, err)}
	}
	// Agent 错误信封:application/json
	if ct := resp.Header.Get("Content-Type"); ct != "" && !bytes.Contains([]byte(ct), []byte("ndjson")) {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		var env struct {
			OK    bool `json:"ok"`
			Error *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(raw, &env) == nil && env.Error != nil {
			return nil, &AgentError{Status: resp.StatusCode, Code: env.Error.Code, Message: env.Error.Message}
		}
		return nil, &AgentError{Status: resp.StatusCode, Code: "AGENT_ERROR", Message: string(raw)}
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 4<<20)
	return &StreamBody{Body: resp.Body, Sc: sc}, nil
}

// Next 读取下一行 NDJSON;io.EOF = 流结束
func (s *StreamBody) Next() (json.RawMessage, error) {
	if !s.Sc.Scan() {
		if err := s.Sc.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	return json.RawMessage(s.Sc.Bytes()), nil
}

// Close 关闭流
func (s *StreamBody) Close() error {
	if s.Body != nil {
		return s.Body.Close()
	}
	return nil
}

var _ = errors.Is
