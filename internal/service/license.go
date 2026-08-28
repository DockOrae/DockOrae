package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// 内置签名密钥(离线校验;key 由授权方用同一密钥签发)
const licenseSecret = "docker-manager-go-license-v1"

func LicenseSign(payload string) string {
	mac := hmac.New(sha256.New, []byte(licenseSecret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// LicenseGenerateKey 生成 License Key(内部工具:user/type/exp 天)—— 仅测试与签发使用
func LicenseGenerateKey(user, licenseType string, expDays int64) string {
	exp := time.Now().Unix() + expDays*86400
	payload, _ := json.Marshal(map[string]any{"user": user, "type": licenseType, "exp": exp})
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + LicenseSign(encoded)
}

// LicenseVerifyKey 校验并解析 Key
func LicenseVerifyKey(key string) (map[string]any, bool) {
	key = strings.TrimSpace(key)
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return nil, false
	}
	payload, sig := parts[0], parts[1]
	if LicenseSign(payload) != sig {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, false
	}
	var info map[string]any
	if json.Unmarshal(raw, &info) != nil {
		return nil, false
	}
	// 状态:exp 过期 → expired
	if exp := int64(float64(numOr(info["exp"]))); exp > 0 && exp < time.Now().Unix() {
		info["status"] = "expired"
	} else {
		info["status"] = "active"
	}
	return info, true
}

func numOr(v any) float64 {
	if n, ok := v.(float64); ok {
		return n
	}
	return 0
}

func licensePath(st *state.AppState) string {
	return filepath.Join(st.Cfg.DataDir, "license.json")
}

func LicenseDeviceID(dataDir string) string {
	// 优先 /etc/machine-id;fallback:data 目录 + 主机名 hash
	if raw, err := os.ReadFile("/etc/machine-id"); err == nil {
		id := strings.TrimSpace(string(raw))
		if len(id) >= 12 {
			return id[:12]
		}
	}
	host, _ := os.Hostname()
	sum := sha256.Sum256([]byte(dataDir + "|" + host))
	return hex.EncodeToString(sum[:])[:12]
}

// LicenseActive 当前是否已激活(供功能限制检查)
func LicenseActive(st *state.AppState) bool {
	raw, err := os.ReadFile(licensePath(st))
	if err != nil {
		return false
	}
	var stored map[string]any
	if json.Unmarshal(raw, &stored) != nil {
		return false
	}
	info, ok := LicenseVerifyKey(strOr(stored["key"]))
	if !ok {
		return false
	}
	return strOr(info["status"]) == "active"
}

// LicenseInfo 查询授权状态
func LicenseInfo(st *state.AppState) map[string]any {
	deviceID := LicenseDeviceID(st.Cfg.DataDir)
	raw, err := os.ReadFile(licensePath(st))
	if err != nil {
		return map[string]any{"active": false, "key": "", "info": nil, "device_id": deviceID, "bound": false}
	}
	var stored map[string]any
	if json.Unmarshal(raw, &stored) != nil {
		return map[string]any{"active": false, "key": "", "info": nil, "device_id": deviceID, "bound": false}
	}
	key := strOr(stored["key"])
	info, ok := LicenseVerifyKey(key)
	if !ok {
		return map[string]any{"active": false, "key": key, "info": nil, "device_id": deviceID, "bound": false}
	}
	boundID := strOr(stored["device_id"])
	bound := boundID != "" && boundID == deviceID
	return map[string]any{
		"active":    strOr(info["status"]) == "active" && bound,
		"key":       key,
		"info":      info,
		"device_id": deviceID,
		"bound":     bound,
		"bound_to":  boundID,
	}
}

// LicenseDoActivate 激活核心逻辑(校验 key → 绑定本机)
func LicenseDoActivate(st *state.AppState, key string) (map[string]any, error) {
	if strings.TrimSpace(key) == "" {
		return nil, BadRequest("license.keyEmpty")
	}
	info, ok := LicenseVerifyKey(key)
	if !ok {
		return nil, BadRequest("license.invalid")
	}
	if strOr(info["status"]) == "expired" {
		return nil, BadRequest("license.expiredKey")
	}
	// 已绑定到其他设备则拒绝(1Panel 的节点绑定语义)
	deviceID := LicenseDeviceID(st.Cfg.DataDir)
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
	info["device_id"] = deviceID
	return info, nil
}

// LicenseDeactivate 解除激活(删除本地授权文件)
func LicenseDeactivate(st *state.AppState) error {
	path := licensePath(st)
	if _, err := os.Stat(path); err != nil {
		return BadRequest("license.notActivated")
	}
	return os.Remove(path)
}

// licenseDemoKey 演示/开发用 key(2100 年过期,永久授权)
func licenseDemoKey() string {
	return LicenseGenerateKey("demo", "pro", 2100*365)
}

// DemoKey 供 api 层使用(演示/开发用 key)
func DemoKey() string { return licenseDemoKey() }
