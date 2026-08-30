package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DockerManger/Docker_Manager_Go/internal/config"
	"github.com/DockerManger/Docker_Manager_Go/internal/settings"
	"github.com/DockerManger/Docker_Manager_Go/internal/state"
)

// ---------- V3 Event-Driven 客户端测试 ----------
//
// 覆盖:
//   - SSE 客户端解析与事件分发(Test 7 的客户端侧)
//   - 事件幂等(Test 8:evt_100 ×3 → 只处理一次)
//   - 事件乱序(Test 9:version 10/12/11 → 最终 12)
//   - 事件 Gap(Test 10:local=100 server=105 → Verify)
//   - 宽限过期(Test 11)→ Restricted
//   - Last-Event-ID 重连头(Test 7 服务端侧)

// mockSSEServer 模拟 License Server SSE 端点:先回放给定事件块,然后挂起直到被取消。
func mockSSEServer(t *testing.T, eventsBlock string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/activate"):
			_, _ = w.Write([]byte(okActivate))
			return
		case strings.HasSuffix(r.URL.Path, "/verify"):
			_, _ = w.Write([]byte(okVerify))
			return
		case strings.HasSuffix(r.URL.Path, "/events"):
			fl, _ := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(200)
			if eventsBlock != "" {
				_, _ = w.Write([]byte(eventsBlock))
				fl.Flush()
			}
			// 挂起连接,直到客户端断开
			<-r.Context().Done()
			return
		}
		w.WriteHeader(404)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// sseEventBlock 构造 SSE 事件块。
func sseEventBlock(evType, id string, seq, version int64) string {
	data, _ := json.Marshal(map[string]any{
		"event_id": id, "sequence_id": seq, "event_type": evType,
		"state_version": version, "license_id": "DMG-TEST", "activation_id": "ACT-TEST",
	})
	return fmt.Sprintf("event: %s\nid: %s\ndata: %s\n\n", evType, id, string(data))
}

// newTestSync 构造带同步引擎的测试环境(真实 StartLicenseSync,便于测 SSE 闭环)。
func newTestSync(t *testing.T, serverURL string) (*state.AppState, *LicenseSync) {
	t.Helper()
	store, err := settings.Load(t.TempDir(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	t.Setenv("DM_LICENSE_SERVER_URL", serverURL)
	st := &state.AppState{
		Cfg:      &config.Config{DataDir: t.TempDir()},
		Settings: store,
	}
	// 先激活(写入凭据),再启动引擎
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	s := StartLicenseSync(st)
	t.Cleanup(s.Stop)
	return st, s
}

// TestSSEConnectSendsLastEventID 重连时携带 Last-Event-ID(Test 7 客户端侧)。
func TestSSEConnectSendsLastEventID(t *testing.T) {
	var gotLastID string
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/activate"):
			_, _ = w.Write([]byte(okActivate))
			return
		case strings.HasSuffix(r.URL.Path, "/verify"):
			_, _ = w.Write([]byte(okVerify))
			return
		case strings.HasSuffix(r.URL.Path, "/events"):
			gotLastID = r.Header.Get("Last-Event-ID")
			close(done)
			fl, _ := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(": keep-alive\n\n"))
			fl.Flush()
			<-r.Context().Done()
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	// 预置 last_event_id(模拟此前处理过 evt_100)
	m, _ := readLicenseStore(st)
	m["last_event_id"] = "evt_100"
	m["last_event_seq"] = int64(100)
	if err := writeLicenseStore(st, m); err != nil {
		t.Fatal(err)
	}

	s := StartLicenseSync(st)
	defer s.Stop()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("SSE connection not established")
	}
	if gotLastID != "evt_100" {
		t.Fatalf("Last-Event-ID = %q, want evt_100", gotLastID)
	}
}

