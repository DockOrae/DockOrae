package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// ---------- 在线授权闭环(客户端侧) ----------
//
// 与 Docker_Manager_License 服务端契约一致(见 License Server 仓库 docs/integration.md §8):
//   - 激活/验证/解绑一律携带完整 License Key,服务端验签后自行解析
//   - 激活必须在线成功(严格模式);周期验证每 24h;Grace Period 7 天由本地维护
//   - 验证带 10s 超时,独立后台任务,绝不阻塞面板主流程
//
// 接入模式:设置页「授权服务器 URL」留空 = 纯离线激活(存量行为不变);
// 填入 URL = 在线闭环(激活/验证/解绑/吊销触达)。

const (
	licenseVerifyInterval = 24 * time.Hour     // 定期验证间隔
	licenseGracePeriod    = 7 * 24 * time.Hour // 宽限期:最后一次成功验证后 7 天
	licenseRemoteTimeout  = 10 * time.Second   // 远程调用超时(不阻塞主流程)
	licenseVerifyPath     = "/api/v1/public"   // 服务端公开 API 前缀
)

// onlineState 在线验证状态(前端展示 + 门控依据)。
const (
	onlineOffline      = "offline"       // 未配置授权服务器:纯离线模式
	onlineVerified     = "verified"      // 最近验证成功(24h 内)
	onlineGrace        = "grace"         // 验证未成功但宽限期内(24h~7天)
	onlineGraceExpired = "grace_expired" // 超过宽限期仍未验证成功 → 禁用 Pro
	onlineRevoked      = "revoked"       // 服务端吊销/过期/设备无效 → 禁用 Pro
	onlineNever        = "never"         // 在线模式但从未验证成功(防御)
)

// serverURLOf 当前授权服务器地址(所有部署统一走在线授权闭环)。
//
// 优先级:环境变量 DM_LICENSE_SERVER_URL(本地开发/自建服务器覆盖)→ 内置固定域名。
// 环境变量显式设为空字符串 = 离线模式(本地调试/无授权服务器场景)。
func serverURLOf(st *state.AppState) string {
	if v, ok := os.LookupEnv("DM_LICENSE_SERVER_URL"); ok {
		return strings.TrimSpace(strings.TrimSuffix(v, "/"))
	}
	return defaultLicenseServerURL
}

// defaultLicenseServerURL 官方授权服务器固定地址。
// License Server 与面板同域部署,由 nginx 将 /license-api/ 反代到 License Server 容器(:3000)。
const defaultLicenseServerURL = "https://manager.kejizero.xyz/license-api"

// ---------- license.json 存储(在线字段扩展) ----------
//
// 存储结构:
//
//	{
//	  "key": "...", "device_id": "...", "activated_at": 123,
//	  "activation_id": "...",            // 在线激活返回的凭据
//	  "last_successful_verify": 123,     // 最近一次验证成功时间
//	  "verify_state": "",                // 服务端判定的吊销/无效状态(非空 = 禁用)
//	  "server_url": "..."                // 激活时的服务器地址(冗余,便于识别模式)
//	}

func readLicenseStore(st *state.AppState) (map[string]any, bool) {
	raw, err := os.ReadFile(licensePath(st))
	if err != nil {
		return nil, false
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil, false
	}
	return m, true
}

// writeLicenseStore 原子写 license.json(临时文件 + rename)。
func writeLicenseStore(st *state.AppState, m map[string]any) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	path := licensePath(st)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// loadLicenseCred 提取在线验证所需凭据(key/activation_id/device_id)。
func loadLicenseCred(st *state.AppState) (key, activationID, deviceID string, ok bool) {
	m, ok := readLicenseStore(st)
	if !ok {
		return "", "", "", false
	}
	key = strOr(m["key"])
	deviceID = strOr(m["device_id"])
	if key == "" || deviceID == "" {
		return "", "", "", false
	}
	activationID = strOr(m["activation_id"])
	return key, activationID, deviceID, true
}

// onlineStateOf 计算当前在线验证状态(不发起请求,纯本地判定)。
func onlineStateOf(st *state.AppState) string {
	if serverURLOf(st) == "" {
		return onlineOffline
	}
	m, ok := readLicenseStore(st)
	if !ok {
		return ""
	}
	if vs := strOr(m["verify_state"]); vs != "" {
		return vs // revoked / invalid / expired:服务端已判定禁用
	}
	last := int64(float64(numOr(m["last_successful_verify"])))
	now := time.Now().Unix()
	if last <= 0 {
		return onlineNever
	}
	if now-last <= int64(licenseVerifyInterval.Seconds()) {
		return onlineVerified
	}
	if now-last <= int64(licenseGracePeriod.Seconds()) {
		return onlineGrace
	}
	return onlineGraceExpired
}

// licenseOnlineAllowed 门控判断:在线模式下 revoked/grace_expired 一律禁用 Pro。
func licenseOnlineAllowed(st *state.AppState) bool {
	switch onlineStateOf(st) {
	case onlineOffline, onlineVerified, onlineGrace:
		return true
	default:
		return false // revoked / grace_expired / never
	}
}

// ---------- 远程调用(License Server) ----------

