package api

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/auth"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/notify"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// ---------------- 公共 ----------------

func systemHealth(c *gin.Context, st *state.AppState) error {
	c.JSON(200, gin.H{
		"ok":      true,
		"name":    "docker-manager-go",
		"version": "1.0.0",
	})
	return nil
}

// systemDefaultAccount 登录页提示"默认账号"的条件:admin 用户存在且仍为默认密码
func systemDefaultAccount(c *gin.Context, st *state.AppState) error {
	show := false
	st.UsersMu.Lock()
	for i := range st.Users {
		u := &st.Users[i]
		if u.Username == "admin" && u.MustChangePassword {
			show = true
			break
		}
	}
	st.UsersMu.Unlock()
	c.JSON(200, gin.H{"show": show})
	return nil
}

func publicUser(u *state.StoredUser) gin.H {
	return gin.H{
		"username":           u.Username,
		"nickname":           u.Nickname,
		"avatar":             u.Avatar,
		"must_change_password": u.MustChangePassword,
		"totp_enabled":       u.TotpSecret != nil,
	}
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func systemLogin(c *gin.Context, st *state.AppState) error {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("login.errPwd")
	}
	u := st.FindUser(req.Username)
	if u == nil || !auth.VerifyPassword(req.Password, u.PasswordHash) {
		// 登录失败 → 通知
		notify.Notify(st.Settings, notify.EvLoginFail, "面板登录失败",
			"用户: "+req.Username+"\nIP: "+c.ClientIP()+"\n时间: "+time.Now().Format("2006-01-02 15:04:05"))
		return NewApiError(401, "login.errPwd")
	}
	// 已启用 2FA:第一步只校验密码,返回 totp_required
	if u.TotpSecret != nil {
		c.JSON(200, gin.H{"totp_required": true, "username": u.Username})
		return nil
	}
	token := auth.MakeToken(st.Cfg.JWTSecret, u.Username, st.Settings.SessionTTLSeconds())
	notify.Notify(st.Settings, notify.EvLogin, "面板登录成功",
		"用户: "+u.Username+"\nIP: "+c.ClientIP()+"\n时间: "+time.Now().Format("2006-01-02 15:04:05"))
	c.JSON(200, gin.H{
		"token": token, "username": u.Username,
		"nickname": u.Nickname, "avatar": u.Avatar,
		"must_change_password": u.MustChangePassword, "totp_enabled": false,
	})
	return nil
}

type loginTotpReq struct {
	Username string `json:"username"`
	Code     string `json:"code"`
}

func systemLoginTotp(c *gin.Context, st *state.AppState) error {
	var req loginTotpReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("login.errTotpCode")
	}
	u := st.FindUser(req.Username)
	if u == nil || u.TotpSecret == nil {
		return NewApiError(401, "login.errTotpCode")
	}
	if !auth.VerifyTotp(*u.TotpSecret, req.Code) {
		return NewApiError(401, "login.errTotpCode")
	}
	token := auth.MakeToken(st.Cfg.JWTSecret, u.Username, st.Settings.SessionTTLSeconds())
	c.JSON(200, gin.H{
		"token": token, "username": u.Username,
		"nickname": u.Nickname, "avatar": u.Avatar,
		"must_change_password": u.MustChangePassword, "totp_enabled": true,
	})
	return nil
}

func systemMe(c *gin.Context, st *state.AppState) error {
	username := c.GetString("username")
	u := st.FindUser(username)
	if u == nil {
		return NewApiError(404, "user.notFound")
	}
	c.JSON(200, publicUser(u))
	return nil
}

// ---------------- 个人资料 ---------------- 

type updateProfileReq struct {
	Nickname *string `json:"nickname"`
	Username *string `json:"username"`
}

