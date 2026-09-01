package api

import (
	"context"
	"net/http"

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
// 读循环在独立 goroutine;ctx 取消时主动关闭连接解除 ReadMessage 阻塞,保证及时退出。
// 不用 SetReadDeadline+超时轮询:gorilla 会把读超时记入 readErr,后续 ReadMessage
// 不再阻塞直接返回同一错误,外层 continue 形成热循环,1000 次后 panic
// ("repeated read on failed websocket connection")。
func wsPump(ctx context.Context, conn *websocket.Conn, onMsg func(mt int, data []byte) bool) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if !onMsg(mt, data) {
				return
			}
		}
	}()
	select {
	case <-ctx.Done():
		_ = conn.Close()
		<-done
	case <-done:
	}
}
