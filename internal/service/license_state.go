package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/DockOrae/DockOrae/internal/state"
)

// ---------- LicenseStateManager(V3 本地 License 状态唯一入口) ----------
//
// 职责(§35/§36):
//   - license.json 的唯一写入口:所有状态变更必须经过 Update(),
//     禁止散落多处直接修改文件
//   - 原子安全写入:临时文件 → fsync → rename(防止 Crash 导致 license.json 损坏)
//   - 变更通知:状态变化后通知订阅者(Vue 经 /ws/license 实时更新,无需刷新/轮询)
//
// 存储字段(扩展原 V3 字段):
//
//	key / device_id / activation_id / activation_token(绝不写日志)
//	last_successful_verify / verify_state / server_url / clock_offset 族
//	sync_state           在线同步状态:online/offline/grace/grace_expired/
//	                      server_recovered/revoked/blocked
//	last_event_id        最近处理的事件 ID(evt_N,SSE Replay 用)
//	state_version        Server 权威状态版本(事件乱序保护)
//	grace_deadline       宽限截止时间(Server 不可达时的有限保护)

// LicenseStateManager 本地 License 状态管理。
type LicenseStateManager struct {
	st   *state.AppState
	mu   sync.Mutex
	subs map[chan struct{}]struct{}
}

// NewLicenseStateManager 构造。
func NewLicenseStateManager(st *state.AppState) *LicenseStateManager {
	return &LicenseStateManager{st: st, subs: make(map[chan struct{}]struct{})}
}

// Get 读取当前状态(缓存于内存;失败时返回 nil)。
func (m *LicenseStateManager) Get() map[string]any {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readLocked()
}

// readLocked 读取(调用方必须持有 mu)。
func (m *LicenseStateManager) readLocked() map[string]any {
	raw, err := os.ReadFile(licensePath(m.st))
	if err != nil {
		return nil
	}
	var out map[string]any
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return out
}

// Update 读取 → 变更 → 原子写入 → 通知订阅者。任何状态修改都必须走这里。
func (m *LicenseStateManager) Update(mutate func(map[string]any)) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	cur := m.readLocked()
	if cur == nil {
		cur = map[string]any{}
	}
	if mutate != nil {
		mutate(cur)
	}
	if err := atomicWriteLicenseFile(m.st, cur); err != nil {
		return err
	}
	m.notifyLocked()
	return nil
}

// UpdateBytes 直接写入完整状态对象(激活时构造全新 store 用)。
func (m *LicenseStateManager) UpdateBytes(store map[string]any) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := atomicWriteLicenseFile(m.st, store); err != nil {
		return err
	}
	m.notifyLocked()
	return nil
}

// Subscribe 订阅状态变更通知(缓冲 channel,慢消费者丢弃)。
func (m *LicenseStateManager) Subscribe() chan struct{} {
	ch := make(chan struct{}, 8)
	m.mu.Lock()
	m.subs[ch] = struct{}{}
	m.mu.Unlock()
	return ch
}

// Unsubscribe 取消订阅。
func (m *LicenseStateManager) Unsubscribe(ch chan struct{}) {
	m.mu.Lock()
	delete(m.subs, ch)
	m.mu.Unlock()
}

// notifyLocked 通知全部订阅者(调用方必须持有 mu)。
func (m *LicenseStateManager) notifyLocked() {
	for ch := range m.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// atomicWriteLicenseFile 原子写 license.json:临时文件 → fsync → rename。
// 权限 0600;绝不在日志中输出内容。
//
// 并发安全:所有 LicenseStateManager 实例(引擎/激活/解绑)共用 licenseFileMu,
// 串行化"临时文件写入 + rename"整段 —— 否则两个实例并发写固定 .tmp 名会互相
// 覆盖/占用(Windows 上 rename 目标被占用直接失败,曾导致解绑返回 500)。
// 临时文件用唯一名(CreateTemp),rename 失败(目标被短暂占用)重试 3 次。
var licenseFileMu sync.Mutex

func atomicWriteLicenseFile(st *state.AppState, m map[string]any) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	path := licensePath(st)

	licenseFileMu.Lock()
	defer licenseFileMu.Unlock()

	tmp, err := os.CreateTemp(filepath.Dir(path), "license-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // rename 成功后无害;失败时清理

	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	// fsync:确保数据落盘后再 rename(防 Crash 后 tmp 未刷盘导致空/旧文件覆盖)
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// 目标被短暂占用(并发读/杀软扫描)时重试,避免偶发 500
	for i := 0; i < 3; i++ {
		if err := os.Rename(tmpName, path); err == nil {
			return nil
		} else if i == 2 {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return err
}
