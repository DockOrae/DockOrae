package service

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/DockOrae/DockOrae/internal/state"
)

// runtimeGOOS / runtimeGOARCH 供激活请求上报平台信息(独立函数便于测试注入)。
func runtimeGOOS() string   { return runtime.GOOS }
func runtimeGOARCH() string { return runtime.GOARCH }

// License 验证 — V2 Ed25519 唯一路径(V1 HMAC 已于 2026-08 完全移除,
// 不保留任何 V1 兼容分派;secret 已公开,旧 V1 Key 无法激活,需重新签发 V2)。
//
// V2 Key 由 Docker_Manager_License(授权方)签发,客户端只持有公钥即可离线验证 ——
// 源码完全公开也无法伪造合法 Key。
//
// key_id 支持密钥轮换:新增公钥在此追加,旧 key 继续有效;
// 未知 key_id 一律拒绝(UNSUPPORTED_KEY,不静默接受)。

// licensePublicKeys Ed25519 公钥注册表(key_id → PEM)。
// 由 Docker_Manager_License 签发端提供,替换公钥后同步更新。
var licensePublicKeys = map[string]string{
	// 固定公钥(与 Docker_Manager_License 仓库 private/license.key 配对,永久不变)。
	// 私钥由授权方保管(部署端 private/license.key),此公钥可安全公开。
	"2026-01": `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAdD7pzvzYQClRQC6AfDBed6vottConCnihO881v1A008=
-----END PUBLIC KEY-----`,
}

// licensePayload V2 License 载荷(与 Docker_Manager_License internal/license 契约一致)。
// 新增可选字段(customer_id/subscription_id)向前兼容:未知字段在 JSON 解析时天然忽略,
// 但未知 version 必须明确拒绝(UNSUPPORTED_LICENSE_VERSION),不得静默接受。
type licensePayload struct {
	Version        int      `json:"version"`
	KeyID          string   `json:"key_id"`
	LicenseID      string   `json:"license_id"`
	Product        string   `json:"product"`
	Plan           string   `json:"plan"`
	Features       []string `json:"features,omitempty"`
	Customer       string   `json:"customer"`
	CustomerID     string   `json:"customer_id,omitempty"`
	SubscriptionID string   `json:"subscription_id,omitempty"`
	IssuedAt       int64    `json:"issued_at"`
	ExpiresAt      int64    `json:"expires_at"`
	MaxDevices     int      `json:"max_devices"`
}

// licenseFeatureRegistry 与 Docker_Manager_License 完全一致的 Feature 集合。
// 未知 feature 一律拒绝(防止未来契约漂移)。
var licenseFeatureRegistry = []string{"compose", "container_create", "appstore"}

// LicenseVerifyKey 校验并解析 Key(V2 Ed25519 唯一路径)。
// 返回 map 兼容前端展示;解析失败返回 false。
func LicenseVerifyKey(key string) (map[string]any, bool) {
	p, ok := licenseVerifyV2(key)
	if !ok {
		return nil, false
	}
	// features 转 []any,兼容下游 strOr/[]any 断言与 JSON 序列化
	feats := make([]any, len(p.Features))
	for i, f := range p.Features {
		feats[i] = f
	}
	info := map[string]any{
		"version":         p.Version,
		"key_id":          p.KeyID,
		"license_id":      p.LicenseID,
		"product":         p.Product,
		"plan":            p.Plan,
		"features":        feats,
		"customer":        p.Customer,
		"customer_id":     p.CustomerID,
		"subscription_id": p.SubscriptionID,
		"issued_at":       p.IssuedAt,
		"expires_at":      p.ExpiresAt,
		"max_devices":     p.MaxDevices,
		"status":          "active",
	}
	if p.ExpiresAt > 0 && p.ExpiresAt < time.Now().Unix() {
		info["status"] = "expired"
	}
	return info, true
}

