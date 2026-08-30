package service

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DockerManger/Docker_Manager_Go/internal/config"
	"github.com/DockerManger/Docker_Manager_Go/internal/settings"
	"github.com/DockerManger/Docker_Manager_Go/internal/state"
)

// ---------- 在线授权闭环测试(客户端侧) ----------
//
// 用 httptest 模拟 Docker_Manager_License 服务端公开 API,验证:
// 在线激活成功/设备上限/服务器不可达、周期验证(valid/revoked)、Grace Period 计算、离线模式兼容。

// mockLicenseServer 模拟授权服务器公开 API(activate/verify/deactivate)。
func mockLicenseServer(t *testing.T, activateStatus int, activateBody string, verifyStatus int, verifyBody string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/activate"):
			w.WriteHeader(activateStatus)
			_, _ = w.Write([]byte(activateBody))
		case strings.HasSuffix(r.URL.Path, "/verify"):
			w.WriteHeader(verifyStatus)
			_, _ = w.Write([]byte(verifyBody))
		case strings.HasSuffix(r.URL.Path, "/deactivate"):
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"?"}}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// testLicState 构造测试 AppState(可指定授权服务器 URL;空 = 离线模式)。
func testLicState(t *testing.T, serverURL string) *state.AppState {
	t.Helper()
	store, err := settings.Load(t.TempDir(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	// 授权服务器地址走环境变量(生产为内置固定域名;测试注入 mock 地址)
	t.Setenv("DM_LICENSE_SERVER_URL", serverURL)
	return &state.AppState{
		Cfg:      &config.Config{DataDir: t.TempDir()},
		Settings: store,
	}
}

// ---------- V2 测试密钥(固定注册,供在线闭环测试) ----------

var (
	testKeyOnce sync.Once
	testPriv    ed25519.PrivateKey
)

// ensureTestKey 注册一个固定测试公钥到注册表(全测试共享,不清理)。
func ensureTestKey() {
	testKeyOnce.Do(func() {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			panic(err)
		}
		der, err := x509.MarshalPKIXPublicKey(pub)
		if err != nil {
			panic(err)
		}
		licensePublicKeys["test-local"] = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
		testPriv = priv
	})
}

// v2TestKey 生成测试用 V2 Key(本地验签通过,status=active,features 全开)。
func v2TestKey() string {
	ensureTestKey()
	p := v2Payload("test-local")
	raw, _ := json.Marshal(p)
	sig := ed25519.Sign(testPriv, raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." +
		base64.RawURLEncoding.EncodeToString(sig)
}

const okActivate = `{"status":"active","activation_id":"ACT-TESTACTIVATION0001","activation_token":"tok-0123456789abcdef0123456789abcdef","license_id":"DMG-TEST","expires_at":1900000000,"max_devices":3,"server_time":1900000000,"next_verify_after":86400}`
const okVerify = `{"status":"valid","valid":true,"license_id":"DMG-TEST","expires_at":1900000000,"features":["compose"],"server_time":1900000000,"next_verify_after":86400}`
const errBody = `{"error":{"code":"%s","message":"%s"}}`

// TestOnlineActivateSuccess 在线激活成功:license.json 写入 activation_id + activation_token,状态 verified,功能可用。
func TestOnlineActivateSuccess(t *testing.T) {
	srv := mockLicenseServer(t, 200, okActivate, 200, okVerify)
	st := testLicState(t, srv.URL)

	info, err := LicenseDoActivate(st, v2TestKey())
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if strOr(info["device_id"]) == "" {
		t.Fatal("device_id missing")
	}
	m, ok := readLicenseStore(st)
	if !ok {
		t.Fatal("license.json not written")
	}
	if strOr(m["activation_id"]) != "ACT-TESTACTIVATION0001" {
		t.Fatalf("activation_id not stored: %v", m["activation_id"])
	}
	// V3:activation_token 必须保存
	if strOr(m["activation_token"]) != "tok-0123456789abcdef0123456789abcdef" {
		t.Fatalf("activation_token not stored: %v", m["activation_token"])
	}
	if int64(float64(numOr(m["last_successful_verify"]))) == 0 {
		t.Fatal("last_successful_verify not stored")
	}
	// 在线状态:verified + 功能可用
	if s := onlineStateOf(st); s != onlineVerified {
		t.Fatalf("state = %s, want verified", s)
	}
	if !LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must be active after online activation")
	}
	if !LicenseActive(st) {
		t.Fatal("license must be active")
	}
	// LicenseInfo.online 详情
	li := LicenseInfo(st)
	on, _ := li["online"].(map[string]any)
	if strOr(on["mode"]) != "online" || strOr(on["state"]) != onlineVerified {
		t.Fatalf("online info wrong: %+v", on)
	}
}

// TestOnlineActivateDeviceLimit 设备上限:服务端 409 → 激活拒绝。
func TestOnlineActivateDeviceLimit(t *testing.T) {
	srv := mockLicenseServer(t, 409, `{"error":{"code":"DEVICE_LIMIT_REACHED","message":"limit"}}`, 200, okVerify)
	st := testLicState(t, srv.URL)

	_, err := LicenseDoActivate(st, v2TestKey())
	if err == nil {
		t.Fatal("activate must fail on device limit")
	}
	ae, ok := err.(*ApiError)
	if !ok || ae.Message != "license.deviceLimit" {
		t.Fatalf("want license.deviceLimit error, got %v", err)
	}
	if LicenseActive(st) {
		t.Fatal("license must not be active after failed activation")
	}
}

// TestOnlineActivateRevoked 服务端吊销 → 激活拒绝。
func TestOnlineActivateRevoked(t *testing.T) {
	srv := mockLicenseServer(t, 403, `{"error":{"code":"LICENSE_REVOKED","message":"revoked"}}`, 200, okVerify)
	st := testLicState(t, srv.URL)

	_, err := LicenseDoActivate(st, v2TestKey())
	if err == nil {
		t.Fatal("activate must fail on revoked")
	}
	ae, ok := err.(*ApiError)
	if !ok || ae.Message != "license.revokedKey" {
		t.Fatalf("want license.revokedKey error, got %v", err)
	}
}

// TestOnlineActivateServerDown 服务器不可达 → 激活拒绝(严格在线)。
func TestOnlineActivateServerDown(t *testing.T) {
	st := testLicState(t, "http://127.0.0.1:1") // 必然连接失败

	_, err := LicenseDoActivate(st, v2TestKey())
	if err == nil {
		t.Fatal("activate must fail when server unreachable")
	}
	ae, ok := err.(*ApiError)
	if !ok || ae.Message != "license.serverUnreachable" {
		t.Fatalf("want license.serverUnreachable error, got %v", err)
	}
}

// TestOfflineModeUnchanged 未配置授权服务器:维持纯离线激活,不受在线逻辑影响。
func TestOfflineModeUnchanged(t *testing.T) {
	st := testLicState(t, "")

	info, err := LicenseDoActivate(st, v2TestKey())
	if err != nil {
		t.Fatalf("offline activate: %v", err)
	}
	if !LicenseActive(st) {
		t.Fatal("offline license must be active")
	}
	if strOr(info["device_id"]) == "" {
		t.Fatal("device_id missing")
	}
	// 离线模式无 activation_id
	m, _ := readLicenseStore(st)
	if strOr(m["activation_id"]) != "" {
		t.Fatal("offline activation must not have activation_id")
	}
	if s := onlineStateOf(st); s != onlineOffline {
		t.Fatalf("state = %s, want offline", s)
	}
}

// TestVerifyNowValid 手动验证 valid → 恢复 online,功能保持。
func TestVerifyNowValid(t *testing.T) {
	srv := mockLicenseServer(t, 200, okActivate, 200, okVerify)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	// 模拟 Server 故障离线:sync_state=grace(引擎 markOffline 写入)
	m, _ := readLicenseStore(st)
	m["sync_state"] = "grace"
	m["last_successful_verify"] = time.Now().Unix() - 2*86400
	if err := writeLicenseStore(st, m); err != nil {
		t.Fatal(err)
	}
	if s := onlineStateOf(st); s != onlineGrace {
		t.Fatalf("state = %s, want grace", s)
	}
	// 手动验证成功 → verified
	out := VerifyNow(st)
	if strOr(out["state"]) != onlineVerified {
		t.Fatalf("verify result: %+v", out)
	}
	if s := onlineStateOf(st); s != onlineVerified {
		t.Fatalf("state after verify = %s, want verified", s)
	}
	if !LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must stay active after valid verify")
	}
}

