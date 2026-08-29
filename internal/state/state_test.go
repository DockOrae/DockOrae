package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/auth"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/config"
)

// TestResetAdminPasswordIfMarked install.sh reset-passwd 配套机制(SQLite 迁移后):
// data 目录存在 .reset-admin-password 标记时,admin 密码重置为 123456 并强制修改,
// 旧 token 失效(pca 更新),标记删除。
func TestResetAdminPasswordIfMarked(t *testing.T) {
	dir := t.TempDir()
	hash, _ := auth.HashPassword("oldpass123")
	st := &AppState{
		Cfg:   &config.Config{DataDir: dir, JWTSecret: "s"},
		Users: []StoredUser{{Username: "admin", PasswordHash: hash}},
	}

	// 无标记 → 不重置
	if st.ResetAdminPasswordIfMarked() {
		t.Fatal("无标记不应重置")
	}
	if !auth.VerifyPassword("oldpass123", st.FindUser("admin").PasswordHash) {
		t.Fatal("无标记时密码不应变化")
	}

	// 有标记 → 重置
	marker := filepath.Join(dir, ".reset-admin-password")
	if err := os.WriteFile(marker, []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !st.ResetAdminPasswordIfMarked() {
		t.Fatal("有标记应执行重置")
	}
	u := st.FindUser("admin")
	if !auth.VerifyPassword("123456", u.PasswordHash) {
		t.Fatal("密码未重置为 123456")
	}
	if !u.MustChangePassword {
		t.Fatal("must_change_password 未置位")
	}
	if u.PasswordChangedAt == 0 {
		t.Fatal("password_changed_at 未更新(旧 token 不会失效)")
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("标记文件未删除")
	}
	// 重置后凭据已写入 users.json(降级路径)
	if _, err := os.Stat(filepath.Join(dir, "users.json")); err != nil {
		t.Fatalf("users.json 未写入: %v", err)
	}

	// admin 不存在时重建默认 admin
	dir2 := t.TempDir()
	st2 := &AppState{Cfg: &config.Config{DataDir: dir2, JWTSecret: "s"}}
	if err := os.WriteFile(filepath.Join(dir2, ".reset-admin-password"), []byte("1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !st2.ResetAdminPasswordIfMarked() {
		t.Fatal("应重建 admin")
	}
	if u2 := st2.FindUser("admin"); u2 == nil || !auth.VerifyPassword("123456", u2.PasswordHash) {
		t.Fatal("admin 未正确重建")
	}
}
