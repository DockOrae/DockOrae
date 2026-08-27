package api

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// Host SSH 主机
type Host struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Addr        string `json:"addr"`
	Port        uint16 `json:"port"`
	User        string `json:"user"`
	AuthMode    string `json:"auth_mode"` // password | key
	Password    string `json:"password"`
	PrivateKey  string `json:"private_key"`
	Group       string `json:"group"`
	Description string `json:"description"`
	Created     int64  `json:"created"`
}

func hostsPath(st *state.AppState) string {
	return filepath.Join(st.Cfg.DataDir, "hosts.json")
}

func loadHosts(st *state.AppState) []Host {
	var hosts []Host
	if raw, err := os.ReadFile(hostsPath(st)); err == nil {
		_ = json.Unmarshal(raw, &hosts)
	}
	return hosts
}

func saveHosts(st *state.AppState, hosts []Host) error {
	out, err := json.MarshalIndent(hosts, "", "  ")
	if err != nil {
		return BadRequest(err.Error())
	}
	if err := os.WriteFile(hostsPath(st), out, 0o644); err != nil {
		return BadRequest("write failed: " + err.Error())
	}
	return nil
}

func groupsOf(hosts []Host) []string {
	seen := map[string]bool{}
	var groups []string
	for _, h := range hosts {
		if h.Group != "" && !seen[h.Group] {
			seen[h.Group] = true
			groups = append(groups, h.Group)
		}
	}
	if !seen["Default"] {
		groups = append(groups, "Default")
	}
	sortStrings(groups)
	return groups
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// hostsList 主机列表(含分组)
func hostsList(c *gin.Context, st *state.AppState) error {
	hosts := loadHosts(st)
	c.JSON(200, gin.H{"hosts": hosts, "groups": groupsOf(hosts)})
	return nil
}

// hostsAdd 新增主机
func hostsAdd(c *gin.Context, st *state.AppState) error {
	var payload Host
	if err := c.ShouldBindJSON(&payload); err != nil {
		return BadRequest("err.requestFailed")
	}
	if strings.TrimSpace(payload.Addr) == "" {
		return BadRequest("host.addrRequired")
	}
	if payload.Port == 0 || payload.Port > 65535 {
		return BadRequest("host.portInvalid")
	}
	if payload.AuthMode == "password" && payload.Password == "" {
		return BadRequest("host.pwdRequired")
	}
	if payload.AuthMode == "key" && strings.TrimSpace(payload.PrivateKey) == "" {
		return BadRequest("host.keyRequired")
	}
	hosts := loadHosts(st)
	host := Host{
		ID:          "h" + strconv.FormatInt(time.Now().UnixMilli(), 10),
		Name:        orDefault(strings.TrimSpace(payload.Name), payload.Addr),
		Addr:        strings.TrimSpace(payload.Addr),
		Port:        payload.Port,
		User:        orDefault(strings.TrimSpace(payload.User), "root"),
		AuthMode:    "password",
		Password:    payload.Password,
		PrivateKey:  payload.PrivateKey,
		Group:       orDefault(strings.TrimSpace(payload.Group), "Default"),
		Description: payload.Description,
		Created:     time.Now().Unix(),
	}
	if payload.AuthMode == "key" {
		host.AuthMode = "key"
	}
	hosts = append(hosts, host)
	if err := saveHosts(st, hosts); err != nil {
		return err
	}
	c.JSON(200, gin.H{"host": host})
	return nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// hostsUpdate 编辑主机
func hostsUpdate(c *gin.Context, st *state.AppState) error {
	id := c.Param("id")
	var payload Host
	if err := c.ShouldBindJSON(&payload); err != nil {
		return BadRequest("err.requestFailed")
	}
	hosts := loadHosts(st)
	found := false
	for i := range hosts {
		if hosts[i].ID != id {
			continue
		}
		h := &hosts[i]
		h.Name = orDefault(strings.TrimSpace(payload.Name), payload.Addr)
		h.Addr = strings.TrimSpace(payload.Addr)
		h.Port = payload.Port
		h.User = orDefault(strings.TrimSpace(payload.User), "root")
		h.AuthMode = "password"
		if payload.AuthMode == "key" {
			h.AuthMode = "key"
		}
		if payload.Password != "" {
			h.Password = payload.Password
		}
		if strings.TrimSpace(payload.PrivateKey) != "" {
			h.PrivateKey = payload.PrivateKey
		}
		h.Group = orDefault(strings.TrimSpace(payload.Group), "Default")
		h.Description = payload.Description
		found = true
		break
	}
	if !found {
		return BadRequest("host.notFound")
	}
	if err := saveHosts(st, hosts); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// hostsDelete 删除主机
func hostsDelete(c *gin.Context, st *state.AppState) error {
	id := c.Param("id")
	hosts := loadHosts(st)
	var kept []Host
	for _, h := range hosts {
		if h.ID != id {
			kept = append(kept, h)
		}
	}
	if err := saveHosts(st, kept); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// ---------------- SSH 终端 ----------------

func sshClientConfig(h Host) (*ssh.ClientConfig, error) {
	cfg := &ssh.ClientConfig{
		User:            h.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	if h.AuthMode == "key" {
		signer, err := ssh.ParsePrivateKey([]byte(h.PrivateKey))
		if err != nil {
			return nil, err
		}
		cfg.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else {
		cfg.Auth = []ssh.AuthMethod{ssh.Password(h.Password)}
	}
	return cfg, nil
}

func sshConnect(h Host) (*ssh.Client, error) {
	cfg, err := sshClientConfig(h)
	if err != nil {
		return nil, err
	}
	return ssh.Dial("tcp", net.JoinHostPort(h.Addr, strconv.Itoa(int(h.Port))), cfg)
}

// hostsTest 测试主机连接
func hostsTest(c *gin.Context, st *state.AppState) error {
	id := c.Param("id")
	hosts := loadHosts(st)
	var host *Host
	for i := range hosts {
		if hosts[i].ID == id {
			h := hosts[i]
			host = &h
			break
		}
	}
	if host == nil {
		return BadRequest("host.notFound")
	}
	client, err := sshConnect(*host)
	if err != nil {
		return BadRequest("host.connectFailed: " + err.Error())
	}
	client.Close()
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// hostsTerminalWS 主机 SSH 终端 WebSocket
func hostsTerminalWS(c *gin.Context, st *state.AppState) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	id := c.Param("id")
	var host *Host
	hosts := loadHosts(st)
	for i := range hosts {
		if hosts[i].ID == id {
			h := hosts[i]
			host = &h
			break
		}
	}
	if host == nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[host not found]\r\n"))
		conn.Close()
		return nil
	}
	sshTerminalLoop(c, st, conn, *host)
	return nil
}

func sshTerminalLoop(c *gin.Context, st *state.AppState, conn *websocket.Conn, host Host) {
	client, err := sshConnect(host)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[ssh connect failed: "+err.Error()+"]\r\n"))
		conn.Close()
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[channel open failed: "+err.Error()+"]\r\n"))
		conn.Close()
		return
	}
	defer session.Close()

	if err := session.RequestPty("xterm", 80, 24, ssh.TerminalModes{}); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[pty request failed]\r\n"))
		conn.Close()
		return
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		conn.Close()
		return
	}
	out := sshWSWriter{conn: conn}
	session.Stdout = out
	session.Stderr = out
	if err := session.Shell(); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[shell request failed]\r\n"))
		conn.Close()
		return
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// SSH 会话结束时关闭 WS
	go func() {
		_ = session.Wait()
		cancel()
	}()

	wsPump(ctx, conn, func(mt int, data []byte) bool {
		switch mt {
		case websocket.BinaryMessage:
			_, _ = stdin.Write(data)
		case websocket.TextMessage:
			text := string(data)
			if strings.HasPrefix(text, "resize:") {
				parts := strings.SplitN(strings.TrimPrefix(text, "resize:"), ",", 2)
				if len(parts) == 2 {
					w, err1 := strconv.Atoi(parts[0])
					h, err2 := strconv.Atoi(parts[1])
					if err1 == nil && err2 == nil {
						_ = session.WindowChange(w, h)
					}
				}
			} else if text == "stop" {
				return false
			} else {
				_, _ = stdin.Write(data)
			}
		case websocket.CloseMessage:
			return false
		}
		return true
	})
	_ = conn.WriteMessage(websocket.CloseMessage, nil)
}

type sshWSWriter struct {
	conn *websocket.Conn
}

func (w sshWSWriter) Write(p []byte) (int, error) {
	if err := w.conn.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

var _ io.Writer = sshWSWriter{}
