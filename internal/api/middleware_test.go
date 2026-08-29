package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/DockerManger/Docker_Manager_Go/internal/auth"
	"github.com/DockerManger/Docker_Manager_Go/internal/config"
	"github.com/DockerManger/Docker_Manager_Go/internal/settings"
	"github.com/DockerManger/Docker_Manager_Go/internal/state"
)

// TestAuthMiddlewarePCA SEC-003 中间件回归:
//   - pca 匹配 → 放行
//   - pca 落后(改密/2FA 变更后)→ 401
//   - 用户名不存在(改用户名后旧 token)→ 401(修复:u==nil 不再放行)
func TestAuthMiddlewarePCA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store, err := settings.Load(t.TempDir(), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	st := &state.AppState{
		Cfg:      &config.Config{JWTSecret: "s"},
		Settings: store,
		Users:    []state.StoredUser{{Username: "admin", PasswordHash: "", PasswordChangedAt: 100}},
	}
	r := gin.New()
	r.Use(AuthMiddleware(st))
	r.GET("/x", func(c *gin.Context) { c.Status(200) })

	do := func(token string) int {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	// 1. pca 匹配 → 200
	if code := do(auth.MakeToken("s", "admin", 3600, 100)); code != 200 {
		t.Fatalf("valid token = %d, want 200", code)
	}
	// 2. pca 落后(修改密码后旧 token)→ 401
	if code := do(auth.MakeToken("s", "admin", 3600, 99)); code != 401 {
		t.Fatalf("stale pca token = %d, want 401", code)
	}
	// 3. 用户名不存在(改用户名后旧 token)→ 401
	if code := do(auth.MakeToken("s", "oldadmin", 3600, 100)); code != 401 {
		t.Fatalf("renamed-user token = %d, want 401", code)
	}
	// 4. 签名错误 → 401
	if code := do("bad.token.here"); code != 401 {
		t.Fatalf("bad signature = %d, want 401", code)
	}
}
