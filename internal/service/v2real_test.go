package service

import (
	"os"
	"strings"
	"testing"
)

// TestVerifyRealIssuedKey 用 License Server 真实签发的 Key 验证(手动集成测试)。
// 需要环境变量 DML_REAL_KEY(真实签发的 Key),未设置则跳过。
func TestVerifyRealIssuedKey(t *testing.T) {
	key := os.Getenv("DML_REAL_KEY")
	if key == "" {
		t.Skip("DML_REAL_KEY 未设置,跳过真实签发集成验证")
	}
	info, ok := LicenseVerifyKey(key)
	if !ok {
		t.Fatal("真实签发的 V2 Key 必须验证通过")
	}
	t.Logf("license_id=%v plan=%v features=%v status=%v",
		info["license_id"], info["plan"], info["features"], info["status"])
	if strings.Contains(key, "docker-manager-go-license-v1") {
		t.Fatal("V2 key 不应包含 V1 机密")
	}
}