func systemUpdateProfile(c *gin.Context, st *state.AppState) error {
	username := c.GetString("username")
	var req updateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("err.requestFailed")
	}
	// 第一遍:只读校验
	var nickname *string
	if req.Nickname != nil {
		t := strings.TrimSpace(*req.Nickname)
		if t != "" {
			if len([]rune(t)) > 32 {
				return BadRequest("user.nicknameTooLong")
			}
			nickname = &t
		}
	}
	var newUsername *string
	if req.Username != nil {
		t := strings.TrimSpace(*req.Username)
		if t != "" {
			if len([]rune(t)) > 32 || !isValidUsername(t) {
				return BadRequest("user.usernameInvalid")
			}
			st.UsersMu.Lock()
			for i := range st.Users {
				if st.Users[i].Username == t && st.Users[i].Username != username {
					st.UsersMu.Unlock()
					return NewApiError(409, "user.usernameTaken")
				}
			}
			st.UsersMu.Unlock()
			newUsername = &t
		}
	}
	// 第二遍:应用修改
	st.UsersMu.Lock()
	var u *state.StoredUser
	for i := range st.Users {
		if st.Users[i].Username == username {
			u = &st.Users[i]
			break
		}
	}
	if u == nil {
		st.UsersMu.Unlock()
		return NewApiError(404, "user.notFound")
	}
	u.Nickname = nickname
	var newToken string
	if newUsername != nil {
		u.Username = *newUsername
		newToken = auth.MakeToken(st.Cfg.JWTSecret, *newUsername, st.Settings.SessionTTLSeconds())
	}
	snapshot := *u
	st.UsersMu.Unlock()
	if err := st.SaveUsers(); err != nil {
		return err
	}
	resp := publicUser(&snapshot)
	if newToken != "" {
		resp["token"] = newToken
	}
	c.JSON(200, resp)
	return nil
}

func isValidUsername(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

// ---------------- 头像 ---------------- 

const maxAvatarBytes = 2 * 1024 * 1024

type avatarReq struct {
	Data string `json:"data"`
}

func sniffImageExt(b []byte) string {
	switch {
	case len(b) > 8 && bytes.Equal(b[:8], []byte("\x89PNG\r\n\x1a\n")):
		return "png"
	case len(b) > 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff:
		return "jpg"
	case len(b) > 6 && (bytes.Equal(b[:6], []byte("GIF87a")) || bytes.Equal(b[:6], []byte("GIF89a"))):
		return "gif"
	case len(b) > 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return "webp"
	}
	return ""
}

func systemUploadAvatar(c *gin.Context, st *state.AppState) error {
	username := c.GetString("username")
	var req avatarReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("user.avatarInvalid")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.Data))
	if err != nil || len(data) == 0 {
		return BadRequest("user.avatarInvalid")
	}
	if len(data) > maxAvatarBytes {
		return BadRequest("user.avatarTooLarge")
	}
	ext := sniffImageExt(data)
	if ext == "" {
		return BadRequest("user.avatarType")
	}
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	fileName := hex.EncodeToString(id) + "." + ext
	path := filepath.Join(st.AvatarDir, fileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}

	st.UsersMu.Lock()
	var u *state.StoredUser
	for i := range st.Users {
		if st.Users[i].Username == username {
			u = &st.Users[i]
			break
		}
	}
	if u == nil {
		st.UsersMu.Unlock()
		_ = os.Remove(path)
		return NewApiError(404, "user.notFound")
	}
	old := u.Avatar
	u.Avatar = &fileName
	st.UsersMu.Unlock()
	if err := st.SaveUsers(); err != nil {
		return err
	}
	if old != nil && *old != "" {
		_ = os.Remove(filepath.Join(st.AvatarDir, *old))
	}
	c.JSON(200, gin.H{"ok": true, "avatar": fileName})
	return nil
}

func systemServeAvatar(c *gin.Context, st *state.AppState) error {
	file := c.Param("file")
	if file == "" || len(file) > 64 || !safeFileName(file) {
		return BadRequest("user.avatarNameInvalid")
	}
	data, err := os.ReadFile(filepath.Join(st.AvatarDir, file))
	if err != nil {
		return NewApiError(404, "user.avatarNotFound")
	}
	ctype := mimeByExt(file)
	c.Header("Content-Type", ctype)
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(200, ctype, data)
	return nil
}

