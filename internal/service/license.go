package service

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// 内置签名密钥(离线校验;仅用于 V1 旧版 Key 兼容校验)
const licenseSecret = "docker-manager-go-license-v1"

// ---------- V2: Ed25519 公钥注册表 ----------
//
// 由 Docker_Manager_License(授权方)签发,V2 Key 用 Ed25519 签名,
// 客户端只持有公钥即可离线验证 —— 源码完全公开也无法伪造合法 Key。
//
// key_id 支持密钥轮换:新增公钥在此追加,旧 key 继续有效;
// 未知 key_id 一律拒绝(UNSUPPORTED_KEY,不静默接受)。
//
// 替换公钥:在 Docker_Manager_License 日志(首次启动 PUBLIC KEY)或
// `license-server pubkey` 命令获取当前公钥,替换下方对应 key_id 的值。
var licensePublicKeys = map[string]string{
	// 固定公钥(与 Docker_Manager_License 仓库 private/license.key 配对,永久不变)。
	// 私钥由授权方保管(部署端 private/license.key),此公钥可安全公开。
	"2026-01": `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAdD7pzvzYQClRQC6AfDBed6vottConCnihO881v1A008=
-----END PUBLIC KEY-----`,
}

func LicenseSign(payload string) string {
	mac := hmac.New(sha256.New, []byte(licenseSecret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// LicenseGenerateKey 生成 V1 License Key(内部工具:user/type/exp 天)—— 仅测试/演示使用
func LicenseGenerateKey(user, licenseType string, expDays int64) string {
	exp := time.Now().Unix() + expDays*86400
	payload, _ := json.Marshal(map[string]any{"user": user, "type": licenseType, "exp": exp})
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + LicenseSign(encoded)
}

// LicenseVerifyKey 校验并解析 Key(V1 HMAC / V2 Ed25519 双轨兼容)
//
// 分派规则(按签名段特征):
//   - V1:签名段为 32 位 hex(HMAC-SHA256 截断)—— 旧版兼容
//   - V2:签名段为 88 位 base64url(Ed25519 64B 签名)—— 新版,由 License Server 签发
func LicenseVerifyKey(key string) (map[string]any, bool) {
	key = strings.TrimSpace(key)
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return nil, false
	}
	payload, sig := parts[0], parts[1]

	var raw []byte
	var ok bool
	if isV2Signature(sig) {
		raw, ok = licenseVerifyV2(payload, sig)
	} else {
		if LicenseSign(payload) != sig {
			return nil, false
		}
		var err error
		raw, err = base64.RawURLEncoding.DecodeString(payload)
		ok = err == nil
	}
	if !ok {
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

// isV2Signature 判断签名段是否为 V2(Ed25519 64B 签名 → base64url 无填充 = 86 字符)
func isV2Signature(sig string) bool {
	if len(sig) != 86 {
		return false
	}
	for i := 0; i < len(sig); i++ {
		c := sig[i]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

// licenseVerifyV2 校验 V2 Key:payload 必须 version=2 且 key_id 在注册表中,
// 用对应公钥验证 Ed25519 签名。未知版本/未知 key_id 一律拒绝。
func licenseVerifyV2(payload, sig string) ([]byte, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, false
	}
	var head struct {
		Version int    `json:"version"`
		KeyID   string `json:"key_id"`
	}
	if json.Unmarshal(raw, &head) != nil || head.Version != 2 {
		return nil, false // 未知版本:拒绝而非静默接受
	}
	pubPEM, ok := licensePublicKeys[head.KeyID]
	if !ok {
		return nil, false // 未知 key_id:拒绝
	}
	pub, err := parseEd25519PublicKey([]byte(pubPEM))
	if err != nil {
		return nil, false
	}
	sigRaw, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || len(sigRaw) != ed25519.SignatureSize {
		return nil, false
	}
	if !ed25519.Verify(pub, raw, sigRaw) {
		return nil, false
	}
	return raw, true
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

func numOr(v any) float64 {
	if n, ok := v.(float64); ok {
		return n
	}
	return 0
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
//   - 离线模式:V1 旧 Key(无 features 字段)= 全部功能开启(兼容旧授权);V2 Key 按 payload.features 精确控制
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
	feats, _ := info["features"].([]any)
	if len(feats) == 0 {
		return true // V1 无 features:全功能兼容
	}
	for _, f := range feats {
		if strOr(f) == feature {
			return true
		}
	}
	return false
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
	bound := boundID != "" && boundID == deviceID
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
	if lv := int64(float64(numOr(m["last_successful_verify"]))); lv > 0 {
		out["last_verify"] = lv
		out["grace_deadline"] = lv + int64(licenseGracePeriod.Seconds())
	}
	if vs := strOr(m["verify_state"]); vs != "" {
		out["verify_state"] = vs
	}
	if ra := int64(float64(numOr(m["revoked_at"]))); ra > 0 {
		out["revoked_at"] = ra
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
		res, code, err := licenseActivateRemote(serverURL, store["key"].(string), deviceID, DisplayVersion())
		if err != nil {
			switch code {
			case "DEVICE_LIMIT_REACHED":
				return nil, BadRequest("license.deviceLimit")
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
		store["last_successful_verify"] = time.Now().Unix()
		store["server_url"] = serverURL
	}
	if err := writeLicenseStore(st, store); err != nil {
		return nil, BadRequest("write failed: " + err.Error())
	}
	info["device_id"] = deviceID
	return info, nil
}

// LicenseDeactivate 解除激活(删除本地授权文件;在线模式先通知服务端解绑)。
func LicenseDeactivate(st *state.AppState) error {
	m, ok := readLicenseStore(st)
	if !ok {
		return BadRequest("license.notActivated")
	}
	// 在线模式:先尝试服务端解绑(尽力而为,失败不阻断本地解绑)
	serverURL := serverURLOf(st)
	key := strOr(m["key"])
	activationID := strOr(m["activation_id"])
	deviceID := strOr(m["device_id"])
	if serverURL != "" && key != "" && deviceID != "" && activationID != "" {
		licenseDeactivateRemote(serverURL, key, activationID, deviceID)
	}
	return os.Remove(licensePath(st))
}

// licenseDemoKey 演示/开发用 key(2100 年过期,永久授权)
func licenseDemoKey() string {
	return LicenseGenerateKey("demo", "pro", 2100*365)
}

// DemoKey 供 api 层使用(演示/开发用 key)
func DemoKey() string { return licenseDemoKey() }
