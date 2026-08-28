package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// 内置签名密钥(离线校验;key 由授权方用同一密钥签发)
const licenseSecret = "docker-manager-go-license-v1"

func licenseSign(payload string) string {
	mac := hmac.New(sha256.New, []byte(licenseSecret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// licenseGenerateKey 生成 License Key(内部工具:user/type/exp 天)—— 仅测试与签发使用
func licenseGenerateKey(user, licenseType string, expDays int64) string {
	exp := time.Now().Unix() + expDays*86400
	payload, _ := json.Marshal(map[string]any{"user": user, "type": licenseType, "exp": exp})
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + licenseSign(encoded)
}

// licenseVerifyKey 校验并解析 Key
func licenseVerifyKey(key string) (map[string]any, bool) {
	key = strings.TrimSpace(key)
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return nil, false
	}
	if licenseSign(parts[0]) != parts[1] {
		return nil, false
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, false
	}
	var payload map[string]any
	if json.Unmarshal(payloadBytes, &payload) != nil {
		return nil, false
	}
	exp, ok := payload["exp"].(float64)
	if !ok {
		return nil, false
	}
	status := "active"
	if int64(exp) < time.Now().Unix() {
		status = "expired"
	}
	info := map[string]any{
		"user":   strOr(payload["user"]),
		"type":   strOr(payload["type"]),
		"exp":    int64(exp),
		"status": status,
	}
	return info, true
}

func strOr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func licensePath(st *state.AppState) string {
	return filepath.Join(st.Cfg.DataDir, "license.json")
}

// licenseDeviceID 本机设备标识(绑定用)
// Linux 裸机用 hostname;Docker 容器里 hostname 每次重建都会变(随机容器 ID),
// 会导致重建后误报"已绑定到其他设备" —— 改用数据目录的 (dev, inode):
// 挂载卷(如 /data)在宿主机上稳定,容器重建不受影响。
func licenseDeviceID(dataDir string) string {
	if runtime.GOOS == "windows" {
		if c := os.Getenv("COMPUTERNAME"); c != "" {
			return c
		}
		return "localhost"
	}
	if id, ok := dataDirDevID(dataDir); ok {
		return id + "@docker-manager"
	}
	host, _ := os.Hostname()
	return host + "@docker-manager"
}

// licenseActive 当前是否已激活(供功能限制检查)
func licenseActive(st *state.AppState) bool {
	raw, err := os.ReadFile(licensePath(st))
	if err != nil {
		return false
	}
	var stored map[string]any
	if json.Unmarshal(raw, &stored) != nil {
		return false
	}
	info, ok := licenseVerifyKey(strOr(stored["key"]))
	if !ok {
		return false
	}
	return strOr(info["status"]) == "active"
}

// licenseGet 查询授权状态
func licenseGet(c *gin.Context, st *state.AppState) error {
	deviceID := licenseDeviceID(st.Cfg.DataDir)
	raw, err := os.ReadFile(licensePath(st))
	if err != nil {
		c.JSON(200, gin.H{"active": false, "key": "", "info": nil, "device_id": deviceID, "bound": false})
		return nil
	}
	var stored map[string]any
	if json.Unmarshal(raw, &stored) != nil {
		c.JSON(200, gin.H{"active": false, "key": "", "info": nil, "device_id": deviceID, "bound": false})
		return nil
	}
	key := strOr(stored["key"])
	info, ok := licenseVerifyKey(key)
	if !ok {
		c.JSON(200, gin.H{"active": false, "key": key, "info": nil, "device_id": deviceID, "bound": false})
		return nil
	}
	boundID := strOr(stored["device_id"])
	bound := boundID != "" && boundID == deviceID
	c.JSON(200, gin.H{
		"active":    strOr(info["status"]) == "active" && bound,
		"key":       key,
		"info":      info,
		"device_id": deviceID,
		"bound":     bound,
		"bound_to":  boundID,
	})
	return nil
}

// licenseDoActivate 激活核心逻辑(校验 key → 绑定本机)
func licenseDoActivate(st *state.AppState, key string) (map[string]any, *ApiError) {
	if strings.TrimSpace(key) == "" {
		return nil, BadRequest("license.keyEmpty")
	}
	info, ok := licenseVerifyKey(key)
	if !ok {
		return nil, BadRequest("license.invalid")
	}
	if strOr(info["status"]) == "expired" {
		return nil, BadRequest("license.expiredKey")
	}
	// 已绑定到其他设备则拒绝(1Panel 的节点绑定语义)
	deviceID := licenseDeviceID(st.Cfg.DataDir)
	path := licensePath(st)
	if raw, err := os.ReadFile(path); err == nil {
		var existing map[string]any
		if json.Unmarshal(raw, &existing) == nil {
			boundID := strOr(existing["device_id"])
			if boundID != "" && boundID != deviceID {
				return nil, BadRequest("license.boundElsewhere")
			}
		}
	}
	out, _ := json.MarshalIndent(map[string]any{
		"key":          strings.TrimSpace(key),
		"device_id":    deviceID,
		"activated_at": time.Now().Unix(),
	}, "", "  ")
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return nil, BadRequest("write failed: " + err.Error())
	}
	return gin.H{"ok": true, "info": info, "device_id": deviceID}, nil
}

// licenseActivate 激活许可证(绑定到本机)
func licenseActivate(c *gin.Context, st *state.AppState) error {
	var payload struct {
		Key string `json:"key"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		return BadRequest("err.requestFailed")
	}
	resp, ae := licenseDoActivate(st, payload.Key)
	if ae != nil {
		return ae
	}
	c.JSON(200, resp)
	return nil
}

// licenseActivateFile 上传许可文件激活(1Panel 的"添加许可证 → 上传授权文件")
func licenseActivateFile(c *gin.Context, st *state.AppState) error {
	file, err := c.FormFile("file")
	if err != nil {
		return BadRequest("license.fileEmpty")
	}
	f, err := file.Open()
	if err != nil {
		return BadRequest("license.fileEmpty")
	}
	defer f.Close()
	buf := make([]byte, 64*1024)
	n, _ := f.Read(buf)
	key := strings.TrimSpace(string(buf[:n]))
	if key == "" {
		return BadRequest("license.fileEmpty")
	}
	resp, ae := licenseDoActivate(st, key)
	if ae != nil {
		return ae
	}
	c.JSON(200, resp)
	return nil
}

// licenseDeactivate 解绑并删除授权(1Panel 的解绑/删除语义)
func licenseDeactivate(c *gin.Context, st *state.AppState) error {
	path := licensePath(st)
	if _, err := os.Stat(path); err == nil {
		_ = os.Remove(path)
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// licenseDemoKey 生成永久授权 Key(内部签发工具,经授权渠道交付;exp = 2100 年 ≈ 永久)
func licenseDemoKey(c *gin.Context, st *state.AppState) error {
	key := licenseGenerateKey("MinimaxFlora", "pro", 0)
	// 覆盖为固定 2100-01-01
	exp := int64(4102444800)
	payload, _ := json.Marshal(map[string]any{"user": "MinimaxFlora", "type": "pro", "exp": exp})
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	key = encoded + "." + licenseSign(encoded)
	info, _ := licenseVerifyKey(key)
	c.JSON(200, gin.H{"key": key, "info": info})
	return nil
}