func safeFileName(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

// ---------------- 修改密码 ---------------- 

type changePasswordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func systemChangePassword(c *gin.Context, st *state.AppState) error {
	username := c.GetString("username")
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("err.requestFailed")
	}
	st.UsersMu.Lock()
	var u *state.StoredUser
	for i := range st.Users {
		if st.Users[i].Username == username {
			u = &st.Users[i]
			break
		}
	}
	if u == nil {
		st.UsersMu.Unlock()
		return NewApiError(404, "user.notFound")
	}
	if !auth.VerifyPassword(req.OldPassword, u.PasswordHash) {
		st.UsersMu.Unlock()
		return BadRequest("user.oldPwdWrong")
	}
	if len(req.NewPassword) < 6 {
		st.UsersMu.Unlock()
		return BadRequest("user.pwdTooShort")
	}
	if auth.VerifyPassword(req.NewPassword, u.PasswordHash) {
		st.UsersMu.Unlock()
		return BadRequest("user.pwdSame")
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		st.UsersMu.Unlock()
		return err
	}
	u.PasswordHash = hash
	u.MustChangePassword = false
	st.UsersMu.Unlock()
	if err := st.SaveUsers(); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// ---------------- 双因素验证 (TOTP) ---------------- 

type totpSetupReq struct {
	Password string `json:"password"`
}

func systemTotpSetup(c *gin.Context, st *state.AppState) error {
	username := c.GetString("username")
	var req totpSetupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("err.requestFailed")
	}
	u := st.FindUser(username)
	if u == nil {
		return NewApiError(404, "user.notFound")
	}
	if !auth.VerifyPassword(req.Password, u.PasswordHash) {
		return BadRequest("user.pwdWrong")
	}
	if u.TotpSecret != nil {
		return BadRequest("totp.alreadyEnabled")
	}
	secret, err := auth.GenerateTotpSecret()
	if err != nil {
		return err
	}
	uri := auth.TotpURI(secret, "Docker Manager", u.Username)
	st.TotpMu.Lock()
	st.TotpPending = &state.TotpPending{Username: username, Secret: secret}
	st.TotpMu.Unlock()
	c.JSON(200, gin.H{"secret": secret, "uri": uri})
	return nil
}

type totpEnableReq struct {
	Code string `json:"code"`
}

func systemTotpEnable(c *gin.Context, st *state.AppState) error {
	username := c.GetString("username")
	var req totpEnableReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("totp.getKeyFirst")
	}
	st.TotpMu.Lock()
	pending := st.TotpPending
	st.TotpMu.Unlock()
	if pending == nil || pending.Username != username {
		return BadRequest("totp.getKeyFirst")
	}
	if !auth.VerifyTotp(pending.Secret, req.Code) {
		return BadRequest("totp.codeWrong")
	}
	st.UsersMu.Lock()
	var u *state.StoredUser
	for i := range st.Users {
		if st.Users[i].Username == username {
			u = &st.Users[i]
			break
		}
	}
	if u == nil {
		st.UsersMu.Unlock()
		return NewApiError(404, "user.notFound")
	}
	u.TotpSecret = &pending.Secret
	st.UsersMu.Unlock()
	if err := st.SaveUsers(); err != nil {
		return err
	}
	st.TotpMu.Lock()
	st.TotpPending = nil
	st.TotpMu.Unlock()
	c.JSON(200, gin.H{"ok": true})
	return nil
}

type totpDisableReq struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

func systemTotpDisable(c *gin.Context, st *state.AppState) error {
	username := c.GetString("username")
	var req totpDisableReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("err.requestFailed")
	}
	st.UsersMu.Lock()
	var u *state.StoredUser
	for i := range st.Users {
		if st.Users[i].Username == username {
			u = &st.Users[i]
			break
		}
	}
	if u == nil {
		st.UsersMu.Unlock()
		return NewApiError(404, "user.notFound")
	}
	if !auth.VerifyPassword(req.Password, u.PasswordHash) {
		st.UsersMu.Unlock()
		return BadRequest("user.pwdWrong")
	}
	if u.TotpSecret == nil {
		st.UsersMu.Unlock()
		return BadRequest("totp.notEnabled")
	}
	if !auth.VerifyTotp(*u.TotpSecret, req.Code) {
		st.UsersMu.Unlock()
		return BadRequest("totp.codeWrong")
	}
	u.TotpSecret = nil
	st.UsersMu.Unlock()
	if err := st.SaveUsers(); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// ---------------- Docker 信息 / 事件 ---------------- 

func systemInfo(c *gin.Context, st *state.AppState) error {
	info, err := st.Docker.Info(c.Request.Context(), client.InfoOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, info)
	return nil
}

func systemEventsWS(c *gin.Context, st *state.AppState) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch := st.Events.Subscribe()
	defer st.Events.Unsubscribe(ch)
	_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"connected"}`))
	for {
		select {
		case m, ok := <-ch:
			if !ok {
				return nil
			}
			payload, _ := json.Marshal(map[string]any{"type": "event", "data": state.EventToValue(m)})
			if conn.WriteMessage(websocket.TextMessage, payload) != nil {
				return nil
			}
		case <-time.After(30 * time.Second):
			// 心跳探测连接存活
			if conn.WriteMessage(websocket.PingMessage, nil) != nil {
				return nil
			}
		}
	}
}
