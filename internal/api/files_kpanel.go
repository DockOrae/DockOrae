// 宿主文件管理 API(自 KPanel internal/panel/files.go 移植,2026-09-02)。
// 端点(KPanel 同款契约):
//   GET  /api/v1/files                    → 目录列表(path/limit/offset/search)
//   GET  /api/v1/files/entry              → 单条目属性(path)
//   POST /api/v1/files/entries            → 批量属性(paths)
//   GET  /api/v1/files/trash              → 回收站列表
//   GET  /api/v1/files/content            → 读文件/下载(path/disposition/mode/version)
//   PUT  /api/v1/files/content            → 写文件(path, JSON)
//   GET  /api/v1/files/archive            → 压缩下载(selection JSON + name)
//   GET  /api/v1/files/text               → 文本读取(≤64KiB)
//   GET  /api/v1/files/tail               → 文本尾部(≤64KiB)
//   POST /api/v1/files/download-tickets   → 下载 ticket(浏览器 <a> 无 header 场景)
//   POST /api/v1/files/archive-download-tickets → 压缩下载 ticket
//   GET  /api/v1/files/download/{token}   → ticket 消费(流式下载)
//   POST /api/v1/files/upload             → 上传(path/name/overwrite, octet-stream)
//   POST /api/v1/files/actions            → 批量操作
// 全部经 Agent 透传;下载走内存 ticket(30 分钟过期,与 KPanel 同款)。
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/DockOrae/DockOrae/internal/service"
)

const (
	panelFileTransferMaxDuration = 2 * time.Hour
	panelFileTransferIdleTimeout = 45 * time.Second
	fileDownloadTicketTTL        = 30 * time.Minute
	fileDownloadTicketLimit      = 256
)

// fileDownloadTicket 内存下载票据
type fileDownloadTicket struct {
	Path             string
	ArchiveSelection string
	ArchiveName      string
	ExpiresAt        time.Time
}

var (
	downloadTicketMu sync.Mutex
	downloadTickets  = map[string]fileDownloadTicket{}
)

// handleKFileList GET /api/v1/files
func handleKFileList(c *gin.Context, d *Deps) error {
	resp, err := d.St.Agent.RawCall(c.Request.Context(), http.MethodGet, "/v1/files", c.Request.URL.RawQuery, nil, c.GetString("username"))
	if err != nil {
		return agentErr(err)
	}
	c.Data(http.StatusOK, "application/json", resp.Body)
	return nil
}

// handleKFileEntry GET /api/v1/files/entry
func handleKFileEntry(c *gin.Context, d *Deps) error {
	resp, err := d.St.Agent.RawCall(c.Request.Context(), http.MethodGet, "/v1/files/entry", c.Request.URL.RawQuery, nil, c.GetString("username"))
	if err != nil {
		return agentErr(err)
	}
	c.Data(http.StatusOK, "application/json", resp.Body)
	return nil
}

