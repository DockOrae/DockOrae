package api

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/DockOrae/DockOrae/internal/service"
)

// licenseEventsWS GET /ws/license — License 状态实时推送(Event-Driven → Vue 自动更新)。
//
// 链路:License Server → SSE → Docker Manager Verify → LicenseStateManager → /ws/license → Vue。
// 浏览器无需刷新/轮询/重新登录,管理员解绑后 Pro 徽标与许可证页面即时变化。
//
// 连接建立后立即推送一次当前完整状态,之后每次状态变更推送一次。
func licenseEventsWS(c *gin.Context, d *Deps) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()

	// 推送当前状态(立即一次,前端拿到初始值)
	push := func() {
		payload, _ := json.Marshal(map[string]any{"type": "license", "data": service.LicenseInfo(d.St)})
		_ = conn.WriteMessage(websocket.TextMessage, payload)
	}
	push()

	// 订阅状态变更(LicenseStateManager 通知)
	ch := service.SubscribeLicenseState(d.St)
	defer service.UnsubscribeLicenseState(d.St, ch)

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return nil
			}
			push()
		case <-pingTicker.C:
			if conn.WriteMessage(websocket.PingMessage, nil) != nil {
				return nil
			}
		}
	}
}