// licenseVerifyV2 校验 V2 Key:
//   - payload 必须 version=2(未知版本拒绝)
//   - key_id 必须在注册表(未知 key_id 拒绝)
//   - product 必须为 docker-manager-go
//   - plan / features 必须属于已注册集合(未知 feature 拒绝)
//   - issued_at / expires_at / max_devices 必须合法
//   - Ed25519 签名必须通过(篡改即失败)
func licenseVerifyV2(key string) (*licensePayload, bool) {
	key = strings.TrimSpace(key)
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, false
	}
	sigRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(sigRaw) != ed25519.SignatureSize {
		return nil, false
	}
	var p licensePayload
	if json.Unmarshal(raw, &p) != nil {
		return nil, false
	}
	// 未知版本:拒绝而非静默接受
	if p.Version != 2 {
		return nil, false
	}
	// 未知 key_id:拒绝
	pubPEM, ok := licensePublicKeys[p.KeyID]
	if !ok {
		return nil, false
	}
	pub, err := parseEd25519PublicKey([]byte(pubPEM))
	if err != nil {
		return nil, false
	}
	if !ed25519.Verify(pub, raw, sigRaw) {
		return nil, false
	}
	// 契约字段校验
	if p.Product != "docker-manager-go" {
		return nil, false
	}
	if p.Plan != "pro" {
		return nil, false // 当前只签发 pro;未知 plan 拒绝
	}
	for _, f := range p.Features {
		if !containsStr(licenseFeatureRegistry, f) {
			return nil, false // 未知 feature 拒绝
		}
	}
	if p.IssuedAt <= 0 || p.ExpiresAt <= 0 || p.MaxDevices < 1 {
		return nil, false
	}
	return &p, true
}

// parseEd25519PublicKey 解析 PEM 编码的 Ed25519 公钥(PKIX SubjectPublicKeyInfo)。
func parseEd25519PublicKey(pemBytes []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid public key PEM")
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := pubAny.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("not an ed25519 public key")
	}
	return pub, nil
}

