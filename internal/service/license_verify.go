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
	"sync"
	"time"

	"github.com/DockerManger/Docker_Manager_Go/internal/state"
)

// ---------- 在线授权闭环(客户端侧,V3) ----------
//
// 与 Docker_Manager_License 服务端契约一致(见 License Server 仓库 docs/integration.md §8):
//   - License Key 只用于首次激活/重新激活(Skill §13)
//   - 正常运行使用 Activation Token:本地 license.json 保存,绝不写入日志
//   - verify/deactivate 携带 timestamp + nonce 防重放(Skill §16)
//   - 服务端响应含 server_time,客户端据此维护 clock_offset(trusted_now),防本地时间作弊(Skill §14)
//   - 本地时钟回退(>5min)检测 → CLOCK_ROLLBACK_DETECTED,禁用 Pro(Skill §14)
//   - 服务端 minimum_client_version / blocked_versions 控制客户端版本(Skill §21)
//   - 激活必须在线成功(严格模式);周期验证每 24h;Grace Period 7 天由本地维护
//   - 验证带 10s 超时,独立后台任务,绝不阻塞面板主流程
//
// 升级兼容:新客户端 + 旧服务端(未升级窗口)时,服务端返回 "key is required",
// 客户端自动回退旧格式(key + activation_id)请求。

const (
	licenseVerifyInterval  = 24 * time.Hour     // 定期验证间隔
	licenseGracePeriod     = 7 * 24 * time.Hour // 宽限期:最后一次成功验证后 7 天
	licenseRemoteTimeout   = 10 * time.Second   // 远程调用超时(不阻塞主流程)
	licenseVerifyPath      = "/api/v1/public"   // 服务端公开 API 前缀
	clockRollbackThreshold = 5 * time.Minute    // 本地时钟回退判定阈值(Skill §14:5 分钟)
	replayNonceBytes       = 32                 // nonce 长度(32 字节随机 → 64 hex)
)