// TestVerifyNowRevoked 服务端吊销 → 本地标记,功能立即禁用(下一次门控判断生效)。
func TestVerifyNowRevoked(t *testing.T) {
	srv := mockLicenseServer(t, 200, okActivate, 200, `{"status":"revoked","valid":false}`)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	if !LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must be active before revocation")
	}

	out := VerifyNow(st)
	if strOr(out["state"]) != onlineRevoked {
		t.Fatalf("verify result: %+v", out)
	}
	if LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must be disabled after revocation")
	}
	if LicenseActive(st) {
		t.Fatal("license must be inactive after revocation")
	}
}

// TestGraceExpiredDisablesFeature 宽限过期(sync_state=grace_expired)→ 功能禁用(即使本地签名有效)。
func TestGraceExpiredDisablesFeature(t *testing.T) {
	st := testLicState(t, "http://127.0.0.1:1") // 服务器不可达,验证必然失败
	if _, err := LicenseDoActivate(st, v2TestKey()); err == nil {
		t.Fatal("activate must fail with unreachable server")
	}
	// 直接写入在线激活结果(模拟此前激活成功过),再置为宽限过期状态
	key := v2TestKey()
	deviceID := LicenseDeviceID(st.Cfg.DataDir)
	if err := writeLicenseStore(st, map[string]any{
		"key": key, "device_id": deviceID,
		"activation_id":          "old-code",
		"last_successful_verify": time.Now().Unix() - 8*86400,
		"sync_state":             "grace_expired", // 引擎在宽限到期后写入
		"server_url":             "http://127.0.0.1:1",
	}); err != nil {
		t.Fatal(err)
	}
	if s := onlineStateOf(st); s != onlineGraceExpired {
		t.Fatalf("state = %s, want grace_expired", s)
	}
	if LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must be disabled after grace period expires")
	}
	if LicenseActive(st) {
		t.Fatal("license must be inactive after grace period expires")
	}
}