func containsStr(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func licensePath(st *state.AppState) string {
	return filepath.Join(st.Cfg.DataDir, "license.json")
}

// LicenseDeviceID 本机设备唯一标识(在线激活/绑定的凭据)。
//
// 稳定性要求:容器重建/面板升级后必须保持不变,否则在线绑定失效。
//   - Linux:数据目录的 (dev, inode) 特征(挂载卷在宿主机稳定,推荐部署形态)
//   - Windows:COMPUTERNAME
//   - 兜底:机器 ID / (数据目录+主机名) hash
func LicenseDeviceID(dataDir string) string {
	if id, ok := dataDirDeviceID(dataDir); ok {
		return id
	}
	// 兜底:机器 ID 优先;否则 (dataDir|hostname) hash
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

func numOr(v any) float64 {
	if n, ok := v.(float64); ok {
		return n
	}
	return 0
}

// LicenseDeviceFingerprint 稳定设备指纹(Skill §18):
// SHA-256(稳定机器信息)hex,不依赖 MAC 地址。
// 组成:设备 ID(平台稳定特征)+ machine-id(如存在)+ 主机名。
// 容器重建/升级后保持不变(只要数据卷保留);用于服务端设备画像与异常检测。
func LicenseDeviceFingerprint(dataDir string) string {
	parts := make([]string, 0, 3)
	if id, ok := dataDirDeviceID(dataDir); ok {
		parts = append(parts, id)
	}
	if raw, err := os.ReadFile("/etc/machine-id"); err == nil {
		if id := strings.TrimSpace(string(raw)); len(id) >= 8 {
			parts = append(parts, id[:8])
		}
	}
	if host, err := os.Hostname(); err == nil && host != "" {
		parts = append(parts, host)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

// LicenseActive 当前是否已激活(供 UI 展示与整体判断)
func LicenseActive(st *state.AppState) bool {
	if b, ok := LicenseInfo(st)["active"].(bool); ok {
		return b
	}
	return false
}

// LicenseFeatureActive 指定功能是否可用(功能级门控)。
//
//   - 在线模式(配置了授权服务器):先过在线状态检查(吊销/宽限过期 → 全部禁用),再按 features 判断
//   - 离线模式:V2 Key 按 payload.features 精确控制(空 features = 无商业功能)
//
// Feature 名称(与 Docker_Manager_License 完全一致,勿改):
//   - compose           Compose 编排部署
//   - container_create  容器创建
//   - appstore          应用商店安装
func LicenseFeatureActive(st *state.AppState, feature string) bool {
	if !licenseOnlineAllowed(st) {
		return false // 在线模式 revoked / grace 过期 → 功能全部禁用
	}
	raw, err := os.ReadFile(licensePath(st))
	if err != nil {
		return false
	}
	var stored map[string]any
	if json.Unmarshal(raw, &stored) != nil {
		return false
	}
	info, ok := LicenseVerifyKey(strOr(stored["key"]))
	if !ok || strOr(info["status"]) != "active" {
		return false
	}
	// trusted_now 过期判断:本地时间 + clock_offset(防时间作弊绕过)
	if exp := int64(float64(numOr(info["expires_at"]))); exp > 0 && exp < trustedNow(st) {
		return false
	}
	feats, _ := info["features"].([]any)
	for _, f := range feats {
		if strOr(f) == feature {
			return true
		}
	}
	return false // 无该 feature(含空 features)→ 不授权
}

// LicenseInfo 查询授权状态(含在线验证状态,供前端展示)。
func LicenseInfo(st *state.AppState) map[string]any {
	deviceID := LicenseDeviceID(st.Cfg.DataDir)
	m, ok := readLicenseStore(st)
	if !ok {
		return map[string]any{"active": false, "key": "", "info": nil, "device_id": deviceID, "bound": false, "online": onlineInfo(st)}
	}
	key := strOr(m["key"])
	info, ok := LicenseVerifyKey(key)
	if !ok {
		return map[string]any{"active": false, "key": key, "info": nil, "device_id": deviceID, "bound": false, "online": onlineInfo(st)}
	}
	boundID := strOr(m["device_id"])
	// unbound(已解绑/未激活):保留 key 但不再视为绑定,active=false
	unbound := strOr(m["verify_state"]) == "unbound" || strOr(m["sync_state"]) == "unbound"
	bound := boundID != "" && boundID == deviceID && !unbound
	active := strOr(info["status"]) == "active" && bound && licenseOnlineAllowed(st)
	return map[string]any{
		"active":        active,
		"key":           key,
		"info":          info,
		"device_id":     deviceID,
		"bound":         bound,
		"bound_to":      boundID,
		"activation_id": strOr(m["activation_id"]),
		"online":        onlineInfo(st),
	}
}

// onlineInfo 在线验证状态详情(未配置服务器 = 离线模式)。
// V3:包含同步状态(sync_state)、事件游标(last_event_id)、权威版本(state_version)。
func onlineInfo(st *state.AppState) map[string]any {
	out := map[string]any{"mode": "offline", "state": onlineStateOf(st)}
	serverURL := serverURLOf(st)
	if serverURL == "" {
		return out
	}
	out["mode"] = "online"
	out["server_url"] = serverURL
	m, ok := readLicenseStore(st)
	if !ok {
		return out
	}
	if ss := strOr(m["sync_state"]); ss != "" {
		out["sync_state"] = ss
	}
	if lv := int64(float64(numOr(m["last_successful_verify"]))); lv > 0 {
		out["last_verify"] = lv
		out["grace_deadline"] = lv + int64(licenseGracePeriod().Seconds())
	}
	if vs := strOr(m["verify_state"]); vs != "" {
		out["verify_state"] = vs
	}
	if ra := int64(float64(numOr(m["revoked_at"]))); ra > 0 {
		out["revoked_at"] = ra
	}
	if le := strOr(m["last_event_id"]); le != "" {
		out["last_event_id"] = le
	}
	if sv := int64(float64(numOr(m["state_version"]))); sv > 0 {
		out["state_version"] = sv
	}
	// 解绑来源(admin_unbound / user_unbound),UI 据此区分提示文案
	if ur := strOr(m["last_unbind_reason"]); ur != "" {
		out["unbind_reason"] = ur
	}
	if us := strOr(m["unbind_source"]); us != "" {
		out["unbind_source"] = us
	}
	if ua := int64(float64(numOr(m["last_unbind_at"]))); ua > 0 {
		out["unbound_at"] = ua
	}
	return out
}

// LicenseDoActivate 激活核心逻辑(本地验签 → 绑定本机;在线模式必须激活成功)。
//
// 在线模式(配置了授权服务器):本地验签通过后,必须 POST 服务端 /activate 成功
// (设备绑定/上限/吊销由服务端权威判定;服务器不可达 → 拒绝激活,严格在线)。
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
	store := map[string]any{
		"key":          strings.TrimSpace(key),
		"device_id":    deviceID,
		"activated_at": time.Now().Unix(),
	}
	if serverURL := serverURLOf(st); serverURL != "" {
		// 在线激活(严格):服务器不可达/拒绝 → 激活失败
		res, code, err := licenseActivateRemote(serverURL, store["key"].(string), deviceID,
			DisplayVersion(), LicenseDeviceFingerprint(st.Cfg.DataDir), runtimeGOOS(), runtimeGOARCH())
		if err != nil {
			switch code {
			case "DEVICE_LIMIT_REACHED":
				return nil, BadRequest("license.deviceLimit")
			case "DEVICE_BOUND":
				// 本设备已绑定其他 License(一个 DMG 只能绑定一个许可证):需先解绑旧的
				return nil, BadRequest("license.deviceBound")
			case "LICENSE_REVOKED":
				return nil, BadRequest("license.revokedKey")
			case "LICENSE_EXPIRED":
				return nil, BadRequest("license.expiredKey")
			case "INVALID_SIGNATURE", "LICENSE_NOT_FOUND":
				return nil, BadRequest("license.invalid")
			default:
				return nil, BadRequest("license.serverUnreachable")
			}
		}
		activationID := strOr(res["activation_id"])
		if activationID == "" {
			return nil, BadRequest("license.serverError")
		}
		store["activation_id"] = activationID
		// V3:保存 Activation Token(数据库只存 hash;本地保存明文,绝不写入日志)
		if tok := strOr(res["activation_token"]); tok != "" {
			store["activation_token"] = tok
		}
		store["last_successful_verify"] = time.Now().Unix()
		store["server_url"] = serverURL
		// V3:server_time → clock_offset(防本地时间作弊)
		if serverTime := int64(float64(numOr(res["server_time"]))); serverTime > 0 {
			localNow := time.Now().Unix()
			diff := serverTime - localNow
			if diff <= 86400 && diff >= -86400 { // 差异超 24h 视为异常,忽略
				store["last_server_time"] = serverTime
				store["last_local_time"] = localNow
				store["clock_offset"] = diff
			}
		}
		// V3 Event-Driven:激活即获权威状态版本与同步状态
		if sv := int64(float64(numOr(res["state_version"]))); sv > 0 {
			store["state_version"] = sv
		}
		store["sync_state"] = string(SyncOnline)
		store["verify_state"] = ""
	}
	// 统一走 LicenseStateManager 原子写入(临时文件 + fsync + rename)
	mgr := NewLicenseStateManager(st)
	if err := mgr.UpdateBytes(store); err != nil {
		return nil, BadRequest("write failed: " + err.Error())
	}
	// 唤醒 SSE 循环用新凭据建立连接(激活后立即订阅事件流)
	if s := LicenseSyncInst(); s != nil {
		s.wakeSSE()
	}
	info["device_id"] = deviceID
	return info, nil
}

// LicenseDeactivate 解除激活(用户主动解绑)。
// 语义:只解除 Binding —— 保留 Key(可一键重新激活),清除本地凭据
// (activation_token/activation_id 已作废),状态置 unbound。
// 绝不删除 Key,绝不标记 revoked。在线模式先通知服务端解绑。
func LicenseDeactivate(st *state.AppState) error {
	m, ok := readLicenseStore(st)
	if !ok {
		return BadRequest("license.notActivated")
	}
	// 在线模式:先尝试服务端解绑(尽力而为,失败不阻断本地解绑;token 唯一凭据)
	serverURL := serverURLOf(st)
	token := strOr(m["activation_token"])
	deviceID := strOr(m["device_id"])
	if serverURL != "" && token != "" && deviceID != "" {
		licenseDeactivateRemote(serverURL, token, deviceID)
	}
	// 本地状态:保留 key/device_id,清除凭据,标记 unbound(重启后仍保持未激活)
	mgr := NewLicenseStateManager(st)
	now := time.Now().Unix()
	if err := mgr.Update(func(m map[string]any) {
		delete(m, "activation_token")
		delete(m, "activation_id")
		m["verify_state"] = "unbound"
		m["sync_state"] = string(SyncUnbound)
		m["unbind_source"] = "user"
		m["last_unbind_reason"] = "user_unbound"
		m["last_unbind_at"] = now
		delete(m, "revoked_at")
		delete(m, "grace_deadline")
		m["last_successful_verify"] = now
	}); err != nil {
		return err
	}
	// 唤醒 SSE 循环(无凭据 → 停止连接)
	if s := LicenseSyncInst(); s != nil {
		s.wakeSSE()
	}
	return nil
}