// licensePostJSON POST JSON 到授权服务器,解析统一错误体。
// 返回 (http 状态码, 解析后的响应对象, error)。网络错误返回 error。
func licensePostJSON(url string, body map[string]any, out any) (int, error) {
	raw, _ := json.Marshal(body)
	client := &http.Client{Timeout: licenseRemoteTimeout}
	resp, err := client.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		var eb struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(data, &eb)
		return resp.StatusCode, fmt.Errorf("license server: %s: %s", eb.Error.Code, eb.Error.Message)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.StatusCode, fmt.Errorf("license server: bad response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

// licenseActivateRemote 服务端激活,返回 (响应 map, 服务端错误码, error)。
// 网络/超时错误 → error(code 为空);服务端拒绝(4xx)→ code 非空 + error。
func licenseActivateRemote(serverURL, key, deviceID, productVersion string) (map[string]any, string, error) {
	var out map[string]any
	_, err := licensePostJSON(serverURL+licenseVerifyPath+"/activate", map[string]any{
		"key":             key,
		"device_id":       deviceID,
		"product_version": productVersion,
	}, &out)
	if err != nil {
		return nil, serverErrorCode(err), err // 4xx:code 从错误消息提取;网络错误:code 为空
	}
	return out, "", nil
}

// serverErrorCode 从 licensePostJSON 的错误消息中提取服务端错误码(如 DEVICE_LIMIT_REACHED)。
func serverErrorCode(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	const p = "license server: "
	if strings.HasPrefix(msg, p) {
		if rest := strings.TrimPrefix(msg, p); strings.Contains(rest, ":") {
			return strings.TrimSpace(strings.SplitN(rest, ":", 2)[0])
		}
	}
	return ""
}

// licenseVerifyRemote 服务端验证,返回 status(valid/revoked/expired/invalid)。
func licenseVerifyRemote(serverURL, key, activationID, deviceID, productVersion string) (string, error) {
	var out map[string]any
	status, err := licensePostJSON(serverURL+licenseVerifyPath+"/verify", map[string]any{
		"key":             key,
		"activation_id":   activationID,
		"device_id":       deviceID,
		"product_version": productVersion,
	}, &out)
	if err != nil {
		return "", err
	}
	if status != 200 {
		return "", fmt.Errorf("verify failed: http %d", status)
	}
	return strOr(out["status"]), nil
}

// licenseDeactivateRemote 服务端解绑(尽力而为,失败不阻断本地解绑)。
func licenseDeactivateRemote(serverURL, key, activationID, deviceID string) {
	_, _ = licensePostJSON(serverURL+licenseVerifyPath+"/deactivate", map[string]any{
		"key":           key,
		"activation_id": activationID,
		"device_id":     deviceID,
	}, nil)
}

// ---------- 周期验证器 ----------

// LicenseVerifier 周期验证后台任务:启动 5s 后首次验证,之后每 24h。
// 独立 goroutine,10s 超时,失败不重试(等下一周期,Grace 自然推进)。
type LicenseVerifier struct {
	st   *state.AppState
	done chan struct{}
	mu   sync.Mutex // 防手动 + 周期并发
}

// StartLicenseVerifier 启动验证器(在 state 初始化后调用)。
func StartLicenseVerifier(st *state.AppState) *LicenseVerifier {
	v := &LicenseVerifier{st: st, done: make(chan struct{})}
	go v.loop()
	return v
}

// Stop 停止验证器。
func (v *LicenseVerifier) Stop() {
	select {
	case <-v.done:
	default:
		close(v.done)
	}
}

func (v *LicenseVerifier) loop() {
	timer := time.NewTimer(5 * time.Second) // 启动后首次验证(不阻塞面板启动)
	defer timer.Stop()
	tick := time.NewTicker(licenseVerifyInterval)
	defer tick.Stop()
	for {
		select {
		case <-timer.C:
			_ = v.verifyOnce()
		case <-tick.C:
			_ = v.verifyOnce()
		case <-v.done:
			return
		}
	}
}

// verifyOnce 执行一次验证(手动与周期共用)。
func (v *LicenseVerifier) verifyOnce() map[string]any {
	return VerifyNow(v.st)
}

// VerifyNow 立即执行一次在线验证(手动触发 API 与周期任务共用)。
// 未配置服务器 / 未激活时不发请求,直接返回当前状态。
func VerifyNow(st *state.AppState) map[string]any {
	serverURL := serverURLOf(st)
	if serverURL == "" {
		return map[string]any{"mode": "offline", "state": onlineOffline}
	}
	key, activationID, deviceID, ok := loadLicenseCred(st)
	if !ok {
		return map[string]any{"mode": "online", "state": "", "error": "not activated"}
	}
	status, err := licenseVerifyRemote(serverURL, key, activationID, deviceID, DisplayVersion())
	now := time.Now().Unix()
	if err != nil {
		// 网络/服务不可达:不动 last_successful_verify(Grace 自然推进)
		return map[string]any{"mode": "online", "state": onlineStateOf(st), "error": err.Error()}
	}
	switch status {
	case "valid":
		if m, ok := readLicenseStore(st); ok {
			m["last_successful_verify"] = now
			m["verify_state"] = ""
			_ = writeLicenseStore(st, m)
		}
		return map[string]any{"mode": "online", "state": onlineVerified, "last_verify": now}
	case "revoked", "expired", "invalid":
		// 吊销/过期/设备无效 → 本地标记禁用(下一次门控判断即生效)
		if m, ok := readLicenseStore(st); ok {
			m["verify_state"] = status
			m["revoked_at"] = now
			_ = writeLicenseStore(st, m)
		}
		return map[string]any{"mode": "online", "state": onlineRevoked, "verify_state": status, "revoked_at": now}
	default:
		return map[string]any{"mode": "online", "state": onlineStateOf(st), "error": "unexpected status: " + status}
	}
}