// TestOnlineDeactivateCallsServer 在线解绑:通知服务端后删除本地文件。
func TestOnlineDeactivateCallsServer(t *testing.T) {
	srv := mockLicenseServer(t, 200, okActivate, 200, okVerify)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	if err := LicenseDeactivate(st); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if _, ok := readLicenseStore(st); ok {
		t.Fatal("license.json must be removed")
	}
	if LicenseActive(st) {
		t.Fatal("license must be inactive after deactivate")
	}
}

// TestVerifyOfflineNoop 未配置服务器时 VerifyNow 不发起请求,返回 offline。
func TestVerifyOfflineNoop(t *testing.T) {
	st := testLicState(t, "")
	out := VerifyNow(st)
	if strOr(out["mode"]) != "offline" {
		t.Fatalf("mode = %v, want offline", out["mode"])
	}
}

// TestVerifyServerErrorKeepsGrace 验证网络失败:不更新 last_successful_verify(宽限自然推进)。
func TestVerifyServerErrorKeepsGrace(t *testing.T) {
	srv := mockLicenseServer(t, 200, okActivate, 500, `{"error":{"code":"INTERNAL","message":"boom"}}`)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	m, _ := readLicenseStore(st)
	m["last_successful_verify"] = time.Now().Unix() - 2*86400 // 推到宽限期
	if err := writeLicenseStore(st, m); err != nil {
		t.Fatal(err)
	}
	out := VerifyNow(st)
	if strOr(out["state"]) != onlineGrace {
		t.Fatalf("state after failed verify = %v, want grace (error=%v)", out["state"], out["error"])
	}
	// 功能在宽限期内仍可用
	if !LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must stay active within grace period")
	}
}

// TestLicenseInfoJSON 序列化完整性(前端消费 online 字段)。
func TestLicenseInfoJSON(t *testing.T) {
	st := testLicState(t, "")
	_, err := LicenseDoActivate(st, v2TestKey())
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(LicenseInfo(st))
	if !strings.Contains(string(raw), `"online"`) || !strings.Contains(string(raw), `"mode"`) {
		t.Fatalf("LicenseInfo must include online info: %s", raw)
	}
}

// ---------- V3 协议测试(token / server_time / 时钟回退 / 版本控制) ----------

// TestVerifyUsesTokenAndNonce verify 请求必须携带 activation_token + timestamp + nonce(不带 key)。
func TestVerifyUsesTokenAndNonce(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/verify") {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = w.Write([]byte(okVerify))
			return
		}
		_, _ = w.Write([]byte(okActivate))
	}))
	t.Cleanup(srv.Close)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	VerifyNow(st)
	if gotBody == nil {
		t.Fatal("no verify request captured")
	}
	if _, hasKey := gotBody["key"]; hasKey {
		t.Fatal("V3 verify must NOT send license key")
	}
	if strOr(gotBody["activation_token"]) == "" {
		t.Fatal("verify must send activation_token")
	}
	if int64(float64(numOr(gotBody["timestamp"]))) == 0 {
		t.Fatal("verify must send timestamp")
	}
	if strOr(gotBody["nonce"]) == "" {
		t.Fatal("verify must send nonce")
	}
}

// TestClockRollbackDetected 本地时钟回退超过 5 分钟 → 禁用 Pro。
func TestClockRollbackDetected(t *testing.T) {
	st := testLicState(t, "http://127.0.0.1:1")
	key := v2TestKey()
	deviceID := LicenseDeviceID(st.Cfg.DataDir)
	// 模拟正常激活(写入在线凭据),last_local_time 在 10 分钟前(当前时间已回退)
	if err := writeLicenseStore(st, map[string]any{
		"key": key, "device_id": deviceID,
		"activation_id":          "ACT-OLD",
		"activation_token":       "old-token",
		"last_successful_verify": time.Now().Unix(),
		"last_local_time":        time.Now().Unix() + 600, // 模拟回退 10 分钟
		"server_url":             "http://127.0.0.1:1",
	}); err != nil {
		t.Fatal(err)
	}
	if s := onlineStateOf(st); s != onlineClockRollback {
		t.Fatalf("state = %s, want clock_rollback", s)
	}
	if LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must be disabled on clock rollback")
	}
	// VerifyNow 也应标记并返回 clock_rollback
	out := VerifyNow(st)
	if strOr(out["state"]) != onlineClockRollback {
		t.Fatalf("VerifyNow state = %v, want clock_rollback", out["state"])
	}
}