// handleKFileEntries POST /api/v1/files/entries
func handleKFileEntries(c *gin.Context, d *Deps) error {
	var input struct {
		Paths []string `json:"paths"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if len(input.Paths) == 0 || len(input.Paths) > 64 {
		return service.BadRequest("files.errRequest")
	}
	resp, err := d.St.Agent.RawCall(c.Request.Context(), http.MethodPost, "/v1/files/entries", "", input, c.GetString("username"))
	if err != nil {
		return agentErr(err)
	}
	c.Data(http.StatusOK, "application/json", resp.Body)
	return nil
}

// handleKFileTrashList GET /api/v1/files/trash
func handleKFileTrashList(c *gin.Context, d *Deps) error {
	resp, err := d.St.Agent.RawCall(c.Request.Context(), http.MethodGet, "/v1/files/trash", "", nil, c.GetString("username"))
	if err != nil {
		return agentErr(err)
	}
	c.Data(http.StatusOK, "application/json", resp.Body)
	return nil
}

// handleKFileContent GET/PUT /api/v1/files/content
func handleKFileContent(c *gin.Context, d *Deps) error {
	switch c.Request.Method {
	case http.MethodGet, http.MethodHead:
		// 流式下载:Agent 原始流 → 浏览器
		rawQuery := c.Request.URL.RawQuery
		resp, err := d.St.Agent.RawStream(c.Request.Context(), c.Request.Method, "/v1/files/content", rawQuery, http.NoBody, c.GetString("username"), nil)
		if err != nil {
			return agentErr(err)
		}
		defer resp.Body.Close()
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}
		c.Header("Cache-Control", "private, no-store")
		c.Status(resp.StatusCode)
		if c.Request.Method == http.MethodHead {
			return nil
		}
		_, _ = io.CopyBuffer(c.Writer, resp.Body, make([]byte, 64<<10))
		return nil
	case http.MethodPut:
		var input struct {
			Content                 string `json:"content"`
			ExpectedResourceVersion string `json:"expectedResourceVersion"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			return service.BadRequest("err.requestFailed")
		}
		pathValue := c.Query("path")
		if pathValue == "" {
			return service.BadRequest("files.errRequest")
		}
		resp, err := d.St.Agent.RawCall(c.Request.Context(), http.MethodPut, "/v1/files/content?path="+url.QueryEscape(pathValue), "", input, c.GetString("username"))
		if err != nil {
			return agentErr(err)
		}
		c.Data(http.StatusOK, "application/json", resp.Body)
		return nil
	default:
		c.Header("Allow", http.MethodGet+", "+http.MethodHead+", "+http.MethodPut)
		return service.NewApiError(http.StatusMethodNotAllowed, "请求方法不允许")
	}
}

// handleKFileArchive GET /api/v1/files/archive — 压缩下载
func handleKFileArchive(c *gin.Context, d *Deps) error {
	if c.Request.Method != http.MethodGet {
		c.Header("Allow", http.MethodGet)
		return service.NewApiError(http.StatusMethodNotAllowed, "请求方法不允许")
	}
	rawQuery := c.Request.URL.RawQuery
	resp, err := d.St.Agent.RawStream(c.Request.Context(), http.MethodGet, "/v1/files/archive", rawQuery, http.NoBody, c.GetString("username"), nil)
	if err != nil {
		return agentErr(err)
	}
	defer resp.Body.Close()
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)
	_, _ = io.CopyBuffer(c.Writer, resp.Body, make([]byte, 64<<10))
	return nil
}

// handleKFileText GET /api/v1/files/text
func handleKFileText(c *gin.Context, d *Deps) error {
	resp, err := d.St.Agent.RawCall(c.Request.Context(), http.MethodGet, "/v1/files/text", c.Request.URL.RawQuery, nil, c.GetString("username"))
	if err != nil {
		return agentErr(err)
	}
	c.Data(http.StatusOK, "application/json", resp.Body)
	return nil
}

// handleKFileTail GET /api/v1/files/tail
func handleKFileTail(c *gin.Context, d *Deps) error {
	resp, err := d.St.Agent.RawCall(c.Request.Context(), http.MethodGet, "/v1/files/tail", c.Request.URL.RawQuery, nil, c.GetString("username"))
	if err != nil {
		return agentErr(err)
	}
	c.Data(http.StatusOK, "application/json", resp.Body)
	return nil
}