// onlineState 在线验证状态(前端展示 + 门控依据)。
const (
	onlineOffline        = "offline"         // 未配置授权服务器:纯离线模式
	onlineVerified       = "verified"        // 最近验证成功(24h 内)
	onlineGrace          = "grace"           // 验证未成功但宽限期内(24h~7天)
	onlineGraceExpired   = "grace_expired"   // 超过宽限期仍未验证成功 → 禁用 Pro
	onlineRevoked        = "revoked"         // 服务端吊销/过期/设备无效 → 禁用 Pro
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

// ---------- license.json 存储(V3 在线字段扩展) ----------
//
// 存储结构:
//
//	{
//	  "key": "...", "device_id": "...", "activated_at": 123,
//	  "activation_id": "ACT-...",           // 激活展示 ID
//	  "activation_token": "...",            // 激活凭据(V3;绝不写入日志)
//	  "last_successful_verify": 123,        // 最近一次验证成功时间
//	  "verify_state": "",                   // 服务端判定的吊销/无效/封禁状态(非空 = 禁用)
//	  "server_url": "...",                  // 激活时的服务器地址(冗余,便于识别模式)
//	  "last_server_time": 123,              // 最近一次服务端时间(trusted_now 基准)
//	  "last_local_time": 123,               // 记录 last_server_time 时的本地时间
//	  "clock_offset": 0                     // server_time - local_time(防时间作弊)
//	}
//
// 敏感信息(activation_token)禁止写入日志;文件权限 0600。

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

// writeLicenseStore 原子写 license.json(临时文件 + rename),权限 0600。
func writeLicenseStore(st *state.AppState, m map[string]any) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	path := licensePath(st)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
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

// onlineStateOf 计算当前在线验证状态(不发起请求,纯本地判定)。
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
		// revoked / invalid / expired / blocked:服务端已判定禁用
		if vs == "update_required" {
			return onlineUpdateRequired
		}
		return onlineRevoked
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

// licenseOnlineAllowed 门控判断:在线模式下 revoked/grace_expired/version_blocked/clock_rollback 一律禁用 Pro。
func licenseOnlineAllowed(st *state.AppState) bool {
	switch onlineStateOf(st) {
	case onlineOffline, onlineVerified, onlineGrace, onlineUpdateRequired:
		return true
	default:
		return false // revoked / grace_expired / version_blocked / clock_rollback / never
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
// 旧服务端(未升级)不认新字段,仅需 key + device_id 即可(兼容)。
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

// licenseVerifyRemote 服务端验证(V3)。
// 优先 token 凭据;旧服务端返回 "key is required" 时回退旧格式(key + activation_id)。
// 返回 (status, 响应 map, error)。
func licenseVerifyRemote(serverURL, key, token, activationID, deviceID, productVersion string) (string, map[string]any, error) {
	now := time.Now().Unix()
	nonce := newNonce()
	// 新格式:activation_token + device_id + timestamp + nonce
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
		// 旧服务端(未升级):不认 activation_token,要求 key → 回退旧格式
		if code := serverErrorCode(err); code == "BAD_REQUEST" || code == "" {
			if token == "" && activationID != "" {
				return licenseVerifyRemoteLegacy(serverURL, key, activationID, deviceID, productVersion)
			}
		}
		return "", nil, err
	}
	if status != 200 {
		return "", nil, fmt.Errorf("verify failed: http %d", status)
	}
	return strOr(out["status"]), out, nil
}

// licenseVerifyRemoteLegacy 旧格式验证(key + activation_id + device_id;兼容未升级服务端)。
func licenseVerifyRemoteLegacy(serverURL, key, activationID, deviceID, productVersion string) (string, map[string]any, error) {
	var out map[string]any
	status, err := licensePostJSON(serverURL+licenseVerifyPath+"/verify", map[string]any{
		"key":             key,
		"activation_id":   activationID,
		"device_id":       deviceID,
		"product_version": productVersion,
	}, &out)
	if err != nil {
		return "", nil, err
	}
	if status != 200 {
		return "", nil, fmt.Errorf("verify failed: http %d", status)
	}
	return strOr(out["status"]), out, nil
}

// licenseDeactivateRemote 服务端解绑(尽力而为,失败不阻断本地解绑)。
// 优先 token 凭据;无 token 时用旧格式(key + activation_id)。
func licenseDeactivateRemote(serverURL, key, token, activationID, deviceID string) {
	now := time.Now().Unix()
	nonce := newNonce()
	if token != "" {
		_, _ = licensePostJSON(serverURL+licenseVerifyPath+"/deactivate", map[string]any{
			"activation_token": token,
			"device_id":        deviceID,
			"timestamp":        now,
			"nonce":            nonce,
		}, nil)
		return
	}
	if key != "" && activationID != "" {
		_, _ = licensePostJSON(serverURL+licenseVerifyPath+"/deactivate", map[string]any{
			"key":           key,
			"activation_id": activationID,
			"device_id":     deviceID,
		}, nil)
	}
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
	key, token, activationID, deviceID, ok := loadLicenseCred(st)
	if !ok {
		return map[string]any{"mode": "online", "state": "", "error": "not activated"}
	}
	now := time.Now().Unix()

	// 本地时钟回退检测(先于远程请求;回退 = 时间作弊嫌疑 → 禁用 Pro)
	if m, ok := readLicenseStore(st); ok && clockRollbackDetected(m) {
		m["verify_state"] = "clock_rollback"
		_ = writeLicenseStore(st, m)
		return map[string]any{"mode": "online", "state": onlineClockRollback, "verify_state": "clock_rollback"}
	}

	status, res, err := licenseVerifyRemote(serverURL, key, token, activationID, deviceID, DisplayVersion())
	if err != nil {
		// 网络/服务不可达:不动 last_successful_verify(Grace 自然推进)
		return map[string]any{"mode": "online", "state": onlineStateOf(st), "error": err.Error()}
	}
	switch status {
	case "valid":
		if m, ok := readLicenseStore(st); ok {
			m["last_successful_verify"] = now
			m["verify_state"] = ""
			applyServerTime(m, res)
			_ = writeLicenseStore(st, m)
		}
		// 版本控制:minimum_client_version 高于当前版本 → UPDATE_REQUIRED(提示,不封禁)
		// 开发构建(unknown)不参与版本控制
		if cv := DisplayVersion(); cv != "unknown" {
			if minVer := strOr(res["minimum_client_version"]); minVer != "" && versionLess(cv, minVer) {
				if m, ok := readLicenseStore(st); ok {
					m["verify_state"] = "update_required"
					_ = writeLicenseStore(st, m)
				}
				return map[string]any{"mode": "online", "state": onlineUpdateRequired, "last_verify": now, "minimum_client_version": minVer}
			}
		}
		return map[string]any{"mode": "online", "state": onlineVerified, "last_verify": now}
	case "blocked":
		// 版本被封禁 → 禁用 Pro
		if m, ok := readLicenseStore(st); ok {
			m["verify_state"] = "blocked"
			m["revoked_at"] = now
			applyServerTime(m, res)
			_ = writeLicenseStore(st, m)
		}
		return map[string]any{"mode": "online", "state": onlineVersionBlocked, "verify_state": "blocked", "revoked_at": now}
	case "revoked", "expired", "invalid":
		// 吊销/过期/设备无效 → 本地标记禁用(下一次门控判断即生效)
		if m, ok := readLicenseStore(st); ok {
			m["verify_state"] = status
			m["revoked_at"] = now
			applyServerTime(m, res)
			_ = writeLicenseStore(st, m)
		}
		return map[string]any{"mode": "online", "state": onlineRevoked, "verify_state": status, "revoked_at": now}
	default:
		return map[string]any{"mode": "online", "state": onlineStateOf(st), "error": "unexpected status: " + status}
	}
}

// applyServerTime 根据服务端返回的 server_time 更新 clock_offset(Skill §14)。
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
