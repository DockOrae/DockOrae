package api

import (
	"fmt"
	"sync"
	"time"

	"github.com/DockerManger/Docker_Manager_Go/internal/service"
)

// ---------- 登录限速(SEC-002) ----------
//
// 轻量内存限速:IP+用户名 维度记录失败次数,达到阈值后锁定一段时间。
// 不引入 Redis/DB;单进程面板场景下内存状态足够。
// 注意:经反代部署时 ClientIP 取自 X-Forwarded-For,攻击者可换 IP 绕过
// (面板类工具的固有取舍);配合域名白名单/TLS 使用效果最佳。

const (
	loginMaxFails = 5                // 密码登录:5 次失败
	loginLockTime = 15 * time.Minute // 锁定 15 分钟
	totpMaxFails  = 3                // TOTP:3 次失败
	totpLockTime  = 5 * time.Minute  // 锁定 5 分钟
	loginGuardMax = 10000            // map 上限,防伪造 IP 撑爆内存
)

type loginFail struct {
	count       int
	lockedUntil time.Time
}

type loginGuard struct {
	mu   sync.Mutex
	keys map[string]*loginFail
}

var loginGuardInst = &loginGuard{keys: map[string]*loginFail{}}

// key IP|username
func loginKey(ip, username string) string {
	return ip + "|" + username
}

// check 返回是否被锁定及剩余秒数(未锁定返回 false, 0)
func (g *loginGuard) check(key string) (bool, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	f, ok := g.keys[key]
	if !ok {
		return false, 0
	}
	// 从未锁定过(lockedUntil 零值)或锁定已过期 → 不阻塞
	if f.lockedUntil.IsZero() {
		return false, 0
	}
	if f.lockedUntil.After(time.Now()) {
		return true, int(time.Until(f.lockedUntil).Seconds()) + 1
	}
	// 锁定已过期:清除记录,重新开始计数
	delete(g.keys, key)
	return false, 0
}

// fail 记录一次失败;达到阈值时锁定并返回 true
func (g *loginGuard) fail(key string, threshold int, lockFor time.Duration) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.gcLocked()
	f, ok := g.keys[key]
	if !ok {
		f = &loginFail{}
		g.keys[key] = f
	}
	f.count++
	if f.count >= threshold {
		f.lockedUntil = time.Now().Add(lockFor)
		return true
	}
	return false
}

// success 登录成功:清除该 key 的失败记录
func (g *loginGuard) success(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.keys, key)
}

// gcLocked 记录数超上限时,清掉所有未锁定条目(防伪造 IP 撑爆内存;调用方持锁)
func (g *loginGuard) gcLocked() {
	if len(g.keys) < loginGuardMax {
		return
	}
	now := time.Now()
	for k, f := range g.keys {
		if !f.lockedUntil.After(now) {
			delete(g.keys, k)
		}
	}
}

// loginThrottled 检查限速;被锁定时返回 429 错误
func loginThrottled(ip, username string) error {
	locked, remain := loginGuardInst.check(loginKey(ip, username))
	if locked {
		return service.NewApiError(429, fmt.Sprintf("登录失败次数过多,请在 %d 秒后重试", remain))
	}
	return nil
}

func totpThrottled(ip, username string) error {
	locked, remain := loginGuardInst.check(loginKey(ip, username))
	if locked {
		return service.NewApiError(429, fmt.Sprintf("验证码错误次数过多,请在 %d 秒后重试", remain))
	}
	return nil
}
