// 宿主终端长轮询 API(自 KPanel internal/panel/terminal.go 移植,2026-09-02)。
// 端点(KPanel 同款契约):
//
//	POST /api/v1/terminal-sessions              → 打开会话 {rows, columns} → {sessionId, offset, ...}
//	GET  /api/v1/terminal-sessions/{id}/output  → 长轮询输出 ?offset=&wait= → {data, nextOffset, ...}
//	POST /api/v1/terminal-sessions/{id}/input   → 输入 {data}
//	POST /api/v1/terminal-sessions/{id}/resize  → 调整尺寸 {rows, columns}
//	POST /api/v1/terminal-sessions/{id}/close   → 关闭会话
//
// 面板仅做会话簿记(publicID ↔ backendID)+ 用户归属;数据经 Agent 长轮询透传。
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DockOrae/DockOrae/internal/agent"
	"github.com/DockOrae/DockOrae/internal/service"
)

const (
	maxPanelTerminals       = 16
	maxPanelTerminalsByUser = 4
	panelTerminalIdleTTL    = 35 * time.Minute
)

// panelTerminalSession 面板侧终端会话(publicID 对前端;backendID 对 Agent)
type panelTerminalSession struct {
	ID        string
	BackendID string
	UserID    string
	Owner     string
	UpdatedAt time.Time
}

// terminalMu 会话表锁(单机面板,无集群)
var (
	terminalMu       sync.Mutex
	terminalSessions = map[string]panelTerminalSession{}
)

// terminalOpenRequest 打开请求
type terminalOpenRequest struct {
	Rows    uint16 `json:"rows"`
	Columns uint16 `json:"columns"`
	Cwd     string `json:"cwd,omitempty"`
}

// terminalOpenResponse 打开响应
type terminalOpenResponse struct {
	SessionID string    `json:"sessionId"`
	Offset    int64     `json:"offset"`
	CreatedAt time.Time `json:"createdAt"`
}

// terminalSessionsOpen POST /api/v1/terminal-sessions
func terminalSessionsOpen(c *gin.Context, d *Deps) error {
	if c.Request.URL.RawQuery != "" {
		return service.BadRequest("invalid_terminal_request")
	}
	var input terminalOpenRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if input.Rows == 0 || input.Columns == 0 || input.Rows > 500 || input.Columns > 1000 {
		return service.BadRequest("invalid_terminal_dimensions")
	}
	userID := c.GetString("username")
	if userID == "" {
		userID = "panel"
	}

	stale := pruneTerminalSessions(time.Now().UTC().Add(-panelTerminalIdleTTL))
	for _, item := range stale {
		_ = closeTerminalBackend(c.Request.Context(), d, item)
	}
	if !reserveTerminalOpen(userID) {
		return service.NewApiError(http.StatusTooManyRequests, "终端会话数已达上限")
	}
	defer releaseTerminalOpen(userID)

	owner := "panel:" + userID
	// 打开 Agent 侧会话
	agentBody := map[string]any{
		"owner": owner, "rows": input.Rows, "columns": input.Columns,
	}
	if strings.TrimSpace(input.Cwd) != "" {
		agentBody["cwd"] = input.Cwd
	}
	resp, err := d.St.Agent.RawCall(c.Request.Context(), http.MethodPost, "/v1/host/terminal", "", agentBody, userID)
	if err != nil {
		return agentErr(err)
	}
	var snapshot struct {
		ID        string    `json:"id"`
		Offset    int64     `json:"offset"`
		CreatedAt time.Time `json:"createdAt"`
	}
	if err := jsonUnmarshal(resp.Body, &snapshot); err != nil || snapshot.ID == "" {
		return service.NewApiError(http.StatusBadGateway, "终端会话响应无效")
	}

	publicID := randomHex(32)
	terminalMu.Lock()
	terminalSessions[publicID] = panelTerminalSession{
		ID: publicID, BackendID: snapshot.ID, UserID: userID, Owner: owner, UpdatedAt: time.Now().UTC(),
	}
	terminalMu.Unlock()
	c.JSON(http.StatusCreated, terminalOpenResponse{
		SessionID: publicID, Offset: snapshot.Offset, CreatedAt: snapshot.CreatedAt,
	})
	return nil
}

