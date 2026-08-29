package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestWSDisconnectReleasesWriter 回归测试(GO-001):
// 复刻 containersLogsWS/StatsWS/TerminalWS 的并发模式 —— writer goroutine 阻塞读
// Docker 流,wsPump 在客户端断开时返回。修复前顺序为「wg.Wait() → handler return →
// defer cancel()」,客户端断开 + 流静默(无新数据触发写失败)时会永久死锁。
// 修复后顺序必须为「wsPump 返回 → cancel()/close → wg.Wait()」。
func TestWSDisconnectReleasesWriter(t *testing.T) {
	up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	handlerDone := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerDone)
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// 模拟 docker 日志/统计流:静默(不产生任何数据),只有 close 才会解除阻塞
		pr, _ := io.Pipe()

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = io.Copy(io.Discard, pr) // 阻塞读;流被 close 或 ctx 取消后才退出
		}()

		wsPump(ctx, conn, func(mt int, data []byte) bool {
			return mt != websocket.CloseMessage
		})

		// 修复后的关键顺序:先取消 ctx 并关闭流,再等待 writer 退出
		cancel()
		pr.Close()

		waitCh := make(chan struct{})
		go func() { wg.Wait(); close(waitCh) }()
		select {
		case <-waitCh:
			// writer goroutine 已退出,无泄漏
		case <-time.After(3 * time.Second):
			t.Error("writer goroutine 未退出:取消顺序错误导致 goroutine 泄漏")
		}
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	// 多次连接/断开:修复前每次断开都会累积泄漏,修复后全部正常退出
	for i := 0; i < 5; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		// 正常关闭(close frame)
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = conn.Close()
	}

	// 等待最后一次 handler 完全退出(泄漏时这里 3s 超时)
	select {
	case <-handlerDone:
	case <-time.After(3 * time.Second):
		t.Fatal("handler 未返回:存在 goroutine 泄漏")
	}
}
