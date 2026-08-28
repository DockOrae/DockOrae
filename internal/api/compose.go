package api

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func ndjsonLine(line string) string {
	b, _ := json.Marshal(line)
	return `{"type":"line","data":` + string(b) + "}\n"
}

// runComposeStream 以 NDJSON 流式执行 docker-compose 命令:每行输出一条
// {"type":"line","data":...},结束时发送 {"type":"done","ok":bool,"error":?}
func runComposeStream(c *gin.Context, st *state.AppState, project string, args ...string) error {
	if _, err := exec.LookPath(service.ComposeBin()); err != nil {
		_, _ = io.WriteString(c.Writer, `{"type":"done","ok":false,"error":"compose.binaryMissing"}`+"\n")
		return nil
	}

	cmd := service.ComposeCommand(st, project, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		c.Header("Content-Type", "application/x-ndjson")
		c.Status(200)
		_, _ = io.WriteString(c.Writer, `{"type":"done","ok":false,"error":"compose.binaryMissing"}`+"\n")
		return nil
	}

	c.Header("Content-Type", "application/x-ndjson")
	c.Status(200)
	flusher, _ := c.Writer.(interface{ Flush() })
	write := func(line string) {
		_, _ = io.WriteString(c.Writer, line)
		if flusher != nil {
			flusher.Flush()
		}
	}

	// 交错读取 stdout / stderr,保证进度实时可见
	outLines := make(chan string, 64)
	errLines := make(chan string, 64)
	go scanLines(stdout, outLines)
	go scanLines(stderr, errLines)

	outOpen, errOpen := true, true
	for outOpen || errOpen {
		select {
		case line, ok := <-outLines:
			if !ok {
				outOpen = false
				continue
			}
			write(ndjsonLine(line))
		case line, ok := <-errLines:
			if !ok {
				errOpen = false
				continue
			}
			write(ndjsonLine(line))
		}
	}
	err = cmd.Wait()
	if err == nil {
		write(`{"type":"done","ok":true}` + "\n")
	} else {
		write(`{"type":"done","ok":false,"error":"compose.failed"}` + "\n")
	}
	return nil
}

func scanLines(r io.Reader, ch chan<- string) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		ch <- sc.Text()
	}
	close(ch)
}

func composeList(c *gin.Context, st *state.AppState) error {
	items, err := service.ComposeList(st, c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, items)
	return nil
}

func composeInspect(c *gin.Context, st *state.AppState) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	insp, err := service.ComposeInspect(st, c.Request.Context(), project)
	if err != nil {
		return err
	}
	c.JSON(200, insp)
	return nil
}

func composeUp(c *gin.Context, st *state.AppState) error {
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
	if err := service.ComposeSaveYaml(st, project, req.Yaml); err != nil {
		return err
	}
	return runComposeStream(c, st, project, "up", "-d", "--remove-orphans")
}

func composeUpdate(c *gin.Context, st *state.AppState) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	if _, err := os.Stat(service.ComposeFile(st, project)); err != nil {
		return service.NewApiError(404, "compose.notManaged")
	}
	var req struct {
		Yaml string `json:"yaml"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := service.ComposeSaveYaml(st, project, req.Yaml); err != nil {
		return err
	}
	return runComposeStream(c, st, project, "up", "-d", "--remove-orphans")
}

func composeStart(c *gin.Context, st *state.AppState) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	res, err := service.RunCompose(st, project, "start")
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func composeStop(c *gin.Context, st *state.AppState) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	res, err := service.RunCompose(st, project, "stop")
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func composeRestart(c *gin.Context, st *state.AppState) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	res, err := service.RunCompose(st, project, "restart")
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func composeDown(c *gin.Context, st *state.AppState) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	args := []string{"down"}
	if parseBool(c.Query("volumes"), false) {
		args = append(args, "-v")
	}
	res, err := service.RunCompose(st, project, args...)
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func composeRemove(c *gin.Context, st *state.AppState) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	if err := service.ComposeRemove(st, project); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// composeLogsWS 日志流(WebSocket 实时)
func composeLogsWS(c *gin.Context, st *state.AppState) error {
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
	if _, err := os.Stat(service.ComposeFile(st, project)); err != nil {
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
	cmd := service.ComposeCommand(st, project, "logs", "-f", "--tail", tail)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("compose.logsFailed"))
		_ = conn.WriteMessage(websocket.CloseMessage, nil)
		return nil
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	go func() {
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			if conn.WriteMessage(websocket.TextMessage, []byte(sc.Text())) != nil {
				break
			}
		}
		cancel()
	}()

	wsPump(ctx, conn, func(mt int, data []byte) bool {
		return mt != websocket.CloseMessage
	})
	_ = cmd.Process.Kill()
	_ = cmd.Wait() // 回收子进程,避免僵尸
	_ = conn.WriteMessage(websocket.CloseMessage, nil)
	return nil
}
