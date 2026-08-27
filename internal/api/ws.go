package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// upgradeWS 升级为 WebSocket 连接
func upgradeWS(c *gin.Context) (*websocket.Conn, error) {
	return upgrader.Upgrade(c.Writer, c.Request, nil)
}

// wsPump 读取 WS 消息直到连接关闭或 onMsg 返回 false。
// 内部用短读超时轮询 ctx 取消(如日志流结束、SSH 会话退出),保证及时退出。
func wsPump(ctx context.Context, conn *websocket.Conn, onMsg func(mt int, data []byte) bool) {
	for {
		if ctx.Err() != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		mt, data, err := conn.ReadMessage()
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return
		}
		if !onMsg(mt, data) {
			return
		}
	}
}
