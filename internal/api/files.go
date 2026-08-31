// 宿主文件管理 API(§55):面板 → Agent 的固定端点映射 + 流式上传/下载/WS 终端桥接。
// 架构:Frontend → DockOrae(本文件)→ DockOrae-Agent → 宿主文件系统。
// Agent 离线时所有调用返回 AGENT_UNAVAILABLE(502),前端显示"宿主机 Agent 未连接"。
package api

import (
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/DockOrae/DockOrae/internal/service"
)

// ---------------- 列表 / 属性 ----------------

func filesList(c *gin.Context, d *Deps) error {
	res, err := d.St.Agent.FilesList(c.Request.Context(), c.Query("path"), c.Query("show_hidden") == "true")
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"path": res.Path, "entries": res.Entries})
	return nil
}

func filesDirSize(c *gin.Context, d *Deps) error {
	size, err := d.St.Agent.FilesDirSize(c.Request.Context(), c.Query("path"))
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"size": size})
	return nil
}

func filesChown(c *gin.Context, d *Deps) error {
	var req struct {
		Path  string `json:"path"`
		Owner string `json:"owner"`
		Group string `json:"group"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.St.Agent.FilesChown(c.Request.Context(), req.Path, req.Owner, req.Group); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func filesStat(c *gin.Context, d *Deps) error {
	e, err := d.St.Agent.FilesStat(c.Request.Context(), c.Query("path"))
	if err != nil {
		return err
	}
	c.JSON(200, e)
	return nil
}

// ---------------- 新建 / 重命名 / 复制 / 移动 ----------------

func filesTouch(c *gin.Context, d *Deps) error {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.St.Agent.FilesTouch(c.Request.Context(), req.Path); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func filesMkdir(c *gin.Context, d *Deps) error {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.St.Agent.FilesMkdir(c.Request.Context(), req.Path); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func filesRename(c *gin.Context, d *Deps) error {
	var req struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.St.Agent.FilesRename(c.Request.Context(), req.OldPath, req.NewPath); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func filesCopy(c *gin.Context, d *Deps) error {
	var req struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.St.Agent.FilesCopy(c.Request.Context(), req.Src, req.Dst); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func filesMove(c *gin.Context, d *Deps) error {
	var req struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.St.Agent.FilesMove(c.Request.Context(), req.Src, req.Dst); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// ---------------- 删除 / 权限 / 写入 ----------------

func filesRemove(c *gin.Context, d *Deps) error {
	var req struct {
		Paths     []string `json:"paths"`
		Recursive bool     `json:"recursive"`
		Force     bool     `json:"force"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if len(req.Paths) == 0 {
		return service.BadRequest("files.errNoTarget")
	}
	trashed, err := d.St.Agent.FilesRemove(c.Request.Context(), req.Paths, req.Recursive, req.Force)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true, "trashed": trashed})
	return nil
}

// ---------------- 回收站 ----------------

func trashStatus(c *gin.Context, d *Deps) error {
	st, err := d.St.Agent.TrashStatus(c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, st)
	return nil
}

func trashSetEnabled(c *gin.Context, d *Deps) error {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.St.Agent.TrashSetEnabled(c.Request.Context(), req.Enabled); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func trashList(c *gin.Context, d *Deps) error {
	items, err := d.St.Agent.TrashList(c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"items": items})
	return nil
}

func trashRestore(c *gin.Context, d *Deps) error {
	var req struct {
		Names []string `json:"names"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.St.Agent.TrashRestore(c.Request.Context(), req.Names); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func trashDelete(c *gin.Context, d *Deps) error {
	var req struct {
		Names []string `json:"names"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.St.Agent.TrashDelete(c.Request.Context(), req.Names); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func trashEmpty(c *gin.Context, d *Deps) error {
	if err := d.St.Agent.TrashEmpty(c.Request.Context()); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func filesChmod(c *gin.Context, d *Deps) error {
	var req struct {
		Path string `json:"path"`
		Mode uint32 `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.St.Agent.FilesChmod(c.Request.Context(), req.Path, req.Mode); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func filesWrite(c *gin.Context, d *Deps) error {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.St.Agent.FilesWrite(c.Request.Context(), req.Path, req.Content); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// ---------------- 上传 / 下载(流式) ----------------

// filesUpload 上传:multipart → 流式转发 Agent → 宿主原子写入
func filesUpload(c *gin.Context, d *Deps) error {
	dir := c.Query("dir")
	file, err := c.FormFile("file")
	if err != nil {
		return service.BadRequest("files.errUpload")
	}
	f, err := file.Open()
	if err != nil {
		return service.BadRequest("files.errUpload")
	}
	defer f.Close()
	resp, err := d.St.Agent.FilesUploadRaw(c.Request.Context(), dir, file.Filename, f)
	if err != nil {
		return err
	}
	resp.Body.Close()
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// filesDownload 下载:Agent 原始流 → 浏览器(带文件名)
func filesDownload(c *gin.Context, d *Deps) error {
	p := c.Query("path")
	resp, err := d.St.Agent.FilesDownloadRaw(c.Request.Context(), p)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	name := path.Base(strings.TrimSuffix(p, "/"))
	if name == "" || name == "." || name == "/" {
		name = "download"
	}
	encoded := url.PathEscape(name)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+encoded)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, resp.Body)
	return nil
}

// ---------------- 压缩 / 解压 / 搜索 ----------------

func filesCompress(c *gin.Context, d *Deps) error {
	var req struct {
		Dir     string   `json:"dir"`
		Archive string   `json:"archive"`
		Format  string   `json:"format"`
		Names   []string `json:"names"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	res, err := d.St.Agent.FilesCompress(c.Request.Context(), req.Dir, req.Archive, req.Format, req.Names)
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func filesExtract(c *gin.Context, d *Deps) error {
	var req struct {
		Archive string `json:"archive"`
		Dest    string `json:"dest"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	res, err := d.St.Agent.FilesExtract(c.Request.Context(), req.Archive, req.Dest)
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func filesSearch(c *gin.Context, d *Deps) error {
	limit, _ := strconv.Atoi(c.Query("limit"))
	results, truncated, err := d.St.Agent.FilesSearch(c.Request.Context(), c.Query("path"), c.Query("q"), limit)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"results": results, "truncated": truncated})
	return nil
}

// ---------------- 宿主终端(WS 桥接) ----------------

// hostTerminalWS 宿主终端:浏览器 WS ↔ Agent WS 全双工透传(复用 relayWS)
func hostTerminalWS(c *gin.Context, d *Deps) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()

	cols, _ := strconv.Atoi(c.Query("cols"))
	rows, _ := strconv.Atoi(c.Query("rows"))
	aconn, err := d.St.Agent.HostTerminalWS(c.Request.Context(), c.Query("cwd"), cols, rows)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[terminal failed: "+err.Error()+"]\r\n"))
		_ = conn.WriteMessage(websocket.CloseMessage, nil)
		return nil
	}
	relayWS(c, conn, aconn)
	return nil
}
