package api

import (
	"testing"
	"time"
)

// TestLoginGuardLockUnlock 密码登录限速:5 次失败 → 锁定;锁定期间 check 返回剩余秒;
// 锁定过期后自动解锁并重新计数。
func TestLoginGuardLockUnlock(t *testing.T) {
	g := &loginGuard{keys: map[string]*loginFail{}}
	key := loginKey("1.2.3.4", "admin")

	// 前 4 次失败未锁定
	for i := 0; i < 4; i++ {
		if locked, _ := g.check(key); locked {
			t.Fatalf("第 %d 次失败不应锁定", i+1)
		}
		g.fail(key, 5, time.Minute)
	}
	if locked, _ := g.check(key); locked {
		t.Fatal("第 4 次失败后不应锁定")
	}
	// 第 5 次失败 → 锁定
	g.fail(key, 5, time.Minute)
	if locked, remain := g.check(key); !locked || remain <= 0 {
		t.Fatalf("第 5 次失败后应锁定(remain=%d)", remain)
	}
	// 锁定期间继续失败不改变状态
	g.fail(key, 5, time.Minute)
	if locked, _ := g.check(key); !locked {
		t.Fatal("锁定期间应保持锁定")
	}
	// 不同 key 不受影响
	if locked, _ := g.check(loginKey("5.6.7.8", "admin")); locked {
		t.Fatal("其他 IP 不应被锁定")
	}
}

// TestLoginGuardExpiry 锁定过期后自动解锁(check 清除过期记录)
func TestLoginGuardExpiry(t *testing.T) {
	g := &loginGuard{keys: map[string]*loginFail{}}
	key := loginKey("1.2.3.4", "admin")
	g.fail(key, 1, 30*time.Millisecond) // 立即锁定
	if locked, _ := g.check(key); !locked {
		t.Fatal("应处于锁定状态")
	}
	time.Sleep(50 * time.Millisecond)
	if locked, _ := g.check(key); locked {
		t.Fatal("锁定过期后应自动解锁")
	}
	// 过期条目已被清除:再次失败重新计数
	g.fail(key, 1, time.Minute)
	if locked, _ := g.check(key); !locked {
		t.Fatal("重新计数后应锁定")
	}
}

// TestLoginGuardSuccessClears 登录成功后清除失败记录
func TestLoginGuardSuccessClears(t *testing.T) {
	g := &loginGuard{keys: map[string]*loginFail{}}
	key := loginKey("1.2.3.4", "admin")
	for i := 0; i < 4; i++ {
		g.fail(key, 5, time.Minute)
	}
	g.success(key)
	if locked, _ := g.check(key); locked {
		t.Fatal("success 后不应再锁定")
	}
	// 清除后重新从 1 计数
	g.fail(key, 5, time.Minute)
	if locked, _ := g.check(key); locked {
		t.Fatal("清除后第一次失败不应锁定")
	}
}

// TestLoginGuardTOTP TOTP 阈值(3 次)
func TestLoginGuardTOTP(t *testing.T) {
	g := &loginGuard{keys: map[string]*loginFail{}}
	key := loginKey("1.2.3.4", "admin")
	g.fail(key, totpMaxFails, totpLockTime)
	g.fail(key, totpMaxFails, totpLockTime)
	if locked, _ := g.check(key); locked {
		t.Fatal("第 2 次 TOTP 失败不应锁定")
	}
	g.fail(key, totpMaxFails, totpLockTime)
	if locked, _ := g.check(key); !locked {
		t.Fatal("第 3 次 TOTP 失败应锁定")
	}
}
