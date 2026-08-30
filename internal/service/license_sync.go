package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DockerManger/Docker_Manager_Go/internal/state"
)

// ---------- V3 License 同步引擎(Event-Driven 主动同步) ----------
//
// 架构(与 Docker_Manager_License 契约一致):
//
//	License Server --(Event)--> SSE --> Docker Manager --(V3 Verify)--> LicenseStateManager --> Vue
//
// 核心原则:
//   - Event = 通知客户端"状态可能变化"(只是 Trigger)
//   - Verify = 从 Server 获取权威状态(唯一 Authority)
//   - SSE Reconnect Success = Server Recovery Signal → 立即 Verify
//   - Grace = Server 故障期间的有限保护(默认 72h,DM_LICENSE_GRACE_PERIOD 可覆盖)
//   - 禁止任何周期性 Verify / Heartbeat / Check-in / Lease
//
// Verify 触发条件(仅此五类):
//   1. Docker Manager 启动
//   2. 收到 License Event(SSE)
//   3. SSE 重连成功(Server Recovery)
//   4. resync_required(Event Replay 无法恢复)
//   5. 用户手动点击"立即检查"

// SyncState 在线同步状态(license.json sync_state 字段)。
type SyncState string

const (
	SyncNone         SyncState = ""                 // 未激活/离线模式
	SyncOnline       SyncState = "online"           // 已连接 + 最近验证有效
	SyncOffline      SyncState = "offline"          // SSE 断开(Server 不可达)
	SyncGrace        SyncState = "grace"            // 宽限期(Server 故障,授权暂时保留)
	SyncGraceExpired SyncState = "grace_expired"    // 宽限过期 → 限制
	SyncRecovered    SyncState = "server_recovered" // SSE 重连成功,正在 Verify
	SyncRevoked      SyncState = "revoked"          // Server 判定吊销/无效/过期
	SyncBlocked      SyncState = "blocked"          // 版本被封禁
)

// sseBackoff SSE 重连指数退避序列(秒;最大间隔封顶,防 Server 故障时疯狂重连)。
var sseBackoff = []time.Duration{
	1 * time.Second, 2 * time.Second, 5 * time.Second,
	10 * time.Second, 30 * time.Second, 60 * time.Second,
}