// TestNormalClockNoFalsePositive 正常时钟(无回退)不误判。
func TestNormalClockNoFalsePositive(t *testing.T) {
	st := testLicState(t, "http://127.0.0.1:1")
	key := v2TestKey()
	deviceID := LicenseDeviceID(st.Cfg.DataDir)
	if err := writeLicenseStore(st, map[string]any{
		"key": key, "device_id": deviceID,
		"activation_id":          "ACT-OLD",
		"activation_token":       "old-token",
		"last_successful_verify": time.Now().Unix(),
		"last_local_time":        time.Now().Unix() - 60, // 正常流逝 1 分钟
		"server_url":             "http://127.0.0.1:1",
	}); err != nil {
		t.Fatal(err)
	}
	if s := onlineStateOf(st); s != onlineVerified {
		t.Fatalf("state = %s, want verified (no false positive)", s)
	}
}

// TestServerTimeClockOffset 服务端 server_time → clock_offset 维护(防时间作弊)。
func TestServerTimeClockOffset(t *testing.T) {
	serverTime := time.Now().Unix() + 3600 // 服务端快 1 小时
	activateBody := fmt.Sprintf(`{"status":"active","activation_id":"ACT-X","activation_token":"tok-x","license_id":"DMG-TEST","expires_at":1900000000,"server_time":%d,"next_verify_after":86400}`, serverTime)
	srv := mockLicenseServer(t, 200, activateBody, 200, okVerify)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	m, _ := readLicenseStore(st)
	offset := int64(float64(numOr(m["clock_offset"])))
	if offset < 3590 || offset > 3610 {
		t.Fatalf("clock_offset = %d, want ~3600", offset)
	}
	// trusted_now 应 ≈ 本地 + 3600
	tn := trustedNow(st)
	if tn < time.Now().Unix()+3590 || tn > time.Now().Unix()+3610 {
		t.Fatalf("trusted_now = %d, want ~now+3600", tn)
	}
}

// TestVerifyNowBlocked 服务端 blocked(版本封禁)→ 禁用 Pro。
func TestVerifyNowBlocked(t *testing.T) {
	srv := mockLicenseServer(t, 200, okActivate, 200, `{"status":"blocked","valid":false,"server_time":1900000000}`)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	if !LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must be active before block")
	}
	out := VerifyNow(st)
	if strOr(out["state"]) != onlineVersionBlocked {
		t.Fatalf("verify result: %+v", out)
	}
	if LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must be disabled after version block")
	}
}

// TestVerifyNowUpdateRequired minimum_client_version 高于当前版本 → 提示升级(不封禁)。
func TestVerifyNowUpdateRequired(t *testing.T) {
	oldVer := AppVersion
	AppVersion = "1.0.0"
	t.Cleanup(func() { AppVersion = oldVer })
	srv := mockLicenseServer(t, 200, okActivate, 200,
		`{"status":"valid","valid":true,"server_time":1900000000,"minimum_client_version":"999.0.0"}`)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	out := VerifyNow(st)
	if strOr(out["state"]) != onlineUpdateRequired {
		t.Fatalf("verify result: %+v", out)
	}
	// update_required 只提示,不禁用功能
	if !LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must stay active on update_required")
	}
}

// TestVerifyNoVersionIssue 当前版本满足 minimum_client_version → 正常 verified。
func TestVerifyNoVersionIssue(t *testing.T) {
	oldVer := AppVersion
	AppVersion = "2.0.0"
	t.Cleanup(func() { AppVersion = oldVer })
	srv := mockLicenseServer(t, 200, okActivate, 200,
		`{"status":"valid","valid":true,"server_time":1900000000,"minimum_client_version":"0.1.0"}`)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	out := VerifyNow(st)
	if strOr(out["state"]) != onlineVerified {
		t.Fatalf("verify result: %+v", out)
	}
}

// TestLicenseFileMode0600 license.json 权限 0600(敏感 token 保护)。
// Windows 无 POSIX 权限位,该测试仅限 Linux/Unix。
func TestLicenseFileMode0600(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 无 POSIX 文件权限位")
	}
	srv := mockLicenseServer(t, 200, okActivate, 200, okVerify)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(licensePath(st))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("license.json mode = %o, want 600", perm)
	}
}
