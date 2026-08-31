package api

import (
	"io"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/DockOrae/DockOrae/internal/agent"
	"github.com/DockOrae/DockOrae/internal/service"
)

// forwardNDJSON 把 Agent NDJSON 流逐行转发给浏览器(compose up / 镜像拉取共用)
func forwardNDJSON(c *gin.Context, stream *agent.StreamBody) error {
	defer stream.Close()
	c.Header("Content-Type", "application/x-ndjson")
	c.Status(200)
	flusher, _ := c.Writer.(interface{ Flush() })
	for {
		raw, err := stream.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			_, _ = c.Writer.Write([]byte(`{"type":"done","ok":false,"error":"stream.failed"}` + "\n"))
			if flusher != nil {
				flusher.Flush()
			}
			return nil
		}
		_, _ = c.Writer.Write(append(raw, '\n'))
		if flusher != nil {
			flusher.Flush()
		}
	}
}

func composeList(c *gin.Context, d *Deps) error {
	items, err := d.Compose.List(c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, items)
	return nil
}

func composeInspect(c *gin.Context, d *Deps) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	insp, err := d.Compose.Inspect(c.Request.Context(), project)
	if err != nil {
		return err
	}
	c.JSON(200, insp)
	return nil
}

func composeUp(c *gin.Context, d *Deps) error {
	var req struct {
		Project string `json:"project"`
		Yaml    string `json:"yaml"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	project, err := service.ValidateProject(req.Project)
	if err != nil {
		return err
	}
	stream, err := d.Compose.UpStream(c.Request.Context(), project, req.Yaml, "")
	if err != nil {
		return err
	}
	return forwardNDJSON(c, stream)
}

func composeUpdate(c *gin.Context, d *Deps) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	if _, err := os.Stat(d.Compose.File(project)); err != nil {
		return service.NewApiError(404, "compose.notManaged")
	}
	var req struct {
		Yaml string `json:"yaml"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	stream, err := d.Compose.UpStream(c.Request.Context(), project, req.Yaml, "")
	if err != nil {
		return err
	}
	return forwardNDJSON(c, stream)
}

func composeStart(c *gin.Context, d *Deps) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	res, err := d.Compose.Run(c.Request.Context(), project, "start")
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func composeStop(c *gin.Context, d *Deps) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	res, err := d.Compose.Run(c.Request.Context(), project, "stop")
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func composeRestart(c *gin.Context, d *Deps) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	res, err := d.Compose.Run(c.Request.Context(), project, "restart")
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func composeDown(c *gin.Context, d *Deps) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	args := []string{}
	if parseBool(c.Query("volumes"), false) {
		args = append(args, "-v")
	}
	res, err := d.Compose.Run(c.Request.Context(), project, "down", args...)
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func composeRemove(c *gin.Context, d *Deps) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	if err := d.Compose.Remove(c.Request.Context(), project); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// composeAdopt 接管外部创建的栈(保存 yaml 到面板目录 → 变为面板管理)
func composeAdopt(c *gin.Context, d *Deps) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	var req struct {
		Yaml string `json:"yaml"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.Compose.Adopt(project, req.Yaml); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// composeLogsWS 日志流(浏览器 ↔ Agent 透传)
func composeLogsWS(c *gin.Context, d *Deps) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		_ = conn.WriteMessage(websocket.CloseMessage, nil)
		return nil
	}
	if _, err := os.Stat(d.Compose.File(project)); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("compose.notManaged"))
		_ = conn.WriteMessage(websocket.CloseMessage, nil)
		return nil
	}
	tail := "300"
	if t := c.Query("tail"); t != "" {
		if _, err := strconv.Atoi(t); err == nil {
			tail = t
		}
	}
	aconn, err := d.St.Agent.ComposeLogsWS(c.Request.Context(), project, tail)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("compose.logsFailed"))
		_ = conn.WriteMessage(websocket.CloseMessage, nil)
		return nil
	}
	relayWS(c, conn, aconn)
	return nil
}