// licenseGracePeriod 宽限期(Server 不可达时的有限保护;禁止无限期 Pro)。
// 默认 72h;环境变量 DM_LICENSE_GRACE_PERIOD 可覆盖(测试用,如 5s)。
func licenseGracePeriod() time.Duration {
	if v := os.Getenv("DM_LICENSE_GRACE_PERIOD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 72 * time.Hour
}

// LicenseSync V3 同步引擎(启动时创建,常驻)。
type LicenseSync struct {
	st     *state.AppState
	state  *LicenseStateManager
	ctx    context.Context
	cancel context.CancelFunc

	// verify 并发控制(singleflight:同一激活最多一个实际 Verify)
	verify *verifyCoordinator

	// SSE 状态
	sseMu          sync.Mutex
	lastSeq        int64         // 最近处理的事件 sequence(幂等)
	lastEventID    string        // 最近处理的事件 ID(evt_N,Last-Event-ID 用)
	stateVersion   int64         // Server 权威状态版本(乱序保护)
	everConnected  bool          // 是否建立过 SSE(区分首次连接与重连)
	disconnected   bool          // 当前是否处于断开状态
	credsValid     bool          // 是否有可用的激活凭据(有凭据才连 SSE)
	wake           chan struct{} // 凭据变化时唤醒 SSE 循环
	manualVerifyMu sync.Mutex    // 手动验证与事件触发的互斥(单飞覆盖)
}

// verifyCoordinator 单飞:同一时刻最多一个实际 Verify,并发请求共享结果。
type verifyCoordinator struct {
	mu       sync.Mutex
	inflight bool
	done     chan struct{}
	result   map[string]any
}

func (c *verifyCoordinator) run(fn func() map[string]any) map[string]any {
	c.mu.Lock()
	if c.inflight {
		done := c.done
		c.mu.Unlock()
		<-done
		return c.result
	}
	c.inflight = true
	c.done = make(chan struct{})
	c.mu.Unlock()

	c.result = fn()

	c.mu.Lock()
	c.inflight = false
	close(c.done)
	c.mu.Unlock()
	return c.result
}

var (
	syncMu   sync.Mutex
	syncInst *LicenseSync // 当前引擎实例(API 层访问入口)
)

// StartLicenseSync 启动 V3 同步引擎(main.go 在 state 初始化后调用):
// 读取本地 License State → 启动时 V3 Verify → 建立 SSE。
// 返回实例,进程退出时 defer Stop()。
func StartLicenseSync(st *state.AppState) *LicenseSync {
	ctx, cancel := context.WithCancel(context.Background())
	s := &LicenseSync{
		st:     st,
		state:  NewLicenseStateManager(st),
		ctx:    ctx,
		cancel: cancel,
		verify: &verifyCoordinator{},
		wake:   make(chan struct{}, 1),
	}
	syncMu.Lock()
	syncInst = s
	syncMu.Unlock()

	// 1) 启动 Verify(独立 goroutine,不阻塞面板启动;Server 不可达 → Grace)
	go s.triggerVerify("startup")

	// 2) SSE 常驻连接 + 自动重连
	go s.sseLoop()

	logLicenseSync("V3 license sync started (event-driven, no periodic verify)")
	return s
}

// Stop 停止引擎(SSE 断开,goroutine 退出,无泄漏)。
func (s *LicenseSync) Stop() {
	s.cancel()
	syncMu.Lock()
	if syncInst == s {
		syncInst = nil
	}
	syncMu.Unlock()
}

// Inst 当前引擎实例(API 层用;未启动时返回 nil)。
func LicenseSyncInst() *LicenseSync {
	syncMu.Lock()
	defer syncMu.Unlock()
	return syncInst
}

// ---------- Verify 触发与执行 ----------

// licenseVerifyCountHook 测试钩子(统计 Verify 触发次数;生产为空操作)。
var licenseVerifyCountHook = func() {}

// triggerVerify 触发一次 Verify(所有触发源统一进入 singleflight)。
func (s *LicenseSync) triggerVerify(reason string) {
	licenseVerifyCountHook()
	if s == nil || s.state == nil || s.verify == nil {
		return
	}
	go func() {
		res := s.verify.run(func() map[string]any { return s.verifyOnce(reason) })
		_ = res
	}()
}

// VerifyNow 手动触发一次在线验证(API「立即检查」;与事件/重连触发共用单飞)。
// 未启动引擎时直接执行一次(测试/兼容路径)。
func VerifyNow(st *state.AppState) map[string]any {
	if s := LicenseSyncInst(); s != nil {
		return s.verify.run(func() map[string]any { return s.verifyOnce("manual") })
	}
	s := &LicenseSync{st: st, state: NewLicenseStateManager(st), verify: &verifyCoordinator{}}
	return s.verify.run(func() map[string]any { return s.verifyOnce("manual") })
}

// verifyOnce 执行一次 V3 Verify(权威状态获取),并更新本地状态。
// 只在 verifyCoordinator 内调用,保证同一激活最多一个实际 Verify。
func (s *LicenseSync) verifyOnce(reason string) map[string]any {
	serverURL := serverURLOf(s.st)
	if serverURL == "" {
		return map[string]any{"mode": "offline", "state": onlineOffline}
	}
	key, token, activationID, deviceID, ok := loadLicenseCred(s.st)
	if !ok {
		s.setState(SyncNone)
		return map[string]any{"mode": "online", "state": "", "error": "not activated"}
	}
	_ = key
	_ = activationID
	now := time.Now().Unix()

	// 本地时钟回退检测(先于远程请求;回退 = 时间作弊嫌疑 → 禁用 Pro)
	if m, ok := readLicenseStore(s.st); ok && clockRollbackDetected(m) {
		s.setState(SyncNone)
		s.state.Update(func(m map[string]any) {
			m["verify_state"] = "clock_rollback"
		})
		return map[string]any{"mode": "online", "state": onlineClockRollback, "verify_state": "clock_rollback"}
	}

	// 远程 Verify(token 唯一凭据,携带 timestamp+nonce 防重放)
	status, res, err := licenseVerifyRemote(serverURL, token, deviceID, DisplayVersion())
	if err != nil {
		// 网络/服务不可达:不动 last_successful_verify(Grace 自然推进)
		s.markOffline(reason)
		logLicenseSync("verify failed (%s): %v -> offline/grace", reason, err)
		return map[string]any{"mode": "online", "state": onlineStateOf(s.st), "error": err.Error()}
	}
	logLicenseSync("verify (%s) -> status=%s state_version=%v", reason, status, numOr(res["state_version"]))

	switch status {
	case "valid":
		s.state.Update(func(m map[string]any) {
			m["last_successful_verify"] = now
			m["verify_state"] = ""
			m["sync_state"] = string(SyncOnline)
			applyServerTime(m, res)
			if sv := int64(float64(numOr(res["state_version"]))); sv > 0 {
				m["state_version"] = sv
			}
		})
		s.sseMu.Lock()
		s.stateVersion = int64(float64(numOr(res["state_version"])))
		s.sseMu.Unlock()
		// 版本控制:minimum_client_version 高于当前版本 → UPDATE_REQUIRED(提示,不封禁)
		if cv := DisplayVersion(); cv != "unknown" {
			if minVer := strOr(res["minimum_client_version"]); minVer != "" && versionLess(cv, minVer) {
				s.state.Update(func(m map[string]any) { m["verify_state"] = "update_required" })
				return map[string]any{"mode": "online", "state": onlineUpdateRequired, "last_verify": now, "minimum_client_version": minVer}
			}
		}
		return map[string]any{"mode": "online", "state": onlineVerified, "last_verify": now, "sync_state": string(SyncOnline)}
	case "blocked":
		s.state.Update(func(m map[string]any) {
			m["verify_state"] = "blocked"
			m["sync_state"] = string(SyncBlocked)
			m["revoked_at"] = now
			applyServerTime(m, res)
		})
		return map[string]any{"mode": "online", "state": onlineVersionBlocked, "verify_state": "blocked", "revoked_at": now}
	case "revoked", "expired", "invalid":
		s.state.Update(func(m map[string]any) {
			m["verify_state"] = status
			m["sync_state"] = string(SyncRevoked)
			m["revoked_at"] = now
			applyServerTime(m, res)
		})
		return map[string]any{"mode": "online", "state": onlineRevoked, "verify_state": status, "revoked_at": now}
	default:
		return map[string]any{"mode": "online", "state": onlineStateOf(s.st), "error": "unexpected status: " + status}
	}
}

// ---------- LicenseState 订阅(供 /ws/license 实时推送) ----------
//
// 复用引擎的 LicenseStateManager:状态变更 → 通知订阅者 → WS 推给 Vue。
// 引擎未启动(测试)时退化为空订阅。

// SubscribeLicenseState 订阅 License 状态变更。
func SubscribeLicenseState(st *state.AppState) chan struct{} {
	if s := LicenseSyncInst(); s != nil && s.state != nil {
		return s.state.Subscribe()
	}
	ch := make(chan struct{}, 8)
	return ch
}

// UnsubscribeLicenseState 取消订阅。
func UnsubscribeLicenseState(st *state.AppState, ch chan struct{}) {
	if s := LicenseSyncInst(); s != nil && s.state != nil {
		s.state.Unsubscribe(ch)
	}
}

// markOffline Server 不可达:进入 Offline → Grace 评估(有限保护)。
// 已判定的禁用状态(revoked/blocked/grace_expired)优先,不被 Server 不可达覆盖回 grace。
func (s *LicenseSync) markOffline(reason string) {
	s.sseMu.Lock()
	s.disconnected = true
	s.sseMu.Unlock()

	m, _ := readLicenseStore(s.st)
	if m == nil {
		return
	}
	// 禁用状态优先:Server Down ≠ Revoked(§9),但反之已 Revoked 也不回退
	switch SyncState(strOr(m["sync_state"])) {
	case SyncRevoked, SyncBlocked, SyncGraceExpired:
		return
	}
	lastOK := int64(float64(numOr(m["last_successful_verify"])))
	now := time.Now().Unix()
	grace := int64(licenseGracePeriod().Seconds())
	if lastOK <= 0 {
		// 从未验证成功:无保护,直接限制
		s.setState(SyncGraceExpired)
		return
	}
	deadline := lastOK + grace
	if now >= deadline {
		logLicenseSync("grace expired: server unreachable since %d, deadline %d -> restricted", lastOK, deadline)
		s.setState(SyncGraceExpired)
		return
	}
	s.state.Update(func(mm map[string]any) {
		mm["grace_deadline"] = deadline
	})
	logLicenseSync("server unreachable (%s): offline -> grace until %d", reason, deadline)
	s.setState(SyncGrace)
}

// ---------- SSE 客户端(自动重连 + Last-Event-ID) ----------

// sseLoop SSE 常驻连接循环:自动重连,指数退避(1s→60s 封顶)。
// Server 恢复的唯一正常触发机制:SSE Reconnect Success → V3 Verify。
func (s *LicenseSync) sseLoop() {
	backoffIdx := 0
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// 无有效凭据 → 等待(激活后由 wake 唤醒)
		_, token, _, deviceID, ok := loadLicenseCred(s.st)
		if !ok || token == "" || deviceID == "" {
			s.sseMu.Lock()
			s.credsValid = false
			s.sseMu.Unlock()
			select {
			case <-s.wake:
				backoffIdx = 0
				continue
			case <-s.ctx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}
		s.sseMu.Lock()
		s.credsValid = true
		wasDisconnected := s.disconnected
		s.sseMu.Unlock()

		// 建立连接并阻塞读取;返回 err = 连接结束(断线/Server 关闭)
		connErr := s.sseConnect(token, deviceID, wasDisconnected)
		if s.ctx.Err() != nil {
			return // 引擎停止
		}

		// SSE 401 = 凭据被服务端明确拒绝(吊销/解绑后 token 作废)→ 立即 Verify 权威状态。
		// 这不是周期检查:401 是服务端响应信号(类似 Event),Verify 会返回 revoked/invalid,
		// 避免客户端在 Grace 里"永久不知情"直到宽限过期(设计 §9:Server Down ≠ Revoked)。
		if he, ok := connErr.(*sseHTTPError); ok && he.Code == http.StatusUnauthorized {
			logLicenseSync("SSE auth rejected (401) -> V3 Verify authoritative state")
			s.triggerVerify("sse-unauthorized")
		}

		// 断线:标记 Offline → Grace
		s.markOffline("sse-disconnect")

		// 指数退避后重连(1s 2s 5s 10s 30s 60s 封顶)
		wait := sseBackoff[backoffIdx]
		if backoffIdx < len(sseBackoff)-1 {
			backoffIdx++
		}
		logLicenseSync("SSE reconnect in %v (err: %v)", wait, connErr)
		select {
		case <-s.wake:
			backoffIdx = 0
			continue
		case <-s.ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// sseHTTPError SSE 连接的服务端 HTTP 状态错误(用于 401 等明确信号识别)。
type sseHTTPError struct {
	Code int
}

func (e *sseHTTPError) Error() string {
	return fmt.Sprintf("SSE status %d", e.Code)
}

// sseConnect 建立一次 SSE 订阅并阻塞读取,直到连接断开。
// wasDisconnected=true 表示这是重连成功 → Server Recovery Signal → 立即 Verify。
func (s *LicenseSync) sseConnect(token, deviceID string, wasDisconnected bool) error {
	serverURL := serverURLOf(s.st)
	req, err := http.NewRequestWithContext(s.ctx, "GET", serverURL+licenseVerifyPath+"/events", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Device-ID", deviceID)
	req.Header.Set("Accept", "text/event-stream")
	s.sseMu.Lock()
	// 从本地恢复事件游标(重启/升级后 Last-Event-ID 不丢失)
	if s.lastEventID == "" {
		if m, ok := readLicenseStore(s.st); ok {
			s.lastEventID = strOr(m["last_event_id"])
			s.lastSeq = int64(float64(numOr(m["last_event_seq"])))
		}
	}
	if s.lastEventID != "" {
		req.Header.Set("Last-Event-ID", s.lastEventID)
	}
	s.sseMu.Unlock()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return &sseHTTPError{Code: resp.StatusCode}
	}
	defer resp.Body.Close()

	// 连接建立成功
	s.sseMu.Lock()
	s.everConnected = true
	s.sseMu.Unlock()
	if wasDisconnected {
		// SSE Reconnect Success = Server Recovery Signal → 立即 V3 Verify
		logLicenseSync("SSE reconnect success -> SERVER RECOVERED, verifying authoritative state")
		s.state.Update(func(m map[string]any) { m["sync_state"] = string(SyncRecovered) })
		s.triggerVerify("server-recovered")
	}

	// 读取事件流(阻塞;断线/超时 → 返回 err 触发重连)
	return s.sseRead(resp.Body)
}

// sseEvent 服务端推送的 License 事件。
type sseEvent struct {
	EventID      string         `json:"event_id"`
	SequenceID   int64          `json:"sequence_id"`
	EventType    string         `json:"event_type"`
	LicenseID    string         `json:"license_id"`
	ActivationID string         `json:"activation_id"`
	DeviceID     string         `json:"device_id"`
	StateVersion int64          `json:"state_version"`
	CreatedAt    int64          `json:"created_at"`
	Payload      map[string]any `json:"payload"`
}

// sseRead 逐行解析 SSE 流(事件块:event/id/data,空行分隔;注释行忽略)。
func (s *LicenseSync) sseRead(body io.Reader) error {
	reader := bufio.NewReader(body)
	var evName, evID string
	var dataParts []string
	dispatch := func() error {
		if evName == "" && evID == "" && len(dataParts) == 0 {
			return nil
		}
		data := strings.Join(dataParts, "\n")
		var ev sseEvent
		if data != "" {
			_ = json.Unmarshal([]byte(data), &ev)
		}
		if ev.EventType == "" {
			ev.EventType = evName
		}
		if ev.EventID == "" {
			ev.EventID = evID
		}
		evName, evID, dataParts = "", "", nil
		if ev.EventType == "resync_required" {
			logLicenseSync("SSE resync_required -> V3 Verify (replay cannot recover)")
			s.triggerVerify("resync-required")
			return nil
		}
		if ev.EventType != "" || ev.SequenceID > 0 {
			s.onLicenseEvent(&ev)
		}
		return nil
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			_ = dispatch() // 收尾残留块
			if s.ctx.Err() != nil {
				return s.ctx.Err()
			}
			return err // EOF/断线 → 重连
		}
		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "":
			if err := dispatch(); err != nil {
				return err
			}
		case strings.HasPrefix(line, ":"):
			// 注释行(: keep-alive)—— 仅连接保活,不是 Heartbeat,不触发 Verify
		case strings.HasPrefix(line, "event: "):
			evName = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "id: "):
			evID = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "data: "):
			dataParts = append(dataParts, strings.TrimPrefix(line, "data: "))
		}
	}
}

