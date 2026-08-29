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

	"github.com/DockerManger/Docker_Manager_Go/internal/service"
)

func ndjsonLine(line string) string {
	b, _ := json.Marshal(line)
	return `{"type":"line","data":` + string(b) + "}\n"
}

// runComposeStream 以 NDJSON 流式执行 docker-compose 命令:每行输出一条
// {"type":"line","data":...},结束时发送 {"type":"done","ok":bool,"error":?}
func runComposeStream(c *gin.Context, d *Deps, project string, args ...string) error {
	if _, err := exec.LookPath(service.ComposeBin()); err != nil {
		_, _ = io.WriteString(c.Writer, `{"type":"done","ok":false,"error":"compose.binaryMissing"}`+"\n")
		return nil
	}

	cmd := d.Compose.Command(project, args...)
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
		// GO-003:channel 关闭后置 nil,select 对其永远阻塞,避免空转 busy-wait
		var outCh, errCh <-chan string
		if outOpen {
			outCh = outLines
		}
		if errOpen {
			errCh = errLines
		}
		select {
		case line, ok := <-outCh:
			if !ok {
				outOpen = false
				continue
			}
			write(ndjsonLine(line))
		case line, ok := <-errCh:
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
	if err := d.Compose.SaveYaml(project, req.Yaml); err != nil {
		return err
	}
	return runComposeStream(c, d, project, "up", "-d", "--remove-orphans")
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
	if err := d.Compose.SaveYaml(project, req.Yaml); err != nil {
		return err
	}
	return runComposeStream(c, d, project, "up", "-d", "--remove-orphans")
}

func composeStart(c *gin.Context, d *Deps) error {
	project, err := service.ValidateProject(c.Param("project"))
	if err != nil {
		return err
	}
	res, err := d.Compose.Run(project, "start")
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
	res, err := d.Compose.Run(project, "stop")
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
	res, err := d.Compose.Run(project, "restart")
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
	args := []string{"down"}
	if parseBool(c.Query("volumes"), false) {
		args = append(args, "-v")
	}
	res, err := d.Compose.Run(project, args...)
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
	if err := d.Compose.Remove(project); err != nil {
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

// composeLogsWS 日志流(WebSocket 实时)
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
	cmd := d.Compose.Command(project, "logs", "-f", "--tail", tail)
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
