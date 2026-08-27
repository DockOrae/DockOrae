package api

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func composeBin() string {
	if b := os.Getenv("COMPOSE_BIN"); b != "" {
		return b
	}
	return "docker-compose"
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func validateProject(p string) (string, *ApiError) {
	if p == "" || len(p) > 64 {
		return "", BadRequest("compose.nameInvalid")
	}
	for _, r := range p {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "", BadRequest("compose.nameInvalid")
	}
	return p, nil
}

func projectDir(st *state.AppState, project string) string {
	return filepath.Join(st.ComposeDir, project)
}

func composeFile(st *state.AppState, project string) string {
	return filepath.Join(projectDir(st, project), "docker-compose.yml")
}

func composeArgs(st *state.AppState, project string, args ...string) *exec.Cmd {
	cmd := exec.Command(composeBin(), append([]string{"-p", project, "-f", composeFile(st, project)}, args...)...)
	return cmd
}

// runCompose 同步执行 compose 命令,返回 {ok, output}
func runCompose(st *state.AppState, project string, args ...string) (map[string]any, error) {
	cmd := composeArgs(st, project, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, NewApiError(502, msg)
	}
	output := strings.TrimSpace(string(out))
	return gin.H{"ok": true, "output": output}, nil
}

// ---------------- 列表 / 详情 ----------------

func composeList(c *gin.Context, st *state.AppState) error {
	filters := make(client.Filters)
	filters.Add("label", "com.docker.compose.project")
	res, err := st.Docker.ContainerList(c.Request.Context(), client.ContainerListOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return dockerError(err)
	}
	projects := map[string][2]int{} // name -> [total, running]
	for _, ctr := range res.Items {
		name := ctr.Labels["com.docker.compose.project"]
		if name == "" {
			continue
		}
		entry := projects[name]
		entry[0]++
		if string(ctr.State) == "running" {
			entry[1]++
		}
		projects[name] = entry
	}
	names := make([]string, 0, len(projects))
	for name := range projects {
		names = append(names, name)
	}
	sortStrings(names)

	out := make([]gin.H, 0, len(names))
	for _, name := range names {
		total, running := projects[name][0], projects[name][1]
		status := "stopped"
		if running == total {
			status = "running"
		} else if running > 0 {
			status = "partial"
		}
		_, hasFile := os.Stat(composeFile(st, name))
		out = append(out, gin.H{
			"project":  name,
			"services": total,
			"running":  running,
			"status":   status,
			"managed":  hasFile == nil,
		})
	}
	c.JSON(200, out)
	return nil
}

func composeInspect(c *gin.Context, st *state.AppState) error {
	project, ae := validateProject(c.Param("project"))
	if ae != nil {
		return ae
	}
	filters := make(client.Filters)
	filters.Add("label", "com.docker.compose.project="+project)
	res, err := st.Docker.ContainerList(c.Request.Context(), client.ContainerListOptions{
		All:     true,
		Filters: filters,
	})
	if err != nil {
		return dockerError(err)
	}
	var yaml *string
	if raw, err := os.ReadFile(composeFile(st, project)); err == nil {
		s := string(raw)
		yaml = &s
	}
	c.JSON(200, gin.H{"project": project, "containers": res.Items, "yaml": yaml})
	return nil
}

// ---------------- 部署 (NDJSON 流式输出) ----------------

type upReq struct {
	Project string `json:"project"`
	Yaml    string `json:"yaml"`
}

func ndjsonLine(line string) string {
	b, _ := json.Marshal(line)
	return `{"type":"line","data":` + string(b) + "}\n"
}

// runComposeStream 以 NDJSON 流式执行 docker-compose 命令:每行输出一条
// {"type":"line","data":...},结束时发送 {"type":"done","ok":bool,"error":?}
func runComposeStream(c *gin.Context, st *state.AppState, project string, args ...string) error {
	cmd := composeArgs(st, project, args...)
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

func composeUp(c *gin.Context, st *state.AppState) error {
	// 许可证限制:未激活时禁止部署 Compose(1Panel 商业版功能锁定)
	if !licenseActive(st) {
		return NewApiError(403, "license.required")
	}
	var req upReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("err.requestFailed")
	}
	project, ae := validateProject(req.Project)
	if ae != nil {
		return ae
	}
	dir := projectDir(st, project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(composeFile(st, project), []byte(req.Yaml), 0o644); err != nil {
		return err
	}
	return runComposeStream(c, st, project, "up", "-d", "--remove-orphans")
}

func composeUpdate(c *gin.Context, st *state.AppState) error {
	project, ae := validateProject(c.Param("project"))
	if ae != nil {
		return ae
	}
	if _, err := os.Stat(composeFile(st, project)); err != nil {
		return NewApiError(404, "compose.notManaged")
	}
	var req upReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("err.requestFailed")
	}
	if err := os.WriteFile(composeFile(st, project), []byte(req.Yaml), 0o644); err != nil {
		return err
	}
	return runComposeStream(c, st, project, "up", "-d", "--remove-orphans")
}

// ---------------- 栈操作 ----------------

func composeStart(c *gin.Context, st *state.AppState) error {
	project, ae := validateProject(c.Param("project"))
	if ae != nil {
		return ae
	}
	res, err := runCompose(st, project, "start")
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func composeStop(c *gin.Context, st *state.AppState) error {
	project, ae := validateProject(c.Param("project"))
	if ae != nil {
		return ae
	}
	res, err := runCompose(st, project, "stop")
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func composeRestart(c *gin.Context, st *state.AppState) error {
	project, ae := validateProject(c.Param("project"))
	if ae != nil {
		return ae
	}
	res, err := runCompose(st, project, "restart")
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func composeDown(c *gin.Context, st *state.AppState) error {
	project, ae := validateProject(c.Param("project"))
	if ae != nil {
		return ae
	}
	args := []string{"down"}
	if parseBool(c.Query("volumes"), false) {
		args = append(args, "-v")
	}
	res, err := runCompose(st, project, args...)
	if err != nil {
		return err
	}
	c.JSON(200, res)
	return nil
}

func composeRemove(c *gin.Context, st *state.AppState) error {
	project, ae := validateProject(c.Param("project"))
	if ae != nil {
		return ae
	}
	_, _ = runCompose(st, project, "down")
	_ = os.RemoveAll(projectDir(st, project))
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// ---------------- 日志 (WebSocket 实时) ----------------

func composeLogsWS(c *gin.Context, st *state.AppState) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()
	project, ae := validateProject(c.Param("project"))
	if ae != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(ae.Message))
		_ = conn.WriteMessage(websocket.CloseMessage, nil)
		return nil
	}
	if _, err := os.Stat(composeFile(st, project)); err != nil {
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
	cmd := composeArgs(st, project, "logs", "-f", "--tail", tail)
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
	_ = conn.WriteMessage(websocket.CloseMessage, nil)
	return nil
}