// onLicenseEvent 处理 License 事件(幂等 + 乱序保护,Event 只是 Trigger → Verify)。
func (s *LicenseSync) onLicenseEvent(ev *sseEvent) {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()

	// 幂等:sequence <= 已处理 → 丢弃(重复事件只处理一次,Test 8)
	if ev.SequenceID > 0 && ev.SequenceID <= s.lastSeq {
		return
	}
	// 乱序保护:事件版本 <= 本地权威版本 → 旧事件,忽略(Test 9)
	if ev.StateVersion > 0 && s.stateVersion > 0 && ev.StateVersion <= s.stateVersion {
		s.lastSeq = ev.SequenceID
		s.lastEventID = ev.EventID
		s.persistEventPosLocked()
		return
	}
	// Gap:事件序号跳变(中间事件丢失)→ 无法逐条补齐 → Verify 权威状态(Test 10)
	gap := s.lastSeq > 0 && ev.SequenceID > s.lastSeq+1
	if ev.SequenceID > 0 {
		s.lastSeq = ev.SequenceID
	}
	if ev.EventID != "" {
		s.lastEventID = ev.EventID
	}
	// 版本推进:事件版本更新即采用(Verify 会最终对齐权威值,防旧事件覆盖新状态)
	if ev.StateVersion > s.stateVersion {
		s.stateVersion = ev.StateVersion
	}
	s.persistEventPosLocked()

	logLicenseSync("license event: %s (seq=%d ver=%d)%s", ev.EventType, ev.SequenceID, ev.StateVersion, map[bool]string{true: " [GAP]"}[gap])

	// Event = Trigger:状态可能变化 → 交给 Server 权威 Verify(不直接改授权结论)
	if gap {
		s.triggerVerify("event-gap")
		return
	}
	s.triggerVerify("sse-event:" + ev.EventType)
}

