package service

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/DockOrae/DockOrae/internal/state"
)

// ---------- 在线授权闭环(客户端侧,V3 Event-Driven) ----------
//
// 与 Docker_Manager_License 服务端契约一致(见 License Server 仓库 docs/integration.md):
//   - License Key 只用于首次激活/重新激活
//   - 正常运行使用 Activation Token:本地 license.json 保存,绝不写入日志
//   - verify/deactivate 携带 timestamp + nonce 防重放
//   - 服务端响应含 server_time,客户端据此维护 clock_offset(trusted_now),防本地时间作弊
//   - 本地时钟回退(>5min)检测 → CLOCK_ROLLBACK_DETECTED,禁用 Pro
//   - 服务端 minimum_client_version / blocked_versions 控制客户端版本
//   - 激活必须在线成功(严格模式)
//
// 同步模型(V3):状态变化由 Server 经 SSE 主动推送(Event-Driven),
// 客户端收到事件后 Verify 获取权威状态 —— 禁止任何周期性 Verify / Heartbeat / Check-in / Lease。
// V2/Legacy 完全移除:不存在 key + activation_id 兼容路径,不存在 V3→V2 fallback。

const (
	licenseRemoteTimeout   = 10 * time.Second // 远程调用超时(不阻塞主流程)
	licenseVerifyPath      = "/api/v3"        // 服务端公开 API V3 前缀
	clockRollbackThreshold = 5 * time.Minute  // 本地时钟回退判定阈值(5 分钟)
	replayNonceBytes       = 32               // nonce 长度(32 字节随机 → 64 hex)
)

// onlineState 在线验证状态(前端展示 + 门控依据)。
const (
	onlineOffline        = "offline"         // 未配置授权服务器:纯离线模式
	onlineVerified       = "verified"        // 最近验证成功(在线)
	onlineGrace          = "grace"           // Server 不可达但宽限期内(授权保留)
	onlineGraceExpired   = "grace_expired"   // 超过宽限期 → 禁用 Pro
	onlineUnbound        = "unbound"         // 许可证已解绑(用户/管理员),License 仍有效,可重新激活
	onlineRevoked        = "revoked"         // 服务端吊销/无效 → 禁用 Pro
	onlineVersionBlocked = "version_blocked" // 客户端版本被服务端封禁 → 禁用 Pro
	onlineUpdateRequired = "update_required" // 客户端版本低于 minimum_client_version → 提示升级
	onlineClockRollback  = "clock_rollback"  // 检测到本地时钟回退 → 禁用 Pro
	onlineNever          = "never"           // 在线模式但从未验证成功(防御)
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

// ---------- license.json 存储(V3 字段;写入统一走 LicenseStateManager) ----------
//
// 存储结构:
//
//	{
//	  "key": "...", "device_id": "...", "activated_at": 123,
//	  "activation_id": "ACT-...",           // 激活展示 ID
//	  "activation_token": "...",            // 激活凭据(V3;绝不写入日志)
//	  "last_successful_verify": 123,        // 最近一次验证成功时间(Grace 基准)
//	  "verify_state": "",                   // 服务端判定:revoked/expired/invalid/blocked/...
//	  "sync_state": "",                     // 在线同步状态:online/offline/grace/grace_expired/...
//	  "last_event_id": "evt_N",             // 最近处理的事件 ID(SSE Replay)
//	  "state_version": 1,                   // Server 权威状态版本(乱序保护)
//	  "grace_deadline": 123,                // 宽限截止时间
//	  "server_url": "...", "clock_offset": 0, ...
//	}
//
// 敏感信息(activation_token)禁止写入日志;文件权限 0600。

// readLicenseStore 读取 license.json(只读;所有写入必须经过 LicenseStateManager)。
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

// writeLicenseStore 直接写 license.json(测试/兼容路径;生产代码统一用 LicenseStateManager)。
// 同样采用 临时文件 + fsync + rename 原子写入。
func writeLicenseStore(st *state.AppState, m map[string]any) error {
	return atomicWriteLicenseFile(st, m)
}

// loadLicenseCred 提取在线验证所需凭据(key/activation_token/activation_id/device_id)。
func loadLicenseCred(st *state.AppState) (key, token, activationID, deviceID string, ok bool) {
	m, ok := readLicenseStore(st)
	if !ok {
		return "", "", "", "", false
	}
	key = strOr(m["key"])
	token = strOr(m["activation_token"])
	activationID = strOr(m["activation_id"])
	deviceID = strOr(m["device_id"])
	if key == "" || deviceID == "" {
		return "", "", "", "", false
	}
	return key, token, activationID, deviceID, true
}

// trustedNow 可信当前时间 = 本地时间 + clock_offset(防本地时间作弊)。
// 无 offset 记录时回退本地时间。
func trustedNow(st *state.AppState) int64 {
	m, ok := readLicenseStore(st)
	if !ok {
		return time.Now().Unix()
	}
	offset := int64(float64(numOr(m["clock_offset"])))
	return time.Now().Unix() + offset
}

// clockRollbackDetected 检测本地时钟回退(本地时间比上次记录倒退超过阈值)。
func clockRollbackDetected(m map[string]any) bool {
	lastLocal := int64(float64(numOr(m["last_local_time"])))
	if lastLocal <= 0 {
		return false
	}
	return time.Now().Unix() < lastLocal-int64(clockRollbackThreshold.Seconds())
}

// onlineStateOf 当前在线验证状态(纯本地判定,不发起请求)。
// V3:直接读取 sync_state(LicenseSync 引擎维护,SSE 事件驱动更新)。
func onlineStateOf(st *state.AppState) string {
	if serverURLOf(st) == "" {
		return onlineOffline
	}
	m, ok := readLicenseStore(st)
	if !ok {
		return ""
	}
	// 本地时钟回退检测优先(即使 verify_state 为空也要拦)
	if clockRollbackDetected(m) {
		return onlineClockRollback
	}
	if vs := strOr(m["verify_state"]); vs != "" {
		if vs == "update_required" {
			return onlineUpdateRequired
		}
		if vs == "grace_expired" {
			return onlineGraceExpired
		}
		if vs == "clock_rollback" {
			return onlineClockRollback
		}
		if vs == "unbound" {
			return onlineUnbound
		}
		return onlineRevoked
	}
	switch ss := strOr(m["sync_state"]); ss {
	case "online", "server_recovered":
		return onlineVerified
	case "grace":
		return onlineGrace
	case "grace_expired":
		return onlineGraceExpired
	case "blocked":
		return onlineVersionBlocked
	case "revoked":
		return onlineRevoked
	case "unbound":
		return onlineUnbound
	}
	last := int64(float64(numOr(m["last_successful_verify"])))
	if last <= 0 {
		return onlineNever
	}
	return onlineVerified // 有成功记录但状态未写入(兼容旧文件)
}

// licenseOnlineAllowed 门控判断:在线模式下 revoked/grace_expired/version_blocked/clock_rollback/unbound 一律禁用 Pro。
// unbound = 已解绑(未激活),本就没有授权。
func licenseOnlineAllowed(st *state.AppState) bool {
	switch onlineStateOf(st) {
	case onlineOffline, onlineVerified, onlineGrace, onlineUpdateRequired:
		return true
	default:
		return false // revoked / unbound / grace_expired / version_blocked / clock_rollback / never
	}
}

// ---------- 远程调用(License Server,V3 token 唯一凭据) ----------

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

// newNonce 生成重放防护 nonce(32 字节随机 hex)。
func newNonce() string {
	b := make([]byte, replayNonceBytes)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 失败为致命错误;回退时间戳 + 随机(理论上不可达)
		return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
	}
	return hex.EncodeToString(b)
}

