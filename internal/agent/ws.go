// Agent Client WebSocket 拨号(unix socket + Bearer 认证)。
// 供容器日志/统计/终端、compose 日志、docker 事件透传使用。
package agent

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// DialWS 拨号 Agent WebSocket 端点
func (c *Client) DialWS(ctx context.Context, path string) (*websocket.Conn, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	d := websocket.Dialer{
		NetDialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", c.SocketPath)
		},
		HandshakeTimeout: 10 * time.Second,
	}
	header := http.Header{}
	header.Set("Authorization", "Bearer "+c.Token)
	conn, _, err := d.DialContext(ctx, "ws://unix"+path, header)
	if err != nil {
		return nil, &AgentError{Status: 502, Code: "AGENT_UNAVAILABLE", Message: agentUnavailableMsg(c.SocketPath, err)}
	}
	return conn, nil
}

// ContainerLogsWS 容器日志 WebSocket(id + tail)
func (c *Client) ContainerLogsWS(ctx context.Context, id, tail string) (*websocket.Conn, error) {
	return c.DialWS(ctx, "/v1/docker/containers/"+id+"/logs?tail="+tail)
}

// ContainerStatsWS 容器 stats WebSocket(id)
func (c *Client) ContainerStatsWS(ctx context.Context, id string) (*websocket.Conn, error) {
	return c.DialWS(ctx, "/v1/docker/containers/"+id+"/stats")
}

// ContainerTerminalWS 容器终端 WebSocket(id + shell)
func (c *Client) ContainerTerminalWS(ctx context.Context, id, shell string) (*websocket.Conn, error) {
	return c.DialWS(ctx, "/v1/docker/containers/"+id+"/terminal?shell="+shell)
}

// DockerEventsWS docker 事件 WebSocket(每帧一条 JSON TEXT 消息)
func (c *Client) DockerEventsWS(ctx context.Context) (*websocket.Conn, error) {
	return c.DialWS(ctx, "/v1/docker/events")
}