// persistEventPosLocked 持久化事件游标(调用方必须持有 sseMu)。
func (s *LicenseSync) persistEventPosLocked() {
	evID := s.lastEventID
	seq := s.lastSeq
	if evID == "" && seq > 0 {
		evID = "evt_" + strconv.FormatInt(seq, 10)
	}
	s.state.Update(func(m map[string]any) {
		if evID != "" {
			m["last_event_id"] = evID
		}
		m["last_event_seq"] = seq
	})
}

// setState 写入 sync_state(带通知)。
func (s *LicenseSync) setState(st SyncState) {
	if s == nil || s.state == nil {
		return
	}
	s.state.Update(func(m map[string]any) {
		m["sync_state"] = string(st)
	})
}

// ---------- 凭据变化唤醒 ----------

// wakeSSE 激活/解绑后唤醒 SSE 循环(立即用新凭据建立连接)。
func (s *LicenseSync) wakeSSE() {
	if s == nil {
		return
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// logLicenseSync 同步引擎日志(带 [license-sync] 前缀,便于排障)。
func logLicenseSync(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	// 通过标准 logger 输出(与面板日志一致)
	stdLogPrintf("[license-sync] %s", msg)
}

// stdLogPrintf 输出到标准日志(保持与现有 logger 一致)。
var stdLogPrintf = func(format string, args ...any) {
	// 延迟绑定,避免 import cycle;main 包注入真实 logger
	licenseLogf(format, args...)
}

// licenseLogf 实际日志输出(log 包,可在测试中替换)。
var licenseLogf = func(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