// TestEventIdempotency 同一事件重复送达只处理一次(Test 8)。
func TestEventIdempotency(t *testing.T) {
	st := testLicState(t, "")
	// 不启动引擎,直接测 onLicenseEvent 的幂等逻辑
	s := &LicenseSync{st: st, state: NewLicenseStateManager(st)}

	ev := &sseEvent{EventID: "evt_100", SequenceID: 100, EventType: "license.changed", StateVersion: 2}
	verifyCount := 0
	// 注入 verify 计数
	orig := licenseVerifyCountHook
	licenseVerifyCountHook = func() { verifyCount++ }
	defer func() { licenseVerifyCountHook = orig }()

	s.onLicenseEvent(ev)
	s.onLicenseEvent(ev) // 重复
	s.onLicenseEvent(ev) // 重复

	if s.lastSeq != 100 {
		t.Fatalf("lastSeq = %d, want 100", s.lastSeq)
	}
	// 只应触发一次 Verify(首个事件触发;重复的被幂等丢弃)
	if verifyCount != 1 {
		t.Fatalf("verify triggered %d times, want 1", verifyCount)
	}
}

// TestEventOrdering 乱序事件(version 10/12/11)→ 最终 state_version=12(Test 9)。
func TestEventOrdering(t *testing.T) {
	st := testLicState(t, "")
	s := &LicenseSync{st: st, state: NewLicenseStateManager(st)}

	// 本地权威版本 10
	s.stateVersion = 10
	ev := func(seq, ver int64) *sseEvent {
		return &sseEvent{EventID: fmt.Sprintf("evt_%d", seq), SequenceID: seq, EventType: "license.changed", StateVersion: ver}
	}

	s.onLicenseEvent(ev(100, 10)) // version 10 <= 本地 10 → 旧事件,忽略
	if s.stateVersion != 10 {
		t.Fatalf("stateVersion = %d, want 10 (stale ignored)", s.stateVersion)
	}
	s.onLicenseEvent(ev(101, 12)) // version 12 > 10 → 接受
	if s.stateVersion != 12 {
		t.Fatalf("stateVersion = %d, want 12", s.stateVersion)
	}
	s.onLicenseEvent(ev(102, 11)) // version 11 <= 12 → 乱序旧事件,忽略
	if s.stateVersion != 12 {
		t.Fatalf("stateVersion = %d, want 12 (out-of-order ignored)", s.stateVersion)
	}
}

// TestEventGapTriggersVerify 事件序号跳变(Gap)→ Verify(Test 10)。
func TestEventGapTriggersVerify(t *testing.T) {
	st := testLicState(t, "")
	s := &LicenseSync{st: st, state: NewLicenseStateManager(st)}
	s.lastSeq = 100 // local = 100

	verifyCount := 0
	orig := licenseVerifyCountHook
	licenseVerifyCountHook = func() { verifyCount++ }
	defer func() { licenseVerifyCountHook = orig }()

	// server = 105:序号 101-104 缺失 → Gap → Verify
	s.onLicenseEvent(&sseEvent{EventID: "evt_105", SequenceID: 105, EventType: "license.changed", StateVersion: 20})
	if verifyCount != 1 {
		t.Fatalf("gap must trigger verify, got %d", verifyCount)
	}
}

// TestGraceExpiredOnStartup 启动时 Server 不可达 + 宽限已过 → grace_expired(Test 11)。
func TestGraceExpiredOnStartup(t *testing.T) {
	st := testLicState(t, "http://127.0.0.1:1") // 不可达
	key := v2TestKey()
	deviceID := LicenseDeviceID(st.Cfg.DataDir)
	// 激活成功过,但 8 天前(宽限 72h 已过);sync_state 未标记(重启场景)
	if err := writeLicenseStore(st, map[string]any{
		"key": key, "device_id": deviceID,
		"activation_id":          "ACT-OLD",
		"activation_token":       "old-token",
		"last_successful_verify": time.Now().Unix() - 8*86400,
		"server_url":             "http://127.0.0.1:1",
	}); err != nil {
		t.Fatal(err)
	}
	// 手动验证(启动 Verify 的等价路径)→ 网络错误 → markOffline → 宽限评估 → 已过期
	out := VerifyNow(st)
	if strOr(out["state"]) != onlineGraceExpired {
		t.Fatalf("state = %v, want grace_expired (error=%v)", out["state"], out["error"])
	}
	if LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must be disabled after grace expired")
	}
}

