package service

// ---------- License 生命周期客户端测试(解绑 ≠ 吊销 / 本地持久化 / 重启保持) ----------
//
// 覆盖(Skill 文档 §52-§54 客户端侧):
//   - 用户解绑:保留 Key,清除凭据,状态 unbound;重启后仍 unbound(不恢复 Active)
//   - 管理员解绑(SSE activation.unbound source=admin + verify unbound):
//     本地状态变为 unbound 且持久化,unbind_source=admin
//   - verify 返回 unbound → 清除 token、保留 key
//   - verify 返回 revoked → 状态 revoked(吊销仍然生效)
//   - 已判定 unbound 不被 Server 不可达覆盖回 grace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DockOrae/DockOrae/internal/config"
	"github.com/DockOrae/DockOrae/internal/settings"
	"github.com/DockOrae/DockOrae/internal/state"
)

// newLifecycleState 构造固定 DataDir 的 AppState(可模拟重启:同一 DataDir 再建一个)。
func newLifecycleState(t *testing.T, serverURL string, dataDir string) *state.AppState {
	t.Helper()
	store, err := settings.Load(t.TempDir(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	t.Setenv("DM_LICENSE_SERVER_URL", serverURL)
	return &state.AppState{Cfg: &config.Config{DataDir: dataDir}, Settings: store}
}

// lifecycleMock 模拟 License Server:verify 返回可配置响应,SSE 推送可配置事件块。
func lifecycleMock(t *testing.T, verifyBody string, eventsBlock string, verifyHits *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/activate"):
			_, _ = w.Write([]byte(okActivate))
			return
		case strings.HasSuffix(r.URL.Path, "/verify"):
			if verifyHits != nil {
				*verifyHits++
			}
			_, _ = w.Write([]byte(verifyBody))
			return
		case strings.HasSuffix(r.URL.Path, "/deactivate"):
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		case strings.HasSuffix(r.URL.Path, "/events"):
			fl, _ := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			if eventsBlock != "" {
				_, _ = w.Write([]byte(eventsBlock))
				fl.Flush()
			}
			<-r.Context().Done()
			return
		}
		w.WriteHeader(404)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// unboundVerifyBody verify 返回 unbound(License 仍 ACTIVE,只是解绑)。
const unboundVerifyBody = `{"status":"unbound","valid":false,"license_id":"DMG-TEST","expires_at":1900000000,
	"features":["compose"],"server_time":1900000000,"state_version":3,"unbind_reason":"unbound"}`

// TestLifecycleUserUnbindKeepsKeyAndPersists 用户解绑:
// 保留 Key,清除凭据,状态 unbound;模拟重启后仍 unbound(绝不恢复 Active)。
func TestLifecycleUserUnbindKeepsKeyAndPersists(t *testing.T) {
	srv := lifecycleMock(t, okVerify, "", nil)
	dataDir := t.TempDir()
	st := newLifecycleState(t, srv.URL, dataDir)

	// 激活成功
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	m, _ := readLicenseStore(st)
	if strOr(m["activation_token"]) == "" {
		t.Fatal("activate must store token")
	}

	// 用户解绑
	if err := LicenseDeactivate(st); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	m, _ = readLicenseStore(st)
	if strOr(m["key"]) == "" {
		t.Fatal("user unbind must KEEP key (can re-activate)")
	}
	if strOr(m["activation_token"]) != "" {
		t.Fatal("user unbind must clear activation_token")
	}
	if strOr(m["sync_state"]) != string(SyncUnbound) {
		t.Fatalf("sync_state = %s, want unbound", strOr(m["sync_state"]))
	}
	if strOr(m["last_unbind_reason"]) != "user_unbound" {
		t.Fatalf("last_unbind_reason = %s, want user_unbound", strOr(m["last_unbind_reason"]))
	}

	// 模拟重启:同一 DataDir 重新加载 → 仍 unbound,不恢复 Active
	st2 := newLifecycleState(t, srv.URL, dataDir)
	info := LicenseInfo(st2)
	if info["active"] != false {
		t.Fatalf("after restart: license must stay inactive, got %v", info)
	}
	if strOr(info["online"].(map[string]any)["state"]) != onlineUnbound {
		t.Fatalf("after restart: online state = %v, want unbound", info["online"])
	}
	if LicenseFeatureActive(st2, "compose") {
		t.Fatal("feature must be disabled after unbind")
	}
}

// TestLifecycleVerifyUnboundClearsCreds verify 返回 unbound:
// 清除 activation_token/activation_id,保留 key,状态 unbound,unbind_reason 记录。
func TestLifecycleVerifyUnboundClearsCreds(t *testing.T) {
	srv := lifecycleMock(t, unboundVerifyBody, "", nil)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	out := VerifyNow(st)
	if strOr(out["state"]) != onlineUnbound {
		t.Fatalf("state = %v, want unbound", out["state"])
	}
	m, _ := readLicenseStore(st)
	if strOr(m["key"]) == "" {
		t.Fatal("verify unbound must KEEP key")
	}
	if strOr(m["activation_token"]) != "" || strOr(m["activation_id"]) != "" {
		t.Fatal("verify unbound must clear activation creds")
	}
	if strOr(m["sync_state"]) != string(SyncUnbound) {
		t.Fatalf("sync_state = %s, want unbound", strOr(m["sync_state"]))
	}
	if LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must be disabled in unbound state")
	}
}

// TestLifecycleAdminUnbindSSEPersistsRestart 管理员强制解绑完整链路:
// SSE 推送 activation.unbound(source=admin)→ verify unbound → 本地 unbound + unbind_source=admin
// → 模拟重启 → 仍 unbound(验收:Admin Unbind → DMG 收到 SSE → 本地未激活 → 重启仍未激活)。
func TestLifecycleAdminUnbindSSEPersistsRestart(t *testing.T) {
	// SSE 事件块:activation.unbound,payload source=admin
	evData, _ := json.Marshal(map[string]any{
		"event_id": "evt_88", "sequence_id": 88, "event_type": "activation.unbound",
		"state_version": 3, "license_id": "DMG-TEST", "activation_id": "ACT-TEST",
		"device_id": "d1", "payload": map[string]any{"source": "admin", "reason": "admin_unbound"},
	})
	block := "event: activation.unbound\nid: evt_88\ndata: " + string(evData) + "\n\n"

	var verifyHits int
	srv := lifecycleMock(t, unboundVerifyBody, block, &verifyHits)
	dataDir := t.TempDir()
	st := newLifecycleState(t, srv.URL, dataDir)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatalf("activate: %v", err)
	}

	// 启动同步引擎:SSE 收到 unbound 事件 → verify → unbound
	s := StartLicenseSync(st)
	defer s.Stop()

	// 等待状态收敛
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		m, _ := readLicenseStore(st)
		if strOr(m["sync_state"]) == string(SyncUnbound) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	m, _ := readLicenseStore(st)
	if strOr(m["sync_state"]) != string(SyncUnbound) {
		t.Fatalf("after admin unbind: sync_state = %s, want unbound (verifyHits=%d)", strOr(m["sync_state"]), verifyHits)
	}
	if strOr(m["unbind_source"]) != "admin" {
		t.Fatalf("unbind_source = %s, want admin (SSE payload must be captured)", strOr(m["unbind_source"]))
	}
	if strOr(m["last_unbind_reason"]) != "admin_unbound" {
		t.Fatalf("last_unbind_reason = %s, want admin_unbound", strOr(m["last_unbind_reason"]))
	}
	if strOr(m["activation_token"]) != "" {
		t.Fatal("admin unbind must clear activation_token")
	}
	if strOr(m["key"]) == "" {
		t.Fatal("admin unbind must KEEP key (can re-activate)")
	}

	// 模拟重启:同一 DataDir → 仍 unbound(绝不恢复 Active)
	st2 := newLifecycleState(t, srv.URL, dataDir)
	info := LicenseInfo(st2)
	if info["active"] != false {
		t.Fatalf("after restart: must stay inactive, got %v", info)
	}
	if strOr(info["online"].(map[string]any)["state"]) != onlineUnbound {
		t.Fatalf("after restart: online state = %v, want unbound", info["online"])
	}
}