// terminalSessionsOperation 处理 /api/v1/terminal-sessions/{id}/{action}
func terminalSessionsOperation(action string) H {
	return func(c *gin.Context, d *Deps) error {
		id := c.Param("id")
		userID := c.GetString("username")
		if userID == "" {
			userID = "panel"
		}
		if action != "output" && c.Request.URL.RawQuery != "" {
			return service.BadRequest("invalid_terminal_request")
		}
		terminalMu.Lock()
		item, ok := terminalSessions[id]
		if ok && item.UserID == userID {
			item.UpdatedAt = time.Now().UTC()
			terminalSessions[id] = item
		} else {
			ok = false
		}
		terminalMu.Unlock()
		if !ok {
			return service.NewApiError(http.StatusNotFound, "终端会话不存在")
		}

		backendPath := "/v1/host/terminal/" + url.PathEscape(item.BackendID)
		switch action {
		case "output":
			if c.Request.Method != http.MethodGet {
				return service.NewApiError(http.StatusMethodNotAllowed, "请求方法不允许")
			}
			offset, waitMS, valid := terminalOutputQuery(c)
			if !valid {
				return service.BadRequest("invalid_terminal_query")
			}
			query := url.Values{}
			query.Set("owner", item.Owner)
			query.Set("offset", strconv.FormatInt(offset, 10))
			query.Set("wait", strconv.Itoa(waitMS))
			resp, err := d.St.Agent.RawCall(c.Request.Context(), http.MethodGet, backendPath+"/output", query.Encode(), nil, userID)
			if err != nil {
				if isTerminalGone(err) {
					deleteTerminalSession(id)
					return service.NewApiError(http.StatusNotFound, "终端会话不存在")
				}
				return agentErr(err)
			}
			// 检测已关闭 → 清理会话表
			var out struct {
				Closed   bool       `json:"closed"`
				ExitedAt *time.Time `json:"exitedAt"`
			}
			_ = jsonUnmarshal(resp.Body, &out)
			if out.Closed || out.ExitedAt != nil {
				deleteTerminalSession(id)
			}
			c.Data(http.StatusOK, "application/json", resp.Body)
			return nil
		case "input":
			if c.Request.Method != http.MethodPost {
				return service.NewApiError(http.StatusMethodNotAllowed, "请求方法不允许")
			}
			var input struct {
				Data string `json:"data"`
			}
			if err := c.ShouldBindJSON(&input); err != nil {
				return service.BadRequest("err.requestFailed")
			}
			if input.Data == "" {
				return service.BadRequest("invalid_terminal_input")
			}
			if _, err := d.St.Agent.RawCall(c.Request.Context(), http.MethodPost, backendPath+"/input", "", map[string]any{
				"owner": item.Owner, "data": input.Data,
			}, userID); err != nil {
				if isTerminalGone(err) {
					deleteTerminalSession(id)
					return service.NewApiError(http.StatusNotFound, "终端会话不存在")
				}
				return agentErr(err)
			}
			c.JSON(http.StatusOK, gin.H{"accepted": true})
			return nil
		case "resize":
			if c.Request.Method != http.MethodPost {
				return service.NewApiError(http.StatusMethodNotAllowed, "请求方法不允许")
			}
			var input struct {
				Rows    uint16 `json:"rows"`
				Columns uint16 `json:"columns"`
			}
			if err := c.ShouldBindJSON(&input); err != nil {
				return service.BadRequest("err.requestFailed")
			}
			if input.Rows == 0 || input.Columns == 0 || input.Rows > 500 || input.Columns > 1000 {
				return service.BadRequest("invalid_terminal_dimensions")
			}
			if _, err := d.St.Agent.RawCall(c.Request.Context(), http.MethodPost, backendPath+"/resize", "", map[string]any{
				"owner": item.Owner, "rows": input.Rows, "columns": input.Columns,
			}, userID); err != nil {
				if isTerminalGone(err) {
					deleteTerminalSession(id)
					return service.NewApiError(http.StatusNotFound, "终端会话不存在")
				}
				return agentErr(err)
			}
			c.JSON(http.StatusOK, gin.H{"accepted": true})
			return nil
		case "close":
			if c.Request.Method != http.MethodPost {
				return service.NewApiError(http.StatusMethodNotAllowed, "请求方法不允许")
			}
			_ = closeTerminalBackend(c.Request.Context(), d, item)
			deleteTerminalSession(id)
			c.JSON(http.StatusOK, gin.H{"closed": true})
			return nil
		default:
			return service.NewApiError(http.StatusNotFound, "路由不存在")
		}
	}
}

func terminalOutputQuery(c *gin.Context) (int64, int, bool) {
	query := c.Request.URL.Query()
	if len(query) != 2 || len(query["offset"]) != 1 || len(query["wait"]) != 1 {
		return 0, 0, false
	}
	offset, err := strconv.ParseInt(query.Get("offset"), 10, 64)
	waitMS, waitErr := strconv.Atoi(query.Get("wait"))
	return offset, waitMS, err == nil && waitErr == nil && offset >= 0 && waitMS >= 0 && waitMS <= 1000
}

func closeTerminalBackend(ctx context.Context, d *Deps, item panelTerminalSession) error {
	// 关闭 Agent 侧会话(失败仅记录,不阻塞)
	_, err := d.St.Agent.RawCall(ctx, http.MethodPost, "/v1/host/terminal/"+url.PathEscape(item.BackendID)+"/close", "", map[string]any{
		"owner": item.Owner,
	}, item.UserID)
	return err
}

func reserveTerminalOpen(userID string) bool {
	terminalMu.Lock()
	defer terminalMu.Unlock()
	userCount := 0
	for _, item := range terminalSessions {
		if item.UserID == userID {
			userCount++
		}
	}
	return len(terminalSessions) < maxPanelTerminals && userCount < maxPanelTerminalsByUser
}

func releaseTerminalOpen(userID string) {}

func pruneTerminalSessions(before time.Time) []panelTerminalSession {
	terminalMu.Lock()
	defer terminalMu.Unlock()
	stale := make([]panelTerminalSession, 0)
	for id, item := range terminalSessions {
		if item.UpdatedAt.After(before) {
			continue
		}
		stale = append(stale, item)
		delete(terminalSessions, id)
	}
	return stale
}

func deleteTerminalSession(id string) {
	terminalMu.Lock()
	delete(terminalSessions, id)
	terminalMu.Unlock()
}

// isTerminalGone 会话在 Agent 侧已消失(404/409)
func isTerminalGone(err error) bool {
	var ae *agent.AgentError
	if !errors.As(err, &ae) {
		return false
	}
	return ae.Code == "terminal_not_found" || ae.Code == "terminal_closed"
}

// randomHex 随机十六进制串
func randomHex(n int) string {
	b := make([]byte, n/2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// contextHolder 兼容 context.Context 的最小接口(避免额外 import)
var _ = json.Marshal