// licenseActivateRemote 服务端激活(V3),返回 (响应 map, 服务端错误码, error)。
// 网络/超时错误 → error(code 为空);服务端拒绝(4xx)→ code 非空 + error。
func licenseActivateRemote(serverURL, key, deviceID, productVersion, deviceFingerprint, platform, architecture string) (map[string]any, string, error) {
	var out map[string]any
	_, err := licensePostJSON(serverURL+licenseVerifyPath+"/activate", map[string]any{
		"key":                key,
		"device_id":          deviceID,
		"product_version":    productVersion,
		"device_fingerprint": deviceFingerprint,
		"platform":           platform,
		"architecture":       architecture,
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

// licenseVerifyRemote 服务端验证(V3,唯一路径:activation_token)。
// 返回 (status, 响应 map, error)。无 Legacy 回退 —— V3 失败就是 V3 错误。
func licenseVerifyRemote(serverURL, token, deviceID, productVersion string) (string, map[string]any, error) {
	now := time.Now().Unix()
	nonce := newNonce()
	body := map[string]any{
		"activation_token": token,
		"device_id":        deviceID,
		"product_version":  productVersion,
		"timestamp":        now,
		"nonce":            nonce,
	}
	var out map[string]any
	status, err := licensePostJSON(serverURL+licenseVerifyPath+"/verify", body, &out)
	if err != nil {
		return "", nil, err
	}
	if status != 200 {
		return "", nil, fmt.Errorf("verify failed: http %d", status)
	}
	return strOr(out["status"]), out, nil
}

// licenseDeactivateRemote 服务端解绑(尽力而为,失败不阻断本地解绑;token 唯一凭据)。
func licenseDeactivateRemote(serverURL, token, deviceID string) {
	now := time.Now().Unix()
	nonce := newNonce()
	_, _ = licensePostJSON(serverURL+licenseVerifyPath+"/deactivate", map[string]any{
		"activation_token": token,
		"device_id":        deviceID,
		"timestamp":        now,
		"nonce":            nonce,
	}, nil)
}

// applyServerTime 根据服务端返回的 server_time 更新 clock_offset(防本地时间作弊)。
// 仅在服务端时间合法(>0)且与本地时间差异合理(<24h)时更新,防异常数据污染。
func applyServerTime(m map[string]any, res map[string]any) {
	if res == nil {
		return
	}
	serverTime := int64(float64(numOr(res["server_time"])))
	if serverTime <= 0 {
		return
	}
	localNow := time.Now().Unix()
	diff := serverTime - localNow
	if diff > 86400 || diff < -86400 {
		return // 差异超 24h:数据异常,忽略
	}
	m["last_server_time"] = serverTime
	m["last_local_time"] = localNow
	m["clock_offset"] = diff
}

// versionLess 语义化版本比较:a < b。
// 非语义化(unknown/v0.0.0)按最老处理。
func versionLess(a, b string) bool {
	pa := parseVersion(a)
	pb := parseVersion(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	return false
}

// parseVersion 解析语义化版本号(v1.2.3 → [1,2,3]);解析失败返回 [0,0,0]。
func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	var out [3]int
	parts := strings.SplitN(v, ".", 3)
	for i := 0; i < len(parts) && i < 3; i++ {
		n := 0
		for _, ch := range parts[i] {
			if ch < '0' || ch > '9' {
				break
			}
			n = n*10 + int(ch-'0')
		}
		out[i] = n
	}
	return out
}