// TestLifecycleRevokeStillRevoked verify 返回 revoked → revoked(吊销仍然生效,与 unbound 区分)。
func TestLifecycleRevokeStillRevoked(t *testing.T) {
	srv := lifecycleMock(t, `{"status":"revoked","valid":false,"server_time":1900000000}`, "", nil)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	out := VerifyNow(st)
	if strOr(out["state"]) != onlineRevoked {
		t.Fatalf("state = %v, want revoked", out["state"])
	}
	m, _ := readLicenseStore(st)
	if strOr(m["sync_state"]) != string(SyncRevoked) {
		t.Fatalf("sync_state = %s, want revoked", strOr(m["sync_state"]))
	}
	// 吊销 ≠ 解绑:key 仍然保留(吊销是 License 状态,本地凭据作废由 server 判定)
	if strOr(m["key"]) == "" {
		t.Fatal("revoke must keep key for display")
	}
	if LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must be disabled after revoke")
	}
}

// TestLifecycleUnboundNotOverriddenByOffline 已判定 unbound 不被 Server 不可达覆盖回 grace。
func TestLifecycleUnboundNotOverriddenByOffline(t *testing.T) {
	// 先在线拿到 unbound 状态
	srv := lifecycleMock(t, unboundVerifyBody, "", nil)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	VerifyNow(st)
	m, _ := readLicenseStore(st)
	if strOr(m["sync_state"]) != string(SyncUnbound) {
		t.Fatalf("setup: want unbound, got %s", strOr(m["sync_state"]))
	}

	// Server 不可达 → markOffline 不得把 unbound 覆盖为 grace
	srv.Close()
	s := &LicenseSync{state: NewLicenseStateManager(st)}
	s.markOffline("test")
	m, _ = readLicenseStore(st)
	if strOr(m["sync_state"]) != string(SyncUnbound) {
		t.Fatalf("offline must not override unbound, got %s", strOr(m["sync_state"]))
	}
}