// TestGraceActiveOnStartup 启动时 Server 不可达但宽限未过 → grace(功能保留)。
func TestGraceActiveOnStartup(t *testing.T) {
	st := testLicState(t, "http://127.0.0.1:1")
	key := v2TestKey()
	deviceID := LicenseDeviceID(st.Cfg.DataDir)
	if err := writeLicenseStore(st, map[string]any{
		"key": key, "device_id": deviceID,
		"activation_id":          "ACT-OLD",
		"activation_token":       "old-token",
		"last_successful_verify": time.Now().Unix() - 3600, // 1 小时前,宽限未过
		"server_url":             "http://127.0.0.1:1",
	}); err != nil {
		t.Fatal(err)
	}
	out := VerifyNow(st)
	if strOr(out["state"]) != onlineGrace {
		t.Fatalf("state = %v, want grace (error=%v)", out["state"], out["error"])
	}
	if !LicenseFeatureActive(st, "compose") {
		t.Fatal("feature must stay active within grace")
	}
}

// TestSSEEventTriggersVerify SSE 事件到达 → 触发 Verify(Test 2 客户端侧)。
func TestSSEEventTriggersVerify(t *testing.T) {
	// 事件块:一条 license.changed(模拟管理端解绑/吊销后的推送)
	block := sseEventBlock("activation.unbound", "evt_50", 50, 2)
	srv := mockSSEServer(t, block)
	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	s := StartLicenseSync(st)
	defer s.Stop()

	// 等待事件处理 + Verify 完成
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		m, _ := readLicenseStore(st)
		if strOr(m["last_event_id"]) == "evt_50" {
			return // 事件已处理
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("SSE event not processed within timeout")
}

// TestSSE401TriggersVerify SSE 401(凭据被服务端拒绝)→ 触发 Verify(吊销后客户端不卡 Grace)。
func TestSSE401TriggersVerify(t *testing.T) {
	verifyCount := 0
	orig := licenseVerifyCountHook
	licenseVerifyCountHook = func() { verifyCount++ }
	defer func() { licenseVerifyCountHook = orig }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/activate"):
			_, _ = w.Write([]byte(okActivate))
			return
		case strings.HasSuffix(r.URL.Path, "/verify"):
			// token 已吊销 → 200 invalid(服务端权威结论)
			_, _ = w.Write([]byte(`{"status":"invalid","valid":false,"server_time":1780000000}`))
			return
		case strings.HasSuffix(r.URL.Path, "/events"):
			w.WriteHeader(401) // token 作废后 SSE 认证失败
			return
		}
		w.WriteHeader(404)
	}))
	t.Cleanup(srv.Close)

	st := testLicState(t, srv.URL)
	if _, err := LicenseDoActivate(st, v2TestKey()); err != nil {
		t.Fatal(err)
	}
	s := StartLicenseSync(st)
	defer s.Stop()

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if verifyCount >= 1 {
			m, _ := readLicenseStore(st)
			if strOr(m["sync_state"]) == string(SyncRevoked) {
				return // 401 → Verify → revoked ✓
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("401 must trigger verify (count=%d)", verifyCount)
}

// TestNoPeriodicVerify 同步引擎不包含任何周期验证(Ticker/Timer/Cron)。
func TestNoPeriodicVerify(t *testing.T) {
	src, err := os.ReadFile("license_sync.go")
	if err != nil {
		t.Fatal(err)
	}
	content := string(src)
	for _, banned := range []string{"time.NewTicker", "time.NewTimer", "time.AfterFunc", "Cron", "tick.C", "timer.C"} {
		if strings.Contains(content, banned) {
			t.Fatalf("license_sync.go must not contain periodic mechanism: %s", banned)
		}
	}
}
