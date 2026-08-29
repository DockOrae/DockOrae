package service

import (
	"testing"

	"github.com/DockerManger/Docker_Manager_Go/internal/auth"
	"github.com/DockerManger/Docker_Manager_Go/internal/config"
	"github.com/DockerManger/Docker_Manager_Go/internal/settings"
	"github.com/DockerManger/Docker_Manager_Go/internal/state"
)

// TestPasswordChangeInvalidatesOldToken SEC-003 回归测试:
// 登录拿 token → 修改密码 → 旧 token 的 pca 落后于用户当前值(中间件会拒绝)
// → 新密码登录的新 token 正常工作。
func TestPasswordChangeInvalidatesOldToken(t *testing.T) {
	store, err := settings.Load(t.TempDir(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	hash, err := auth.HashPassword("oldpass123")
	if err != nil {
		t.Fatal(err)
	}
	st := &state.AppState{
		Cfg:      &config.Config{JWTSecret: "test-secret", DataDir: t.TempDir()},
		Settings: store,
		Users:    []state.StoredUser{{Username: "admin", PasswordHash: hash}},
	}

	// 1. 旧密码登录 → 旧 token
	resp, err := Login(st, "admin", "oldpass123", "127.0.0.1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	oldToken, _ := resp["token"].(string)
	if oldToken == "" {
		t.Fatal("old token empty")
	}
	_, oldPCA, ok := auth.VerifyToken(st.Cfg.JWTSecret, oldToken)
	if !ok {
		t.Fatal("old token invalid")
	}

	// 2. 修改密码 → pca 递增
	if err := ChangePassword(st, "admin", "oldpass123", "newpass456"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	u := st.FindUser("admin")
	if u == nil || u.PasswordChangedAt == 0 {
		t.Fatal("PasswordChangedAt not updated")
	}
	if u.PasswordChangedAt == oldPCA {
		t.Fatal("PasswordChangedAt 未变化,旧 token 不会被拒绝")
	}

	// 3. 旧密码登录必须失败;旧 token 的 pca 必须与用户当前 pca 不一致(中间件据此拒绝)
	if _, err := Login(st, "admin", "oldpass123", "127.0.0.1"); err == nil {
		t.Fatal("old password should fail")
	}
	if u.PasswordChangedAt == oldPCA {
		t.Fatal("pca should differ")
	}

	// 4. 新密码登录 → 新 token 的 pca 与用户当前一致(中间件放行)
	resp2, err := Login(st, "admin", "newpass456", "127.0.0.1")
	if err != nil {
		t.Fatalf("new password login: %v", err)
	}
	newToken, _ := resp2["token"].(string)
	_, newPCA, ok := auth.VerifyToken(st.Cfg.JWTSecret, newToken)
	if !ok {
		t.Fatal("new token invalid")
	}
	if newPCA != st.FindUser("admin").PasswordChangedAt {
		t.Fatal("new token pca mismatch")
	}
}

// TestLoginTokenCarriesPCA 登录签发的 token 必须携带用户当前 pca(中间件校验的前置条件)。
func TestLoginTokenCarriesPCA(t *testing.T) {
	store, err := settings.Load(t.TempDir(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	hash, _ := auth.HashPassword("pass123456")
	st := &state.AppState{
		Cfg:      &config.Config{JWTSecret: "test-secret", DataDir: t.TempDir()},
		Settings: store,
		Users:    []state.StoredUser{{Username: "admin", PasswordHash: hash, PasswordChangedAt: 777}},
	}
	resp, err := Login(st, "admin", "pass123456", "127.0.0.1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	_, pca, ok := auth.VerifyToken(st.Cfg.JWTSecret, resp["token"].(string))
	if !ok || pca != 777 {
		t.Fatalf("token pca = %d (ok=%v), want 777", pca, ok)
	}
}