// handleKFileUpload POST /api/v1/files/upload — octet-stream 透传
func handleKFileUpload(c *gin.Context, d *Deps) error {
	rawQuery := c.Request.URL.RawQuery
	resp, err := d.St.Agent.RawStream(c.Request.Context(), http.MethodPost, "/v1/files/upload", rawQuery, c.Request.Body, c.GetString("username"), c.Request.Header)
	if err != nil {
		return agentErr(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	c.Data(resp.StatusCode, "application/json", body)
	return nil
}

// handleKFileAction POST /api/v1/files/actions
func handleKFileAction(c *gin.Context, d *Deps) error {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		return service.BadRequest("err.requestFailed")
	}
	resp, err := d.St.Agent.RawCall(c.Request.Context(), http.MethodPost, "/v1/files/actions", "", json.RawMessage(body), c.GetString("username"))
	if err != nil {
		return agentErr(err)
	}
	c.Data(http.StatusOK, "application/json", resp.Body)
	return nil
}

// handleKDownloadTicketCreate POST /api/v1/files/download-tickets
func handleKDownloadTicketCreate(c *gin.Context, d *Deps) error {
	var input struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if input.Path == "" || !strings.HasPrefix(input.Path, "/") {
		return service.BadRequest("files.errRequest")
	}
	token, expiresAt, err := issueDownloadTicket(fileDownloadTicket{Path: input.Path})
	if err != nil {
		return service.NewApiError(http.StatusTooManyRequests, "下载请求过多,请稍后重试")
	}
	c.JSON(http.StatusCreated, gin.H{
		"downloadUrl": "/api/v1/files/download/" + token,
		"expiresAt":   expiresAt,
	})
	return nil
}

// handleKArchiveDownloadTicketCreate POST /api/v1/files/archive-download-tickets
func handleKArchiveDownloadTicketCreate(c *gin.Context, d *Deps) error {
	var input struct {
		Sources                  []string          `json:"sources"`
		ExpectedResourceVersions map[string]string `json:"expectedResourceVersions"`
		Name                     string            `json:"name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if len(input.Sources) == 0 || input.Name == "" {
		return service.BadRequest("files.errRequest")
	}
	selection, _ := json.Marshal(map[string]any{
		"sources":                  input.Sources,
		"expectedResourceVersions": input.ExpectedResourceVersions,
	})
	token, expiresAt, err := issueDownloadTicket(fileDownloadTicket{
		ArchiveSelection: string(selection),
		ArchiveName:      input.Name,
	})
	if err != nil {
		return service.NewApiError(http.StatusTooManyRequests, "下载请求过多,请稍后重试")
	}
	c.JSON(http.StatusCreated, gin.H{
		"downloadUrl": "/api/v1/files/download/" + token,
		"expiresAt":   expiresAt,
	})
	return nil
}

// handleKDownloadTicket GET /api/v1/files/download/{token} — 消费 ticket 流式下载
func handleKDownloadTicket(c *gin.Context, d *Deps) error {
	token := c.Param("token")
	downloadTicketMu.Lock()
	ticket, ok := downloadTickets[token]
	if ok {
		delete(downloadTickets, token) // 一次性
	}
	downloadTicketMu.Unlock()
	if !ok || time.Now().After(ticket.ExpiresAt) {
		return service.NewApiError(http.StatusNotFound, "下载链接无效或已过期")
	}
	if ticket.ArchiveSelection != "" {
		query := url.Values{}
		query.Set("name", ticket.ArchiveName)
		resp, err := d.St.Agent.RawStream(c.Request.Context(), http.MethodGet, "/v1/files/archive?selection="+url.QueryEscape(ticket.ArchiveSelection)+"&name="+url.QueryEscape(ticket.ArchiveName), "", http.NoBody, c.GetString("username"), nil)
		if err != nil {
			return agentErr(err)
		}
		defer resp.Body.Close()
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}
		c.Status(resp.StatusCode)
		_, _ = io.CopyBuffer(c.Writer, resp.Body, make([]byte, 64<<10))
		return nil
	}
	query := url.Values{}
	query.Set("path", ticket.Path)
	query.Set("disposition", "attachment")
	resp, err := d.St.Agent.RawStream(c.Request.Context(), http.MethodGet, "/v1/files/content", query.Encode(), http.NoBody, c.GetString("username"), nil)
	if err != nil {
		return agentErr(err)
	}
	defer resp.Body.Close()
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Status(resp.StatusCode)
	_, _ = io.CopyBuffer(c.Writer, resp.Body, make([]byte, 64<<10))
	return nil
}

// issueDownloadTicket 签发一次性下载票据
func issueDownloadTicket(ticket fileDownloadTicket) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(fileDownloadTicketTTL)
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", time.Time{}, err
	}
	token := hex.EncodeToString(tokenBytes)
	downloadTicketMu.Lock()
	defer downloadTicketMu.Unlock()
	if len(downloadTickets) >= fileDownloadTicketLimit {
		// 清理过期
		for key, item := range downloadTickets {
			if now.After(item.ExpiresAt) {
				delete(downloadTickets, key)
			}
		}
		if len(downloadTickets) >= fileDownloadTicketLimit {
			return "", time.Time{}, errors.New("ticket limit")
		}
	}
	ticket.ExpiresAt = expiresAt
	downloadTickets[token] = ticket
	return token, expiresAt, nil
}
